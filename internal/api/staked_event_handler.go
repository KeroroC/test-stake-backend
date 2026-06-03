package api

import (
	"errors"
	"net/http"
	"strconv"
	"test-stake-backend/internal/models"
	"test-stake-backend/internal/repository"
	"test-stake-backend/internal/service"

	"github.com/gin-gonic/gin"
)

// StakedEventHandler 处理 StakedEvent HTTP 请求。
type StakedEventHandler struct {
	service *service.StakedEventService
}

// NewStakedEventHandler 创建 StakedEventHandler。
func NewStakedEventHandler(service *service.StakedEventService) *StakedEventHandler {
	return &StakedEventHandler{service: service}
}

// Register 注册 StakedEvent 路由。
func (h *StakedEventHandler) Register(r gin.IRouter) {
	group := r.Group("/staked-events")
	group.POST("", h.Create)
	group.GET("", h.List)
	group.GET("/:id", h.GetByID)
}

// Create 创建 StakedEvent。
func (h *StakedEventHandler) Create(c *gin.Context) {
	var req createStakedEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	event := &models.StakedEvent{
		ContractAddress: req.ContractAddress,
		User:            req.User,
		Amount:          req.Amount,
		TxHash:          req.TxHash,
		BlockNumber:     req.BlockNumber,
		LogIndex:        req.LogIndex,
		BlockHash:       req.BlockHash,
	}

	created, err := h.service.Create(c.Request.Context(), event)
	if err != nil {
		respondRepositoryError(c, err)
		return
	}

	c.JSON(http.StatusCreated, created)
}

// List 分页查询 StakedEvent。
func (h *StakedEventHandler) List(c *gin.Context) {
	query, err := parseStakedEventQuery(c)
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

// GetByID 按 ID 查询 StakedEvent。
func (h *StakedEventHandler) GetByID(c *gin.Context) {
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

type createStakedEventRequest struct {
	ContractAddress string `json:"contract_address" binding:"required"`
	User            string `json:"user" binding:"required"`
	Amount          string `json:"amount" binding:"required"`
	TxHash          string `json:"tx_hash" binding:"required"`
	BlockNumber     uint64 `json:"block_number" binding:"required"`
	LogIndex        uint   `json:"log_index"`
	BlockHash       string `json:"block_hash" binding:"required"`
}

func parseStakedEventQuery(c *gin.Context) (repository.StakedEventQuery, error) {
	var query repository.StakedEventQuery
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
	case errors.Is(err, repository.ErrInvalidStakedEvent):
		respondError(c, http.StatusBadRequest, err.Error())
	case errors.Is(err, repository.ErrStakedEventNotFound):
		respondError(c, http.StatusNotFound, err.Error())
	default:
		respondError(c, http.StatusInternalServerError, "internal server error")
	}
}

func respondError(c *gin.Context, status int, message string) {
	c.JSON(status, gin.H{"error": message})
}
