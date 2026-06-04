package api

import (
	"net/http"
	"strconv"
	"test-stake-backend/internal/repository"
	"test-stake-backend/internal/service"

	"github.com/gin-gonic/gin"
)

type WithdrawnEventHandler struct {
	service *service.WithdrawnEventService
}

func NewWithdrawnEventHandler(service *service.WithdrawnEventService) *WithdrawnEventHandler {
	return &WithdrawnEventHandler{service: service}
}

func (h *WithdrawnEventHandler) Register(r gin.IRouter) {
	group := r.Group("/withdrawn-events")
	group.GET("", h.List)
	group.GET("/:id", h.GetByID)
}

func (h *WithdrawnEventHandler) List(c *gin.Context) {
	query, err := parseWithdrawnEventQuery(c)
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

func (h *WithdrawnEventHandler) GetByID(c *gin.Context) {
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

func parseWithdrawnEventQuery(c *gin.Context) (repository.WithdrawnEventQuery, error) {
	var query repository.WithdrawnEventQuery
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
