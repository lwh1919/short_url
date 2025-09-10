package ioc

import (
	"short_url/rpc/repository/cache"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedisCache(cmd redis.Cmdable) cache.ShortUrlCache {
	type Config struct {
		Prefix     string `yaml:"prefix"`
		Expiration int    `yaml:"expiration"`
	}
	cfg := &Config{
		Prefix:     "short_url",
		Expiration: 86400,
	}
	if err := viper.UnmarshalKey("redis", &cfg); err != nil {
		panic(err)
	}

	expiration := time.Duration(cfg.Expiration) * time.Second
	return cache.NewRedisShortUrlCache(cmd, cfg.Prefix, expiration)
}
