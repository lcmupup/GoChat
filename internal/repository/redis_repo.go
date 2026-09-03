package repository

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

// RedisRepo 定义了消息服务和下游MQ消费者所需的所有Redis操作。
// 该接口便于在测试中进行mock。
type RedisRepo interface {
	// ── 好友缓存 ──
	SetFriendCache(ctx context.Context, uidA, uidB int64) error
}

// ──────────────────────────────────────────────────────
// RedisRepoImpl — 使用go-redis的具体实现
// ──────────────────────────────────────────────────────

type RedisRepoImpl struct {
	rdb *redis.Client
}

func NewRedisRepo(rdb *redis.Client) *RedisRepoImpl {
	return &RedisRepoImpl{rdb: rdb}
}

// ── 好友缓存 ──

// SetFriendCache 在 Redis 中写入双向好友关系缓存。
// Lua 消息校验脚本依赖此 key 判断好友关系。
func (r *RedisRepoImpl) SetFriendCache(ctx context.Context, uidA, uidB int64) error {
	pipe := r.rdb.Pipeline()
	// 用 pipeline 写入双向好友关系
	pipe.Set(ctx, fmt.Sprintf("friend:%d:%d", uidA, uidB), "1", 0)
	pipe.Set(ctx, fmt.Sprintf("friend:%d:%d", uidB, uidA), "1", 0)
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("设置好友缓存 %d<->%d: %w", uidA, uidB, err)
	}
	return nil
}

// DeleteFriendCache 移除用于私信 Lua 授权检查的双向好友关系键
// 这个方法没有加到 RedisRepo 接口中
func (r *RedisRepoImpl) DeleteFriendCache(ctx context.Context, uidA, uidB int64) error {
	if err := r.rdb.Del(ctx,
		fmt.Sprintf("friend:%d:%d", uidA, uidB),
		fmt.Sprintf("friend:%d:%d", uidB, uidA),
	).Err(); err != nil {
		return fmt.Errorf("删除好友缓存 %d<->%d: %w", uidA, uidB, err)
	}
	return nil
}
