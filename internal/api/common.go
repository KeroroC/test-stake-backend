package api

import (
	"errors"
	"net/http"
	"strconv"
	"test-stake-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

func parseOptionalInt(c *gin.Context, key string) (int, error) {
	value := c.Query(key)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, errors.New(key + " must be an integer")
	}

	return parsed, nil
}

func parseOptionalInt64(c *gin.Context, key string) (int64, error) {
	value := c.Query(key)
	if value == "" {
		return 0, nil
	}

	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, errors.New(key + " must be an integer")
	}

	return parsed, nil
}

func parseOptionalUint64Pointer(c *gin.Context, key string) (*uint64, error) {
	value := c.Query(key)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return nil, errors.New(key + " must be an unsigned integer")
	}

	return &parsed, nil
}

func respondRepositoryError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, repository.ErrInvalidStakedEvent),
		errors.Is(err, repository.ErrInvalidRewardClaimedEvent),
		errors.Is(err, repository.ErrInvalidWithdrawnEvent),
		errors.Is(err, repository.ErrInvalidMinStakeAmountUpdatedEvent),
		errors.Is(err, repository.ErrInvalidRewardRateUpdatedEvent):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrStakedEventNotFound),
		errors.Is(err, repository.ErrRewardClaimedEventNotFound),
		errors.Is(err, repository.ErrWithdrawnEventNotFound),
		errors.Is(err, repository.ErrMinStakeAmountUpdatedEventNotFound),
		errors.Is(err, repository.ErrRewardRateUpdatedEventNotFound):
		respondError(c, http.StatusNotFound, err.Error())
	default:
		respondError(c, http.StatusInternalServerError, "internal server error")
	}
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
