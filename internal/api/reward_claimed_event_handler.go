package api

import (
	"net/http"
	"strconv"
	"test-stake-backend/internal/repository"
	"test-stake-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type RewardClaimedEventHandler struct {
	service *service.RewardClaimedEventService
}

func NewRewardClaimedEventHandler(service *service.RewardClaimedEventService) *RewardClaimedEventHandler {
	return &RewardClaimedEventHandler{service: service}
}

func (h *RewardClaimedEventHandler) Register(r gin.IRouter) {
	group := r.Group("/reward-claimed-events")
	group.GET("", h.List)
	group.GET("/:id", h.GetByID)
}

func (h *RewardClaimedEventHandler) List(c *gin.Context) {
	query, err := parseRewardClaimedEventQuery(c)
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

func (h *RewardClaimedEventHandler) GetByID(c *gin.Context) {
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

func parseRewardClaimedEventQuery(c *gin.Context) (repository.RewardClaimedEventQuery, error) {
	var query repository.RewardClaimedEventQuery
	var err error

	query.ID, err = parseOptionalInt64(c, "id")
	if err != nil {
		return query, err
	}
	query.ContractAddress = c.Query("contract_address")
	query.User = c.Query("user")
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
