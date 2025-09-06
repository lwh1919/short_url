package routes

import (
	"net/http"
	"time"

	"github.com/afex/hystrix-go/hystrix"
	"github.com/gin-gonic/gin"
)

// HealthHandler 结构体定义
type HealthHandler struct {
	circuitBreakerName string
}

// NewHealthHandler 创建健康检查处理器
func NewHealthHandler(circuitBreakerName string) *HealthHandler {
	return &HealthHandler{
		circuitBreakerName: circuitBreakerName,
	}
}

// RegisterRoutes 注册路由
func (h *HealthHandler) RegisterRoutes(srv *gin.Engine) {
	srv.GET("/api/health", h.Health)
	srv.GET("/health", h.Health) // 兼容旧路径
}

// Health 健康检查
func (h *HealthHandler) Health(ctx *gin.Context) {
	// 获取服务状态
	status, details := h.getServiceStatus()

	// 根据状态设置HTTP状态码
	httpStatus := http.StatusOK
	if status == "unavailable" {
		httpStatus = http.StatusServiceUnavailable
	} else if status == "degraded" {
		httpStatus = http.StatusTooManyRequests // 429更适合降级状态
	}

	ctx.JSON(httpStatus, gin.H{
		"status":  status,
		"details": details,
	})
}

// getServiceStatus 获取服务状态
func (h *HealthHandler) getServiceStatus() (string, map[string]interface{}) {
	// 获取熔断器状态
	circuit, _, err := hystrix.GetCircuit(h.circuitBreakerName)
	if err != nil {
		// 如果熔断器不存在，返回健康状态
		details := map[string]interface{}{
			"name":       h.circuitBreakerName,
			"error":      err.Error(),
			"checked_at": time.Now().Format(time.RFC3339),
		}
		return "ok", details
	}

	// 获取熔断器指标
	isOpen := circuit.IsOpen()
	allowRequest := circuit.AllowRequest()

	// 构建详情信息
	details := map[string]interface{}{
		"name":          h.circuitBreakerName,
		"is_open":       isOpen,
		"allow_request": allowRequest,
		"checked_at":    time.Now().Format(time.RFC3339),
	}

	// 确定状态
	var status string
	if isOpen {
		status = "unavailable"
	} else if !allowRequest {
		status = "degraded"
	} else {
		status = "ok"
	}

	return status, details
}
