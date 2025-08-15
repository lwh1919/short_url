package main

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"path/filepath"
)

func main() {
	initViperWatch()
	app := Init()
	err := app.GrpcServer.Serve()
	if err != nil {
		panic(err)
	}
}

const projectRoot = "C:/Users/linweihao/Desktop/short_url_rpc_study/rpc"

func initViperWatch() {
	cfile := pflag.String("config",
		filepath.Join(projectRoot, "config", "config.template.yaml"),
		"配置文件路径")
	pflag.Parse()
	// 直接指定文件路径
	viper.SetConfigFile(*cfile)
	viper.WatchConfig()
	err := viper.ReadInConfig()
	if err != nil {
		panic(err)
	}
}
