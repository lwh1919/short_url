package ioc

import (
	"github.com/spf13/viper"
	short_url_v1 "short_url_rpc_study/proto/short_url/v1"
	"short_url_rpc_study/web/routes"
)

func InitServerHandler(svc short_url_v1.ShortUrlServiceClient) *routes.ServerHandler {
	weights := viper.GetIntSlice("short_url.weights")

	return routes.NewServerHandler(svc, weights)
}
