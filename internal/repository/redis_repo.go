package repository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// RedisRepo 定义了消息服务和下游MQ消费者所需的所有Redis操作。
// 该接口便于在测试中进行mock。
type RedisRepo interface {
	// ── 群组成员关系 ──
	GetGroupMemberships(ctx context.Context, userID int64) ([]int64, error)
	GetGroupMembers(ctx context.Context, groupID int64) ([]int64, error)
	AddGroupMemberRedis(ctx context.Context, groupID, userID int64) error

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

// ── 群组成员关系 ──

// GetGroupMemberships 查这个人加入了哪些群,只得到群聊id
func (r *RedisRepoImpl) GetGroupMemberships(ctx context.Context, userID int64) ([]int64, error) {
	key := fmt.Sprintf("user_groups:%d", userID)
	results, err := r.rdb.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("SMembers用户群组: %w", err)
	}

	groupIDs := make([]int64, 0, len(results))
	for _, s := range results {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			continue
		}
		groupIDs = append(groupIDs, id)
	}
	return groupIDs, nil
}

// GetGroupMembers 查这个群有哪些成员,只得到成员的用户id
func (r *RedisRepoImpl) GetGroupMembers(ctx context.Context, groupID int64) ([]int64, error) {
	key := fmt.Sprintf("group_members:%d", groupID)
	results, err := r.rdb.SMembers(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("SMembers群组成员: %w", err)
	}

	memberIDs := make([]int64, 0, len(results))
	for _, s := range results {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			continue
		}
		memberIDs = append(memberIDs, id)
	}
	return memberIDs, nil
}

func (r *RedisRepoImpl) AddGroupMemberRedis(ctx context.Context, groupID, userID int64) error {
	groupKey := fmt.Sprintf("group_members:%d", groupID)
	userKey := fmt.Sprintf("user_groups:%d", userID)
	userIDStr := strconv.FormatInt(userID, 10)
	groupIDStr := strconv.FormatInt(groupID, 10)

	if err := r.rdb.SAdd(ctx, groupKey, userIDStr).Err(); err != nil {
		return fmt.Errorf("SADD群组成员: %w", err)
	}
	if err := r.rdb.SAdd(ctx, userKey, groupIDStr).Err(); err != nil {
		return fmt.Errorf("SADD用户群组: %w", err)
	}
	if err := r.rdb.HSetNX(ctx, fmt.Sprintf("group_member_info:%d", groupID), userIDStr, `{"role":0,"muted":false}`).Err(); err != nil {
		return fmt.Errorf("HSET群成员信息: %w", err)
	}
	// TODO
	// 新成员已读水位线初始化为当前群序号：历史消息不计入未读。
	// （水位线模型下未读 = 群最新 seq − 游标，缺失游标会使整个群历史都算未读）
	// seq, err := r.GetGroupSeq(ctx, groupID)
	// if err != nil {
	// 	return fmt.Errorf("读取群序号以初始化已读游标: %w", err)
	// }
	// convID := fmt.Sprintf("g_%d", groupID)
	// if err := r.rdb.HSet(ctx, fmt.Sprintf("group_read_pos:%d", userID), convID, strconv.FormatInt(seq, 10)).Err(); err != nil {
	// 	return fmt.Errorf("初始化群已读游标: %w", err)
	// }
	return nil
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
