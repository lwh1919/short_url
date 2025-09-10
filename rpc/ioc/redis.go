package ioc

import (
	"fmt"
	"short_url/pkg/bloom"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

func InitRedis() redis.Cmdable {
	type Config struct {
		Host         string `yaml:"host"`
		Port         int    `yaml:"port"`
		PoolSize     int    `yaml:"poolSize"`
		MinIdleConns int    `yaml:"minIdleConns"`
		MaxIdleConns int    `yaml:"maxIdleConns"`
		DialTimeout  int    `yaml:"dialTimeout"`
		ReadTimeout  int    `yaml:"readTimeout"`
		WriteTimeout int    `yaml:"writeTimeout"`
	}
	cfg := Config{}
	if err := viper.UnmarshalKey("redis", &cfg); err != nil {
		panic(err)
	}

	cmd := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		MaxIdleConns: cfg.MaxIdleConns,
		DialTimeout:  time.Duration(cfg.DialTimeout) * time.Millisecond,
		ReadTimeout:  time.Duration(cfg.ReadTimeout) * time.Millisecond,
		WriteTimeout: time.Duration(cfg.WriteTimeout) * time.Millisecond,
	})
	return cmd
}

// InitBloomFilter 初始化布隆过滤器
func InitBloomFilter(cmd redis.Cmdable) *bloom.BloomService {
	type Config struct {
		M int32 `yaml:"m"`
		K int32 `yaml:"k"`
	}

	cfg := Config{}
	if err := viper.UnmarshalKey("bloom_filter", &cfg); err != nil {
		panic(fmt.Errorf("failed to unmarshal bloom filter config: %w", err))
	}

	// 使用配置文件中的参数，如果没有配置则使用默认值
	if cfg.M <= 0 {
		cfg.M = 1000000 // 默认位数组大小，约100万个元素
	}
	if cfg.K <= 0 {
		cfg.K = 7 // 默认哈希函数数量
	}

	// 安全地获取Redis客户端
	var redisClient *bloom.RedisClient
	switch client := cmd.(type) {
	case *redis.Client:
		redisClient = bloom.NewRedisClient(client)
	default:
		panic(fmt.Errorf("unsupported redis client type: %T", cmd))
	}

	// 创建默认加密器
	encryptor := bloom.NewDefaultEncryptor()

	// 创建布隆过滤器服务
	bloomService, err := bloom.NewBloomService(cfg.M, cfg.K, redisClient, encryptor)
	if err != nil {
		panic(fmt.Errorf("failed to create bloom filter service: %w", err))
	}

	return bloomService
}
