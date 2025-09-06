package pkg

import (
	"context"
)

// RateLimiter 限流器接口
type RateLimiter interface {
	Allow(ctx context.Context, key string, n int64) (bool, error)
	GetStats() map[string]interface{}
}
