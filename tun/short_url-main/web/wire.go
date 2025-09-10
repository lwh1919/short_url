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
		ioc.InitShortUrlClient,
		ioc.InitEtcdClient,
		ioc.InitLogger,

		routes.NewApiHandler,
		ioc.InitServerHandler,

		ioc.InitGinMiddleware,
		ioc.InitWebServer,
	)
	return gin.Default()
}
