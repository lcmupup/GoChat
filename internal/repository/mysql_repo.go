package repository

import (
	"context"
	"database/sql"
	"fmt"
	"gochat/internal/model"
)

// MySQLRepo 定义了服务和消费者所需的所有 MySQL CRUD 操作。
type MySQLRepo interface {
	// 用户
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) error
}

// MySQLRepoImpl — 基于 database/sql 的具体实现
type MySQLRepoImpl struct {
	db *sql.DB
}

func NewMySQLRepo(db *sql.DB) *MySQLRepoImpl {
	return &MySQLRepoImpl{db}
}

func (m *MySQLRepoImpl) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `SELECT id, username, password_hash, nickname, avatar_url, sign, gender, created_at, updated_at
				FROM users WHERE username = ?`
	row := m.db.QueryRowContext(ctx, query, username)
	var u model.User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Nickname, &u.AvatarURL, &u.Sign, &u.Gender, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows { // 如果只是因为用户不存在而发生的失败则不属于程序错误，是正常业务结果
			return nil, nil
		}
		return nil, fmt.Errorf("按用户名获取用户: %w", err) // 真错误
	}
	return &u, nil // 成功返回结果
}

func (m *MySQLRepoImpl) CreateUser(ctx context.Context, user *model.User) error {
	query := `INSERT INTO users (username, password_hash, nickname, avatar_url, sign, gender)
			VALUES (?, ?, ?, ?, ?, ?)`
	result, err := m.db.ExecContext(ctx, query,
		user.Username,
		user.PasswordHash,
		user.Nickname,
		user.AvatarURL,
		user.Sign,
		user.Gender,
	)
	if err != nil {
		return fmt.Errorf("创建用户: %w", err)
	}
	id, err := result.LastInsertId() // 获取当前用户插入后的用户id
	if err != nil {
		return fmt.Errorf("创建用户 获取最后插入ID: %w", err)
	}
	user.ID = id // 将获取到的id写入到传来的结构体中
	return nil
}
