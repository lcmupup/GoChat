package middleware

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims 是一个JWT令牌中的"负载"字段
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 使用给定的过期时间创建签名的 JWT 访问令牌。
// expireHours 是令牌的有效期，以小时为单位（通常为 2）。
func GenerateAccessToken(userID int64, username, secret string, expireHours int) (string, error) {
	claims := &Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireHours) * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret)) // 将token拼接上签名字段返回完整的JWT令牌
}

// GenerateRefreshToken 使用给定的过期时间创建签名的 JWT 刷新令牌。
// expireDays 是令牌的有效期，以天为单位（通常为 7）。
// 刷新令牌不携带用户名 — 仅嵌入 userID。
func GenerateRefreshToken(userID int64, secret string, expireDays int) (string, error) {
	claims := &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireDays) * 24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseToken 使用给定的密钥解析并验证 JWT 令牌字符串。
// 成功时返回解析后的令牌和 claims，失败时返回错误。
// JWTAuthMiddleware 和 ServeWebSocket 都使用此辅助函数，以避免重复 JWT 解析逻辑。
func ParseToken(tokenStr, secret string) (*jwt.Token, *Claims, error) {
	claims := &Claims{}
	parsedToken, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	// 签名对不上，不给过
	if err != nil {
		return nil, nil, err
	}
	// 签名已过期，同样不给过
	if !parsedToken.Valid {
		return nil, nil, fmt.Errorf("令牌无效")
	}
	return parsedToken, claims, nil
}
