package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes 注册所有 HTTP 路由。
// 当前仅包含健康检查接口
func RegisterRoutes(r *gin.Engine) {
	r.GET("/health", healthHandler)
}

// healthHandler 健康检查
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
