package infra

import (
	"database/sql"
	"fmt"
	"gochat/internal/config"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// NewMySQLPool 创建并配置一个 MySQL 连接池。
func NewMySQLPool(cfg *config.MySQLConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.DBName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	// 设置连接池参数
	db.SetMaxOpenConns(100)                // 最多同时多少条连接
	db.SetMaxIdleConns(20)                 // 最多保留几条空闲连接
	db.SetConnMaxLifetime(5 * time.Minute) // 一条连接最多活多久

	return db, nil
}
