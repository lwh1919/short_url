package ioc

import (
	grpc2 "short_url/rpc/grpc"

	"github.com/spf13/viper"
	"github.com/to404hanga/pkg404/grpcx"
	"github.com/to404hanga/pkg404/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"google.golang.org/grpc"
)

func InitGrpcxServer(shortUrl *grpc2.ShortUrlServiceServer, ecli *clientv3.Client, l logger.Logger) *grpcx.Server {
	type Config struct {
		Port     int    `yaml:"port"`
		EtcdAddr string `yaml:"etcdAddr"`
		EtcdTTL  int64  `yaml:"etcdTTL"`
	}
	var cfg Config
	if err := viper.UnmarshalKey("grpc.server", &cfg); err != nil {
		panic(err)
	}
	server := grpc.NewServer()
	shortUrl.Register(server)
	return &grpcx.Server{
		Server:     server,
		Port:       cfg.Port,
		EtcdClient: ecli,
		Name:       "short_url",
		EtcdTTL:    cfg.EtcdTTL,
		L:          l,
	}
}
