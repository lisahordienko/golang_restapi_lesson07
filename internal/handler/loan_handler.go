package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"libratrack/internal/service"
)

type CreateLoanRequest struct {
	CopyID uint `json:"copy_id" binding:"required,gt=0"`
}

type LoanHandler struct {
	service *service.LoanService
}

func NewLoanHandler(service *service.LoanService) *LoanHandler {
	return &LoanHandler{service: service}
}

func (h *LoanHandler) Mine(c *gin.Context) {
	userID, ok := contextUserID(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apiError("unauthenticated", "Authentication required"))
		return
	}
	loans, err := h.service.Mine(c.Request.Context(), userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, apiError("internal_server_error", "Internal server error"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"loans": loans})
}

func (h *LoanHandler) Create(c *gin.Context) {
	userID, ok := contextUserID(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apiError("unauthenticated", "Authentication required"))
		return
	}
	var request CreateLoanRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "copy_id is required"))
		return
	}
	loan, err := h.service.Create(c.Request.Context(), userID, request.CopyID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusConflict, apiError("copy_unavailable", "Copy is not available"))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"loan": loan})
}

func (h *LoanHandler) Return(c *gin.Context) {
	loanID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || loanID == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_id", "Invalid loan id"))
		return
	}
	loan, err := h.service.Return(c.Request.Context(), uint(loanID))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, apiError("not_found", "Loan not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"loan": loan})
}
