package ioc

import (
	"short_url/rpc/repository"
	"short_url/rpc/service"

	"github.com/spf13/viper"
	"github.com/to404hanga/pkg404/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func InitService(ecli *clientv3.Client, repo repository.ShortUrlRepository, l logger.Logger) service.ShortUrlService {
	type Config struct {
		Suffix string `yaml:"suffix"`
	}
	cfg := &Config{
		Suffix: "_TO404HANGA",
	}
	if err := viper.UnmarshalKey("short_url", &cfg); err != nil {
		panic(err)
	}

	// // 获取 etcd 键值对 "weights"，并将其转换为 weights 切片
	// ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	// defer cancel()
	// resp, err := ecli.Get(ctx, "weights")
	// if err != nil {
	// 	panic(err)
	// }
	// kv := resp.Kvs[0]
	// weightStr := string(kv.Value)
	// weights := make([]int, 0, 6)
	// for _, w := range strings.Split(weightStr, ",") {
	// 	if i, err := strconv.Atoi(w); err == nil {
	// 		weights = append(weights, i)
	// 	}
	// }
	weights := viper.GetIntSlice("short_url.weights")
	svc := service.NewCachedShortUrlService(repo, l, cfg.Suffix, weights)

	// // 监听 etcd 键值对的变化并更新 weights
	// go func() {
	// 	watchChan := ecli.Watch(context.Background(), "weights")
	// 	for resp := range watchChan {
	// 		for _, ev := range resp.Events {
	// 			if string(ev.Kv.Key) == "weights" {
	// 				weightStr = string(ev.Kv.Value)
	// 				weights := make([]int, 0, 6)
	// 				for _, w := range strings.Split(weightStr, ",") {
	// 					if i, err := strconv.Atoi(w); err == nil {
	// 						weights = append(weights, i)
	// 					}
	// 				}
	// 				svc.Weights = weights
	// 			}
	// 		}
	// 	}
	// }()

	return svc
}
