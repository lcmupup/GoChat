package service

import (
	"context"
	"fmt"
	"gochat/internal/model"
	"gochat/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

// ── 验证 / 业务错误常量 ──

const (
	ErrUsernameTooShort = "用户名必须为3-50个字符"
	ErrPasswordTooShort = "密码必须至少为6个字符"
	ErrUsernameTaken    = "用户名已被占用"
	ErrUserNotFound     = "用户未找到"
	ErrWrongPassword    = "密码错误"
	ErrInvalidToken     = "刷新令牌无效或已过期"
)

// AuthService 处理用户注册、登录和令牌刷新。
type AuthService struct {
	repo           repository.MySQLRepo
	jwtSecret      string
	bcryptCost     int
	accessExpHours int
	refreshExpDays int
}

// NewAuthService 使用给定的 MySQL 仓库和 JWT 配置创建 AuthService。
func NewAuthService(repo repository.MySQLRepo, jwtSecret string, accessExpHours, refreshExpDays int) *AuthService {
	return &AuthService{
		repo:           repo,
		jwtSecret:      jwtSecret,
		bcryptCost:     10,
		accessExpHours: accessExpHours,
		refreshExpDays: refreshExpDays,
	}
}

// Register 验证输入，对密码进行哈希处理，并创建新用户。
// 成功时返回新用户的 ID 和用户名。
func (s *AuthService) Register(ctx context.Context, username, password string) (int64, string, error) {
	// 验证用户名长度
	if len(username) < 3 || len(username) > 50 {
		return 0, "", fmt.Errorf(ErrUsernameTooShort)
	}
	// 验证密码长度
	if len(password) < 6 {
		return 0, "", fmt.Errorf(ErrPasswordTooShort)
	}

	// 检查用户名唯一性
	existing, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return 0, "", fmt.Errorf("检查用户名: %w", err)
	}
	if existing != nil { // 该用户名已存在
		return 0, "", fmt.Errorf(ErrUsernameTaken)
	}

	// 哈希密码
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return 0, "", fmt.Errorf("哈希密码: %w", err)
	}

	// 创建用户记录
	user := &model.User{
		Username:     username,
		PasswordHash: string(hash),
		Nickname:     username, // 默认昵称 = 用户名
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return 0, "", fmt.Errorf("创建用户: %w", err)
	}

	return user.ID, user.Username, nil
}
