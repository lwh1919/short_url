package routes

import (
	"fmt"
	"github.com/afex/hystrix-go/hystrix"
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
	"log"
	"net/http"
	"short_url/pkg/generator"
	short_url_v1 "short_url/proto/short_url/v1"
	"time"
)

type ServerHandler struct {
	svc          short_url_v1.ShortUrlServiceClient
	weights      []int
	requestGroup singleflight.Group
}

var _ Handler = (*ServerHandler)(nil)

func NewServerHandler(svc short_url_v1.ShortUrlServiceClient, weights []int) *ServerHandler {
	return &ServerHandler{
		svc:          svc,
		weights:      weights,
		requestGroup: singleflight.Group{},
	}
}

func (sh *ServerHandler) RegisterRoutes(srv *gin.Engine) {
	srv.GET("/:short_url", sh.Redirect)
}

func (h *ServerHandler) Redirect(ctx *gin.Context) {
	shortUrl := ctx.Param("short_url")
	if ok := generator.CheckShortUrl(shortUrl, h.weights); !ok {
		ctx.JSON(404, gin.H{"error": "Short URL not found"})
		return
	}
	// 使用带降级的熔断器保护RPC调用
	err := hystrix.Do("short_url",
		func() error {
			// 请求合并层 (防缓存击穿)
			result, err, _ := h.requestGroup.Do(shortUrl, func() (interface{}, error) {
				resp, err := h.svc.GetOriginUrl(ctx, &short_url_v1.GetOriginUrlRequest{
					ShortUrl: shortUrl,
				})
				if err != nil {
					return "", err
				}
				return resp.GetOriginUrl(), nil
			})

			if err != nil {
				return err
			}

			// 重定向到原始URL
			ctx.Redirect(301, result.(string))
			return nil
		},
		// 降级处理逻辑
		func(err error) error {
			// 记录降级日志
			log.Printf("[ServerHandler] Fallback triggered for short URL: %s And err: %s", shortUrl, err.Error())
			// 重定向到维护页面
			ctx.Redirect(302, "/static/maintenance.html")
			return nil
		})

	if err != nil {
		// 使用状态码判断错误类型
		var msg string
		switch err {
		case hystrix.ErrCircuitOpen:
			// 熔断器已打开错误
			msg = fmt.Sprintf("Circuit open error:", err.Error())
		case hystrix.ErrMaxConcurrency:
			// 超过最大并发数错误
			msg = fmt.Sprintf("Max concurrency error:", err.Error())
		default:
			msg = fmt.Sprintf("Other error:", err.Error())
		}
		// 记录错误日志
		log.Printf("[ServerHandler] RPC error [code=%s]: %v", msg)
		ctx.JSON(http.StatusNotFound, gin.H{
			"error":     msg,
			"timestamp": time.Now().Unix(),
		})
	}
}
