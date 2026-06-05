package api

import (
	"fmt"
	"net/http"
	"time"

	"test-stake-backend/internal/repository"
	"test-stake-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func RegisterRoutes(r *gin.Engine, db *gorm.DB, rdb *redis.Client, cacheTTL time.Duration) error {
	r.GET("/health", healthHandler)

	// StakedEvent
	stakedEventRepo, err := repository.NewStakedEventRepository(db)
	if err != nil {
		return fmt.Errorf("register staked event repository: %w", err)
	}
	stakedEventService, err := service.NewStakedEventService(stakedEventRepo, rdb, cacheTTL)
	if err != nil {
		return fmt.Errorf("register staked event service: %w", err)
	}
	NewStakedEventHandler(stakedEventService).Register(r)

	// RewardClaimedEvent
	rewardClaimedEventRepo, err := repository.NewRewardClaimedEventRepository(db)
	if err != nil {
		return fmt.Errorf("register reward claimed event repository: %w", err)
	}
	rewardClaimedEventService, err := service.NewRewardClaimedEventService(rewardClaimedEventRepo, rdb, cacheTTL)
	if err != nil {
		return fmt.Errorf("register reward claimed event service: %w", err)
	}
	NewRewardClaimedEventHandler(rewardClaimedEventService).Register(r)

	// WithdrawnEvent
	withdrawnEventRepo, err := repository.NewWithdrawnEventRepository(db)
	if err != nil {
		return fmt.Errorf("register withdrawn event repository: %w", err)
	}
	withdrawnEventService, err := service.NewWithdrawnEventService(withdrawnEventRepo, rdb, cacheTTL)
	if err != nil {
		return fmt.Errorf("register withdrawn event service: %w", err)
	}
	NewWithdrawnEventHandler(withdrawnEventService).Register(r)

	// MinStakeAmountUpdatedEvent
	minStakeAmountUpdatedEventRepo, err := repository.NewMinStakeAmountUpdatedEventRepository(db)
	if err != nil {
		return fmt.Errorf("register min stake amount updated event repository: %w", err)
	}
	minStakeAmountUpdatedEventService, err := service.NewMinStakeAmountUpdatedEventService(minStakeAmountUpdatedEventRepo, rdb, cacheTTL)
	if err != nil {
		return fmt.Errorf("register min stake amount updated event service: %w", err)
	}
	NewMinStakeAmountUpdatedEventHandler(minStakeAmountUpdatedEventService).Register(r)

	// RewardRateUpdatedEvent
	rewardRateUpdatedEventRepo, err := repository.NewRewardRateUpdatedEventRepository(db)
	if err != nil {
		return fmt.Errorf("register reward rate updated event repository: %w", err)
	}
	rewardRateUpdatedEventService, err := service.NewRewardRateUpdatedEventService(rewardRateUpdatedEventRepo, rdb, cacheTTL)
	if err != nil {
		return fmt.Errorf("register reward rate updated event service: %w", err)
	}
	NewRewardRateUpdatedEventHandler(rewardRateUpdatedEventService).Register(r)

	return nil
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
