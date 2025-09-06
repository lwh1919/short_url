package main

import (
	"path/filepath"

	"github.com/gin-gonic/gin"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// App 应用结构体
type App struct {
	Engine *gin.Engine
}

func main() {
	initViperWatch()
	app, err := Init()
	if err != nil {
		panic(err)
	}
	if err := app.Engine.Run(viper.GetString("app.addr")); err != nil {
		panic(err)
	}
}

const projectRoot = "C:/Users/linweihao/Desktop/demo/st/short_url_rpc_study/web"

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
