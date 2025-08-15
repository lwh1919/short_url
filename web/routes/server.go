package routes

import (
	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
	"short_url_rpc_study/pkg/generator"
	short_url_v1 "short_url_rpc_study/proto/short_url/v1"
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
	//请求合并层 (防缓存击穿)
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
		ctx.JSON(500, gin.H{"error": err.Error()})
		return
	}
	ctx.Redirect(301, result.(string))
}
