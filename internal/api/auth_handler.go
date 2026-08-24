package api

import (
	"gochat/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// AuthHandler 提供认证端点的 Gin HTTP 处理器。
type AuthHandler struct {
	authSvc *service.AuthService
}

// NewAuthHandler 创建一个 AuthHandler，封装给定的 AuthService。
func NewAuthHandler(authSvc *service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

// 请求与响应
type registerRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type registerResponse struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
}

// ── Handlers ──

// Register godoc
// @Summary      用户注册
// @Description  使用用户名和密码注册新用户
// @Tags         认证
// @Accept       json
// @Produce      json
// @Param        body  body  registerRequest  true  "注册信息"
// @Success      201   {object}  ApiResponse{data=registerResponse}  "注册成功"
// @Failure      400   {object}  ApiResponse  "参数错误"
// @Failure      409   {object}  ApiResponse  "用户名已存在"
// @Router       /auth/register [post]
// Register handles POST /api/v1/auth/register.
func (h *AuthHandler) Register(c *gin.Context) {
	var req registerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Error(c, http.StatusBadRequest, CodeMissingParam, "username and password are required")
		return
	}

	userID, username, err := h.authSvc.Register(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch err.Error() {
		case service.ErrUsernameTooShort, service.ErrPasswordTooShort:
			ServiceError(c, http.StatusBadRequest, err.Error())
		case service.ErrUsernameTaken:
			ServiceError(c, http.StatusConflict, err.Error())
		default:
			Error(c, http.StatusInternalServerError, CodeInternalError, "internal error")
		}
		return
	}

	SuccessCreated(c, registerResponse{UserID: userID, Username: username})
}

// RegisterRoutes 在给定的路由组中注册路由
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/register", h.Register)
}
