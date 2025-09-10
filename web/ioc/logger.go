package ioc

import (
	"short_url/pkg/logfile"

	"github.com/spf13/viper"
	"github.com/to404hanga/pkg404/logger"
	"go.uber.org/zap"
)

func InitLogger() logger.Logger {
	var cfg zap.Config
	mode := viper.GetString("log.mode")
	switch mode {
	case "dev", "test":
		cfg = zap.NewDevelopmentConfig()
	case "prod":
		cfg = zap.NewProductionConfig()
	default:
		panic("invalid log mode")
	}

	// 将输出路径加入到 zap 配置中
	outputPaths := viper.GetStringSlice("log.outputPaths")
	errorOutputPaths := viper.GetStringSlice("log.errorOutputPaths")
	logfile.InitLogFilePath(outputPaths...)
	logfile.InitLogFilePath(errorOutputPaths...)
	cfg.OutputPaths = append(cfg.OutputPaths, outputPaths...)
	cfg.ErrorOutputPaths = append(cfg.ErrorOutputPaths, errorOutputPaths...)

	l, err := cfg.Build(
		zap.AddStacktrace(zap.ErrorLevel), // Error 级别日志携带 stack trace
		zap.AddCallerSkip(1),
	)
	if err != nil {
		panic(err)
	}
	return logger.NewZapLogger(l)
}
