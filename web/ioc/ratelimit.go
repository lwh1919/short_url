package ioc

import (
	"context"
	"fmt"
	go_redis_tokenbucket "short_url_rpc_study/pkg/go-redis-tokenbuket"
	"short_url_rpc_study/web/pkg"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
)

// 使用接口包中的RateLimiter接口
type RateLimiter = pkg.RateLimiter

// tokenBucketLimiter 基于令牌桶的限流器实现
type tokenBucketLimiter struct {
	limiter *go_redis_tokenbucket.TokenBucketLimiter
	config  RateLimitConfig
}

// RateLimitConfig 限流配置
type RateLimitConfig struct {
	Rate     time.Duration `yaml:"rate"`     // 令牌生成速率
	Capacity int64         `yaml:"capacity"` // 桶容量
	Expire   time.Duration `yaml:"expire"`   // key过期时间
	Prefix   string        `yaml:"prefix"`   // key前缀
}

// InitRateLimiter 初始化限流器
func InitRateLimiter(cmd redis.Cmdable) (RateLimiter, error) {
	var config RateLimitConfig
	if err := viper.UnmarshalKey("rate_limit", &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rate limit config: %w", err)
	}

	// 设置默认值
	if config.Rate <= 0 {
		config.Rate = 100 * time.Millisecond // 默认100ms一个令牌，即10 QPS
	}
	if config.Capacity <= 0 {
		config.Capacity = 100 // 默认100个容量
	}
	if config.Expire <= 0 {
		config.Expire = 10 * time.Minute // 默认10分钟过期
	}
	if config.Prefix == "" {
		config.Prefix = "rate_limit" // 默认前缀
	}

	// 创建限流器
	// 确保使用 *redis.Client 类型
	client, ok := cmd.(*redis.Client)
	if !ok {
		return nil, fmt.Errorf("redis client must be *redis.Client type")
	}

	limiter, err := go_redis_tokenbucket.NewTokenBucketLimiter(client, config.Rate, config.Capacity)
	if err != nil {
		return nil, fmt.Errorf("failed to create token bucket limiter: %w", err)
	}

	// 设置自定义过期时间
	limiter.SetExpiration(config.Expire)

	return &tokenBucketLimiter{
		limiter: limiter,
		config:  config,
	}, nil
}

// Allow 检查是否允许请求
func (l *tokenBucketLimiter) Allow(ctx context.Context, key string, n int64) (bool, error) {
	fullKey := fmt.Sprintf("%s:%s", l.config.Prefix, key)
	return l.limiter.Allow(ctx, fullKey, n)
}

// GetStats 获取限流器统计信息
func (l *tokenBucketLimiter) GetStats() map[string]interface{} {
	stats := l.limiter.Stats()
	stats["prefix"] = l.config.Prefix
	return stats
}
