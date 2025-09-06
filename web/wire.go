//go:build wireinject

package main

import (
	"github.com/google/wire"
	"short_url_rpc_study/web/ioc"
	"short_url_rpc_study/web/routes"
)

func Init() (*App, error) {
	wire.Build(
		ioc.InitLogger,
		ioc.InitHystrix,
		ioc.InitRedis,
		ioc.InitRateLimiter,
		ioc.InitEtcdClient,
		ioc.InitShortUrlClient,
		ioc.InitServerHandler,
		ioc.InitGinMiddleware,
		ioc.InitWebServer,
		routes.NewApiHandler,
		routes.NewHealthHandler,
		wire.Struct(new(App), "*"),
	)
	return nil, nil
}
