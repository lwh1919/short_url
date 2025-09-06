package ioc

import (
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// InitRedis 初始化Redis客户端，与RPC层使用相同配置结构
func InitRedis() redis.Cmdable {
	type Config struct {
		Host         string `yaml:"host"`
		Port         int    `yaml:"port"`
		Prefix       string `yaml:"prefix"`
		PoolSize     int    `yaml:"poolSize"`
		MinIdleConns int    `yaml:"minIdleConns"`
		MaxIdleConns int    `yaml:"maxIdleConns"`
		DialTimeout  int    `yaml:"dialTimeout"`
		ReadTimeout  int    `yaml:"readTimeout"`
		WriteTimeout int    `yaml:"writeTimeout"`
	}

	var cfg Config
	if err := viper.UnmarshalKey("redis", &cfg); err != nil {
		panic(fmt.Errorf("failed to unmarshal redis config: %w", err))
	}

	// 设置默认值
	if cfg.Host == "" {
		cfg.Host = "localhost"
	}
	if cfg.Port == 0 {
		cfg.Port = 6379
	}
	if cfg.PoolSize == 0 {
		cfg.PoolSize = 1000
	}
	if cfg.MinIdleConns == 0 {
		cfg.MinIdleConns = 100
	}
	if cfg.MaxIdleConns == 0 {
		cfg.MaxIdleConns = 500
	}
	if cfg.DialTimeout == 0 {
		cfg.DialTimeout = 5000
	}
	if cfg.ReadTimeout == 0 {
		cfg.ReadTimeout = 2000
	}
	if cfg.WriteTimeout == 0 {
		cfg.WriteTimeout = 2000
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxIdleConns: cfg.MaxIdleConns,
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Millisecond,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Millisecond,
	})

	return client
}
