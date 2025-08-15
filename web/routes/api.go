package routes

import (
	"github.com/gin-gonic/gin"
	"net/http"
	short_url_v1 "short_url_rpc_study/proto/short_url/v1"
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

	resp, err := ah.svc.GenerateShortUrl(ctx, &short_url_v1.GenerateShortUrlRequest{
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
