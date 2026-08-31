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

// ──────────────────────────────────────────────────────
// 通用错误码
// ──────────────────────────────────────────────────────
const (
	CodeSuccess = 0

	// ── 通用 1000~1099 ──
	CodeInternalError = 1000
	CodeMissingParam  = 1001
	CodeInvalidParam  = 1002
	CodeUnauthorized  = 1003

	// ── 认证 1100~1199 ──
	CodeUsernameTooShort = 1101
	CodePasswordTooShort = 1102
	CodeUsernameTaken    = 1103
	CodeUserNotFound     = 1104
	CodeWrongPassword    = 1105
	CodeInvalidToken     = 1106

	// ── 好友 1200~1299 ──
	CodeSelfRequest      = 1201
	CodeAlreadyFriends   = 1202
	CodeFriendBlocked    = 1203
	CodeDuplicateRequest = 1204
	CodeRequestNotFound  = 1205
	CodeNotRequestTarget = 1206
	CodeAlreadyBlocked   = 1207
)

// ──────────────────────────────────────────────────────
// 服务层 error string → 响应错误码映射
// ──────────────────────────────────────────────────────

// errorCodeMap 将 service 包定义的 error 常量字符串映射到标准错误码。
// 注意：私聊/群聊消息相关错误码（4001-4003, 5001-5003）沿用 redis 包中
var errorCodeMap = map[string]int{
	// 认证
	"用户名必须为3-50个字符": CodeUsernameTooShort,
	"密码必须至少为6个字符":   CodePasswordTooShort,
	"用户名已被占用":       CodeUsernameTaken,
	"用户未找到":         CodeUserNotFound,
	"密码错误":          CodeWrongPassword,
	"刷新令牌无效或已过期":    CodeInvalidToken,

	// 好友
	"不能给自己发送好友请求":     CodeSelfRequest,
	"已经是该用户的好友":       CodeAlreadyFriends,
	"你已拉黑该用户或已被该用户拉黑": CodeFriendBlocked,
	"已存在待处理的好友请求":     CodeDuplicateRequest,
	"好友请求未找到":         CodeRequestNotFound,
	"你不是该好友请求的接收者":    CodeNotRequestTarget,
	"你已经拉黑了该用户":       CodeAlreadyBlocked,
}

// MapErrorCode 将 service 层返回的 error 字符串映射为前端错误码。
// 无法匹配时返回 CodeInternalError。
func MapErrorCode(errStr string) int {
	if code, ok := errorCodeMap[errStr]; ok {
		return code
	}
	return CodeInternalError
}

// ──────────────────────────────────────────────────────
// 便捷响应辅助函数
// ──────────────────────────────────────────────────────

// Success 返回 code=0, message="ok" 的成功响应 (HTTP 200)。
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ApiResponse{
		Code:    CodeSuccess,
		Message: "ok",
		Data:    data,
	})
}

// SuccessCreated 返回 code=0 的成功响应 (HTTP 201)。
func SuccessCreated(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, ApiResponse{
		Code:    CodeSuccess,
		Message: "ok",
		Data:    data,
	})
}

// SuccessMessage 返回 code=0 的成功响应，仅含提示消息，无 data。
func SuccessMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, ApiResponse{
		Code:    CodeSuccess,
		Message: message,
	})
}

// Error 返回自定义错误码和消息的失败响应。
func Error(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, ApiResponse{
		Code:    code,
		Message: message,
	})
}

// ServiceError 根据 service 层返回的 error 字符串自动映射错误码，
// 并使用合适的 HTTP 状态码。适用于 handler 中 switch err.Error() 分支。
// httpStatus 由调用方显式传入（保证语义准确），code 从 MapErrorCode 自动解析。
func ServiceError(c *gin.Context, httpStatus int, errMsg string) {
	c.JSON(httpStatus, ApiResponse{
		Code:    MapErrorCode(errMsg),
		Message: errMsg,
	})
}

// ──────────────────────────────────────────────────────
// 分页
// ──────────────────────────────────────────────────────

// PaginationMeta 是分页列表响应的元数据。
type PaginationMeta struct {
	Total   int64 `json:"total"`    // 总共多少条记录
	Offset  int   `json:"offset"`   // 这页从哪开始
	Limit   int   `json:"limit"`    // 一页多大
	HasMore bool  `json:"has_more"` // 还有没有下一页
}

// PaginatedData 是带分页信息的数据载荷。
type PaginatedData struct {
	Items      interface{}    `json:"items"`
	Pagination PaginationMeta `json:"pagination"`
}

// PaginatedSuccess 返回 code=0 的分页列表响应。
func PaginatedSuccess(c *gin.Context, items interface{}, total int64, offset, limit int) {
	hasMore := int64(offset+limit) < total
	c.JSON(http.StatusOK, ApiResponse{
		Code:    CodeSuccess,
		Message: "ok",
		Data: PaginatedData{
			Items: items,
			Pagination: PaginationMeta{
				Total:   total,
				Offset:  offset,
				Limit:   limit,
				HasMore: hasMore,
			},
		},
	})
}
