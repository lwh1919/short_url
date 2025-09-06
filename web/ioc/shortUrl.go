package ioc

import (
	sentinelBreaker "github.com/alibaba/sentinel-golang/core/circuitbreaker"
	short_url_v1 "short_url_rpc_study/proto/short_url/v1"

	"github.com/spf13/viper"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/naming/resolver"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func InitShortUrlClient(ecli *clientv3.Client) short_url_v1.ShortUrlServiceClient {
	type Config struct {
		Target string `yaml:"target"`
		Secure bool   `yaml:"secure"`
	}
	var cfg Config
	err := viper.UnmarshalKey("grpc.client.shortUrl", &cfg)
	if err != nil {
		panic(err)
	}
	rs, err := resolver.NewBuilder(ecli)
	if err != nil {
		panic(err)
	}
	opts := []grpc.DialOption{
		grpc.WithResolvers(rs),
		grpc.WithDefaultServiceConfig(`{"loadBalancingConfig": [{"round_robin": {}}]}`),
	}
	if !cfg.Secure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}
	cc, err := grpc.NewClient(cfg.Target, opts...)
	if err != nil {
		panic(err)
	}

	// 读取熔断配置
	var cbRule struct {
		Resource         string  `yaml:"resource"`
		Strategy         int     `yaml:"strategy"`
		Threshold        float64 `yaml:"threshold"`
		MinRequestAmount uint64  `yaml:"minRequestAmount"`
		StatIntervalMs   uint32  `yaml:"statIntervalMs"`
		RetryTimeoutMs   uint32  `yaml:"retryTimeoutMs"`
		MaxAllowedRtMs   uint32  `yaml:"maxAllowedRtMs"`
	}
	err = viper.UnmarshalKey("pkg.shortUrlService", &cbRule)
	if err != nil {
		panic(err)
	}

	// 验证配置
	if cbRule.Threshold < 0 || cbRule.Threshold > 1 {
		cbRule.Threshold = 0.5 // 默认值
	}
	if cbRule.MinRequestAmount < 1 {
		cbRule.MinRequestAmount = 10 // 默认值
	}
	if cbRule.StatIntervalMs < 1000 {
		cbRule.StatIntervalMs = 10000 // 默认值10秒
	}
	if cbRule.RetryTimeoutMs < 100 {
		cbRule.RetryTimeoutMs = 3000 // 默认值3秒
	}
	if cbRule.MaxAllowedRtMs < 100 {
		cbRule.MaxAllowedRtMs = 1000 // 默认值1秒
	}

	// 策略映射
	strategy := sentinelBreaker.ErrorRatio // 默认策略
	if cbRule.Strategy == 1 {              // SlowRequestRatio
		strategy = sentinelBreaker.SlowRequestRatio
	} else if cbRule.Strategy == 2 { // ErrorCount
		strategy = sentinelBreaker.ErrorCount
	}

	_, err = sentinelBreaker.LoadRules([]*sentinelBreaker.Rule{
		{
			Resource:         cbRule.Resource,
			Strategy:         strategy,
			RetryTimeoutMs:   cbRule.RetryTimeoutMs,
			MinRequestAmount: cbRule.MinRequestAmount,
			StatIntervalMs:   cbRule.StatIntervalMs,
			Threshold:        cbRule.Threshold,
			MaxAllowedRtMs:   uint64(cbRule.MaxAllowedRtMs),
		},
	})
	if err != nil {
		panic(err)
	}

	return short_url_v1.NewShortUrlServiceClient(cc)
}
