package ioc

import (
	"github.com/afex/hystrix-go/hystrix"
	"github.com/spf13/viper"
)

func InitHystrix() string {
	var cfg hystrix.CommandConfig
	if err := viper.UnmarshalKey("hystrix", &cfg); err != nil {
		panic(err)
	}

	hystrix.ConfigureCommand("short_url", cfg)
	return "short_url"
}
