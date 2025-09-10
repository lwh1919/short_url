//go:build wireinject

package main

import (
	"short_url/web/ioc"
	"short_url/web/routes"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

func Init() *gin.Engine {
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
