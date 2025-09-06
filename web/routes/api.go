package routes

import (
	"github.com/afex/hystrix-go/hystrix"
	"log"
	"net/http"
	short_url_v1 "short_url_rpc_study/proto/short_url/v1"
	"time"

	"github.com/gin-gonic/gin"
)

type ApiHandler struct {
	svc short_url_v1.ShortUrlServiceClient
}

var _ Handler = (*ApiHandler)(nil)

func NewApiHandler(svc short_url_v1.ShortUrlServiceClient) *ApiHandler {
	return &ApiHandler{svc: svc}
}

func (ah *ApiHandler) RegisterRoutes(srv *gin.Engine) {
	api := srv.Group("/api")
	{
		api.POST("/create", ah.Create)
	}
}

func (ah *ApiHandler) Create(ctx *gin.Context) {
	type CreateRequest struct {
		OriginUrl string `json:"origin_url"`
	}
	var req CreateRequest
	if err := ctx.ShouldBind(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 使用带降级的熔断器保护RPC调用
	err := hystrix.Do("short_url",
		// 主要业务逻辑
		func() error {
			resp, err := ah.svc.GenerateShortUrl(ctx, &short_url_v1.GenerateShortUrlRequest{
				OriginUrl: req.OriginUrl,
			})
			if err != nil {
				return err
			}

			ctx.JSON(200, gin.H{
				"short_url": resp.GetShortUrl(),
			})
			return err
		},
		// 降级处理逻辑
		func(err error) error {
			// 记录降级日志
			log.Printf("[ServerHandler] Fallback msg: %s", err.Error())

			// 返回降级响应
			ctx.JSON(503, gin.H{
				"error":       "服务暂时不可用，请稍后再试",
				"code":        "SERVICE_DEGRADED",
				"retry_after": 30,
				"status":      "degraded",
			})
			return nil
		})

	if err != nil {
		// 记录错误日志
		log.Printf("[ApiHandler] RPC error: %v", err)

		// 处理具体错误类型
		ctx.JSON(500, gin.H{
			"error":     "Internal server error",
			"code":      "INTERNAL_ERROR",
			"timestamp": time.Now().Unix(),
		})
	}
}
