package api

import (
	"fmt"
	"net/http"
	"test-stake-backend/internal/repository"
	"test-stake-backend/internal/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterRoutes 注册所有 HTTP 路由。
func RegisterRoutes(r *gin.Engine, db *gorm.DB) error {
	r.GET("/health", healthHandler)

	stakedEventRepo, err := repository.NewStakedEventRepository(db)
	if err != nil {
		return fmt.Errorf("register staked event repository: %w", err)
	}
	stakedEventService, err := service.NewStakedEventService(stakedEventRepo)
	if err != nil {
		return fmt.Errorf("register staked event service: %w", err)
	}
	NewStakedEventHandler(stakedEventService).Register(r)

	return nil
}

// healthHandler 健康检查
func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
