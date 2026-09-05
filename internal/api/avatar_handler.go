package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// AvatarHandler 提供头像端点的 Gin HTTP 处理函数。
type AvatarHandler struct{}

// NewAvatarHandler 创建一个 AvatarHandler。
func NewAvatarHandler() *AvatarHandler {
	return &AvatarHandler{}
}

// GetAvatar godoc
// @Summary      获取用户头像
// @Description  返回用户头像图片。如果用户未设置头像，自动生成包含用户名首字母的 SVG 占位头像
// @Tags         用户
// @Produce      image/svg+xml
// @Param        userID  path  int64  true  "用户ID"
// @Param        name    query string false "用户名（用于生成默认头像首字母）"
// @Success      200  {string}  string  "头像图片（SVG或重定向到图片URL）"
// @Router       /avatar/{userID} [get]
func (h *AvatarHandler) GetAvatar(c *gin.Context) {
	// 这是公开端点，无需认证
	// userIDStr := c.Param("userID")
	// userID, _ := strconv.ParseInt(userIDStr, 10, 64)

	// 获取用户名用于生成首字母（从查询参数，前端传入）
	name := c.DefaultQuery("name", "?")
	if len(name) == 0 {
		name = "?"
	}

	// 取用户名首字母并大写，哪怕 name 是"?"这里也会原样返回"?"
	initial := strings.ToUpper(name[:1]) // strings.ToUpper就算传入的不是字母也不会报错，而会原样返回

	// 根据首字母生成默认SVG占位头像的颜色
	color := initialColor(initial)

	// 生成 SVG 占位头像
	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="200" viewBox="0 0 200 200">
  <rect width="200" height="200" fill="%s"/>
  <text x="100" y="135" font-family="Arial, sans-serif" font-size="100" font-weight="bold" fill="#fff" text-anchor="middle">%s</text>
</svg>`, color, initial)

	c.Header("Content-Type", "image/svg+xml")          // 告诉浏览器返回的内容是一张 SVG 格式的图片，让浏览器当成图片进行渲染
	c.Header("Cache-Control", "public, max-age=86400") // 告诉浏览器这张图片可以被缓存，有效期是 86400 秒（也就是 24 小时）
	c.String(http.StatusOK, svg)                       // 把 svg 字符串作为响应体直接输出
}

// RegisterRoutes 在公开路由组上注册头像路由
func (h *AvatarHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/avatar/:userID", h.GetAvatar)
}

// initialColor 根据用户名首字母返回默认SVG占位头像的颜色（Material Design 调色板）。
func initialColor(initial string) string {
	colors := map[string]string{
		"A": "#F44336", "B": "#E91E63", "C": "#9C27B0", "D": "#673AB7",
		"E": "#3F51B5", "F": "#2196F3", "G": "#03A9F4", "H": "#00BCD4",
		"I": "#009688", "J": "#4CAF50", "K": "#8BC34A", "L": "#CDDC39",
		"M": "#FF9800", "N": "#FF5722", "O": "#795548", "P": "#607D8B",
		"Q": "#E53935", "R": "#D81B60", "S": "#8E24AA", "T": "#5E35B1",
		"U": "#3949AB", "V": "#1E88E5", "W": "#039BE5", "X": "#00ACC1",
		"Y": "#00897B", "Z": "#43A047",
	}
	if c, ok := colors[initial]; ok {
		return c
	}
	return "#9E9E9E"
}
