package router

import (
	"Go-AI-KV-System/internal/gateway/handler"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// NewRouter 初始化 Gin 引擎并注册所有路由
func NewRouter(kvHandler *handler.KVHandler, healthHandler *handler.HealthHandler) *gin.Engine {
	r := gin.Default()

	// 注册 OpenTelemetry 中间件
	r.Use(otelgin.Middleware("gateway-service"))

	// 1. 系统路由
	r.GET("/health", healthHandler.Ping)

	// 2. 业务路由
	v1 := r.Group("api/v1")
	{
		v1.POST("/kv", kvHandler.HandleSet)
		v1.GET("/kv", kvHandler.HandleGet)
		v1.DELETE("/kv", kvHandler.HandleDel)
	}

	return r
}
