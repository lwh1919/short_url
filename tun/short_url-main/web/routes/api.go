package routes

import (
	short_url_v1 "short_url/proto/short_url/v1"

	"github.com/gin-gonic/gin"
)

type ApiHandler struct {
	svc short_url_v1.ShortUrlServiceClient
}

var _ Handler = (*ApiHandler)(nil)

func NewApiHandler(svc short_url_v1.ShortUrlServiceClient) *ApiHandler {
	return &ApiHandler{
		svc: svc,
	}
}

func (h *ApiHandler) RegisterRoutes(srv *gin.Engine) {
	api := srv.Group("/api")
	{
		api.POST("/create", h.Create)
	}
}

func (h *ApiHandler) Create(ctx *gin.Context) {
	type CreateRequest struct {
		OriginUrl string `json:"origin_url"`
	}
	var req CreateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(400, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.GenerateShortUrl(ctx, &short_url_v1.GenerateShortUrlRequest{
		OriginUrl: req.OriginUrl,
	})
	if err != nil {
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.JSON(200, gin.H{
		"short_url": resp.GetShortUrl(),
	})
}
