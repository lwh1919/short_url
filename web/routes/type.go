package routes

import "github.com/gin-gonic/gin"

type Handler interface {
	RegisterRoutes(srv *gin.Engine)
}
