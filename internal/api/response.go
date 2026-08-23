package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 统一 API 响应格式
type ApiResponse struct {
	Code    int         `json:"code"`           // 业务错误码（并非HTTP状态码，管的是业务层面的错误）
	Message string      `json:"message"`        // 展示给用户看的信息
	Data    interface{} `json:"data,omitempty"` // 业务数据
}

// 通用错误码
const (
	CodeSuccess = 0

	// ── 通用 1000~1099 ──
	CodeInternalError = 1000
	// CodeMissingParam  = 1001
	CodeInvalidParam = 1002
	// CodeUnauthorized  = 1003
)

// Success 返回 code=0, message="ok" 的成功响应 (HTTP 200)。
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ApiResponse{
		Code:    CodeSuccess,
		Message: "ok",
		Data:    data,
	})
}

// Error 返回自定义错误码和消息的失败响应。
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, ApiResponse{
		Code:    code,
		Message: message,
	})
}
