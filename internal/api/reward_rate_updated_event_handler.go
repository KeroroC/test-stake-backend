package api

import (
	"net/http"
	"strconv"
	"test-stake-backend/internal/repository"
	"test-stake-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type RewardRateUpdatedEventHandler struct {
	service *service.RewardRateUpdatedEventService
}

func NewRewardRateUpdatedEventHandler(service *service.RewardRateUpdatedEventService) *RewardRateUpdatedEventHandler {
	return &RewardRateUpdatedEventHandler{service: service}
}

func (h *RewardRateUpdatedEventHandler) Register(r gin.IRouter) {
	group := r.Group("/reward-rate-updated-events")
	group.GET("", h.List)
	group.GET("/:id", h.GetByID)
}

func (h *RewardRateUpdatedEventHandler) List(c *gin.Context) {
	query, err := parseRewardRateUpdatedEventQuery(c)
	if err != nil {
		respondError(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.service.List(c.Request.Context(), query)
	if err != nil {
		respondRepositoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *RewardRateUpdatedEventHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || id <= 0 {
		respondError(c, http.StatusBadRequest, "id must be a positive integer")
		return
	}

	event, err := h.service.GetByID(c.Request.Context(), id)
	if err != nil {
		respondRepositoryError(c, err)
		return
	}

	c.JSON(http.StatusOK, event)
}

func parseRewardRateUpdatedEventQuery(c *gin.Context) (repository.RewardRateUpdatedEventQuery, error) {
	var query repository.RewardRateUpdatedEventQuery
	var err error

	query.ID, err = parseOptionalInt64(c, "id")
	if err != nil {
		return query, err
	}
	query.ContractAddress = c.Query("contract_address")
	query.TxHash = c.Query("tx_hash")
	query.BlockNumberFrom, err = parseOptionalUint64Pointer(c, "block_number_from")
	if err != nil {
		return query, err
	}
	query.BlockNumberTo, err = parseOptionalUint64Pointer(c, "block_number_to")
	if err != nil {
		return query, err
	}
	query.Page, err = parseOptionalInt(c, "page")
	if err != nil {
		return query, err
	}
	query.PageSize, err = parseOptionalInt(c, "page_size")
	if err != nil {
		return query, err
	}

	return query, nil
}
