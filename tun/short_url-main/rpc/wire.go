//go:build wireinject

package main

import (
	"short_url/rpc/grpc"
	"short_url/rpc/ioc"
	"short_url/rpc/repository/dao"

	"github.com/google/wire"
)

func Init() *App {
	wire.Build(
		ioc.InitDB,
		ioc.InitLogger,
		ioc.InitRedis,
		ioc.InitEtcdClient,

		dao.NewGormShortUrlDAO,

		ioc.InitRedisCache,
		ioc.InitCachedRepository,
		ioc.InitService,
		grpc.NewShortUrlServiceServer,

		ioc.InitCleanerJob,
		ioc.InitJobs,
		ioc.InitGrpcxServer,

		wire.Struct(new(App), "*"),
	)
	return new(App)
}
