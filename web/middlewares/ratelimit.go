package middlewares

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"short_url_rpc_study/web/pkg"
)

// RateLimitConfig 限流中间件配置
type RateLimitConfig struct {
	// KeyGenerator 生成限流key的函数
	KeyGenerator func(c *gin.Context) string
	// Limit 每次请求消耗的令牌数
	Limit int64
	// ErrorHandler 自定义错误处理
	ErrorHandler func(c *gin.Context, err error)
	// Skipper 跳过限流的条件
	Skipper func(c *gin.Context) bool
}

// NewRateLimiter 创建限流中间件
func NewRateLimiter(limiter pkg.RateLimiter, config RateLimitConfig) gin.HandlerFunc {
	// 设置默认值
	if config.KeyGenerator == nil {
		config.KeyGenerator = defaultKeyGenerator
	}
	if config.Limit <= 0 {
		config.Limit = 1
	}
	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultErrorHandler
	}
	if config.Skipper == nil {
		config.Skipper = defaultSkipper
	}

	return func(c *gin.Context) {
		// 检查是否跳过限流
		if config.Skipper(c) {
			c.Next()
			return
		}

		// 生成限流key
		key := config.KeyGenerator(c)
		if key == "" {
			c.Next()
			return
		}

		// 检查限流
		allowed, err := limiter.Allow(c.Request.Context(), key, config.Limit)
		if err != nil {
			config.ErrorHandler(c, err)
			return
		}

		if !allowed {
			c.Header("X-RateLimit-Remaining", "0")
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error": "请求过于频繁，请稍后再试",
				"code":  429,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// defaultKeyGenerator 默认key生成器（全局限流）
func defaultKeyGenerator(c *gin.Context) string {
	// 返回固定key，实现全局限流
	return "global"
}

// defaultErrorHandler 默认错误处理
func defaultErrorHandler(c *gin.Context, err error) {
	c.JSON(http.StatusInternalServerError, gin.H{
		"error": "限流器内部错误",
		"code":  500,
	})
	c.Abort()
}

// defaultSkipper 默认跳过条件
func defaultSkipper(c *gin.Context) bool {
	// 跳过健康检查等接口
	return c.Request.URL.Path == "/health" ||
		c.Request.URL.Path == "/api/health" ||
		c.Request.Method == "OPTIONS"
}

// NewGlobalRateLimiter 全局限流器（控制总并发量）
// qps 参数仅用于函数命名，实际限流速率由 RateLimiter 实现配置决定
// 每个请求只消耗1个令牌，这是正确的行为
func NewGlobalRateLimiter(limiter pkg.RateLimiter, qps int64) gin.HandlerFunc {
	return NewRateLimiter(limiter, RateLimitConfig{
		KeyGenerator: func(c *gin.Context) string {
			return "global"
		},
		Limit: 1, // 修复：每个请求只消耗1个令牌，而不是消耗qps个令牌
	})
}
