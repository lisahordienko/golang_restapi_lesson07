package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"libratrack/internal/auth"
)

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=72"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type UpdateProfileRequest struct {
	Email string `json:"email" binding:"required,email"`
}

type UpdateUserRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=reader librarian admin"`
}

type AuthHandler struct {
	service *auth.Service
}

func NewAuthHandler(service *auth.Service) *AuthHandler {
	return &AuthHandler{service: service}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "Invalid registration fields"))
		return
	}
	user, err := h.service.Register(c.Request.Context(), auth.RegisterInput{Email: request.Email, Password: request.Password})
	if errors.Is(err, auth.ErrDuplicateEmail) {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "Email is already registered"))
		return
	}
	if errors.Is(err, auth.ErrJWTSecretMissing) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, apiError("internal_server_error", "Authentication is unavailable"))
		return
	}
	if err != nil {
		if strings.Contains(err.Error(), "invalid registration") {
			c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "Invalid registration fields"))
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, apiError("internal_server_error", "Internal server error"))
		return
	}
	c.JSON(http.StatusCreated, gin.H{"user": user})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apiError("unauthorized", "invalid credentials"))
		return
	}
	user, token, err := h.service.Login(c.Request.Context(), request.Email, request.Password)
	if errors.Is(err, auth.ErrInvalidCredentials) {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apiError("unauthorized", "invalid credentials"))
		return
	}
	if errors.Is(err, auth.ErrJWTSecretMissing) {
		c.AbortWithStatusJSON(http.StatusInternalServerError, apiError("internal_server_error", "Authentication is unavailable"))
		return
	}
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, apiError("internal_server_error", "Internal server error"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": token, "user": user})
}

func (h *AuthHandler) BrowserLogin(c *gin.Context) {
	_, token, err := h.service.Login(c.Request.Context(), c.PostForm("email"), c.PostForm("password"))
	if errors.Is(err, auth.ErrInvalidCredentials) {
		c.String(http.StatusUnauthorized, "Invalid email or password")
		return
	}
	if errors.Is(err, auth.ErrJWTSecretMissing) {
		c.String(http.StatusInternalServerError, "Authentication is unavailable")
		return
	}
	if err != nil {
		c.String(http.StatusInternalServerError, "Unable to sign in")
		return
	}
	maxAge := int((24 * time.Hour).Seconds())
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("libratrack_session", token, maxAge, "/", "", c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https", true)
	c.Redirect(http.StatusSeeOther, "/catalog")
}

func (h *AuthHandler) Me(c *gin.Context) {
	userID, ok := contextUserID(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apiError("unauthenticated", "Authentication required"))
		return
	}
	user, err := h.service.User(c.Request.Context(), userID)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, apiError("not_found", "User not found"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"user": user})
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	userID, ok := contextUserID(c)
	if !ok {
		c.AbortWithStatusJSON(http.StatusUnauthorized, apiError("unauthenticated", "Authentication required"))
		return
	}
	var request UpdateProfileRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "Invalid profile fields"))
		return
	}
	if err := h.service.UpdateProfile(c.Request.Context(), userID, request.Email); err != nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "Email could not be updated"))
		return
	}
	h.Me(c)
}

func (h *AuthHandler) ListUsers(c *gin.Context) {
	users, err := h.service.Users(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, apiError("internal_server_error", "Internal server error"))
		return
	}
	c.JSON(http.StatusOK, gin.H{"users": users})
}

func (h *AuthHandler) UpdateUserRole(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || userID == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, apiError("invalid_id", "Invalid user id"))
		return
	}
	var request UpdateUserRoleRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.AbortWithStatusJSON(http.StatusUnprocessableEntity, apiError("validation_error", "Invalid role"))
		return
	}
	if err := h.service.UpdateRole(c.Request.Context(), uint(userID), request.Role); err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, apiError("not_found", "User not found"))
		return
	}
	c.Status(http.StatusNoContent)
}

func contextUserID(c *gin.Context) (uint, bool) {
	value, exists := c.Get("user_id")
	if !exists {
		return 0, false
	}
	id, ok := value.(uint)
	return id, ok && id > 0
}

func apiError(code, message string) gin.H {
	return gin.H{"error": gin.H{"code": code, "message": message}}
}
