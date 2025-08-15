//go:build wireinject

package main

import (
	"short_url_rpc_study/web/ioc"
	"short_url_rpc_study/web/routes"

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
