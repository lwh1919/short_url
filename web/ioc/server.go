package ioc

import (
	short_url_v1 "short_url/proto/short_url/v1"
	"short_url/web/routes"

	"github.com/spf13/viper"
)

func InitServerHandler(svc short_url_v1.ShortUrlServiceClient) *routes.ServerHandler {
	weights := viper.GetIntSlice("short_url.weights")

	return routes.NewServerHandler(svc, weights)
}
