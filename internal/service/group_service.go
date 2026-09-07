package service

import (
	"context"
	"fmt"
	"gochat/internal/model"
	"gochat/internal/repository"

	"go.uber.org/zap"
)

// ── 群组服务错误常量 ──

const (
	ErrNotOwnerOrAdmin    = "只有群主或管理员才能执行此操作"
	ErrGroupNotFound      = "群组不存在"
	ErrAlreadyMember      = "用户已是群组成员"
	ErrGroupFull          = "群组已满（最多 500 人）"
	ErrCannotRemoveOwner  = "无法移除群主"
	ErrCannotLeaveAsOwner = "群主无法退出群组；请先转让群主身份或解散群组"
	ErrInvalidRole        = "角色值必须为 0（普通成员）或 1（管理员）"
	ErrMemberNotFound     = "群成员未找到"
	ErrMemberNotFriend    = "只能邀请好友加入群组"
	ErrCannotRemovePeer   = "管理员只能移除普通成员"
)

const maxGroupMembers = 500

// GroupService 处理群组相关的业务逻辑。
type GroupService struct {
	mysqlRepo repository.MySQLRepo
	redisRepo repository.RedisRepo
	logger    *zap.Logger
}

// NewGroupService 使用给定的仓库和日志记录器创建一个 GroupService。
func NewGroupService(mysqlRepo repository.MySQLRepo, redisRepo repository.RedisRepo, logger *zap.Logger) *GroupService {
	return &GroupService{
		mysqlRepo: mysqlRepo,
		redisRepo: redisRepo,
		logger:    logger,
	}
}

// CreateGroup 创建一个新群组，将群主添加为 role=2（群主）的成员，
// 并更新 Redis 缓存。返回新的 groupID。
func (s *GroupService) CreateGroup(ctx context.Context, ownerID int64, name, notice string) (int64, error) {
	group := &model.Group{
		Name:    name,
		Notice:  notice,
		OwnerID: ownerID,
	}
	groupID, err := s.mysqlRepo.CreateGroup(ctx, group)
	if err != nil {
		return 0, fmt.Errorf("create group in mysql: %w", err)
	}

	// 将群主添加为群组成员，role=2（群主）
	ownerMember := &model.GroupMember{
		GroupID: groupID,
		UserID:  ownerID,
		Role:    2, // 群主
	}
	if err := s.mysqlRepo.AddGroupMember(ctx, ownerMember); err != nil {
		return 0, fmt.Errorf("add owner as group member: %w", err)
	}

	// 更新 Redis 缓存
	if err := s.redisRepo.AddGroupMemberRedis(ctx, groupID, ownerID); err != nil {
		return 0, fmt.Errorf("sync owner membership to redis: %w", err)
	}

	return groupID, nil
}

// UpdateGroup 更新群组名称和公告。仅群主或管理员可以更新。
func (s *GroupService) UpdateGroup(ctx context.Context, userID, groupID int64, name, notice string) error {
	group, err := s.mysqlRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	if group == nil {
		return fmt.Errorf(ErrGroupNotFound)
	}

	// 验证 userID 是否为群主或管理员
	if !s.isOwnerOrAdmin(ctx, groupID, userID) {
		return fmt.Errorf(ErrNotOwnerOrAdmin)
	}

	group.Name = name
	group.Notice = notice
	if err := s.mysqlRepo.UpdateGroup(ctx, group); err != nil {
		return fmt.Errorf("update group: %w", err)
	}
	return nil
}

// GetGroupInfo 通过 ID 返回群组详情。
func (s *GroupService) GetGroupInfo(ctx context.Context, groupID int64) (*model.Group, error) {
	group, err := s.mysqlRepo.GetGroupByID(ctx, groupID)
	if err != nil { // 获取群聊失败
		return nil, fmt.Errorf("get group: %w", err)
	}
	if group == nil { // 没找到这个id对应的群聊
		return nil, fmt.Errorf(ErrGroupNotFound)
	}
	return group, nil
}

// ListUserGroups 获取当前用户所加入的所有群聊
func (s *GroupService) ListUserGroups(ctx context.Context, userID int64) ([]model.Group, error) {
	ids, err := s.redisRepo.GetGroupMemberships(ctx, userID) // 从redis中获取当前用户加入的所有群聊id
	if err != nil {
		return nil, fmt.Errorf("get user groups: %w", err)
	}
	groups := make([]model.Group, 0, len(ids))
	for _, id := range ids {
		group, err := s.mysqlRepo.GetGroupByID(ctx, id)
		if err != nil {
			return nil, fmt.Errorf("get group: %w", err)
		}
		if group != nil {
			groups = append(groups, *group)
		}
	}
	return groups, nil
}

// isOwnerOrAdmin 检查给定的 userID 是否在群组中具有群主（role=2）或管理员（role=1）身份。
// 如果是则返回 true，否则返回 false。
func (s *GroupService) isOwnerOrAdmin(ctx context.Context, groupID, userID int64) bool {
	members, err := s.mysqlRepo.GetGroupMembers(ctx, groupID)
	if err != nil {
		s.logger.Error("获取群组成员以进行权限检查失败", zap.Error(err))
		return false
	}
	for _, m := range members {
		if m.UserID == userID && (m.Role == 1 || m.Role == 2) {
			return true
		}
	}
	return false
}
