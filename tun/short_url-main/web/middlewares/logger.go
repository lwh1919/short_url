package middlewares

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/to404hanga/pkg404/logger"
)

func ZapLogger(l logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		start := time.Now()
		// 处理请求
		c.Next()
		duration := time.Since(start).Milliseconds()

		// 记录日志
		l.Info("GIN",
			logger.TimeString(start),
			logger.Int("status", c.Writer.Status()),
			logger.Int64("duration", duration),
			logger.String("ip", c.ClientIP()),
			logger.String("method", c.Request.Method),
			logger.String("path", path),
			logger.String("query", query),
		)
	}
}
