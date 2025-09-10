package ioc

import (
	short_url_v1 "short_url/proto/short_url/v1"

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
	return short_url_v1.NewShortUrlServiceClient(cc)
}
