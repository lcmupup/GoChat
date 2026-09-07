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

// AddMember 向群组添加新成员。userID 必须是群主或管理员。
// 检查最大成员上限（500）以及 newMemberID 是否已是成员。
func (s *GroupService) AddMember(ctx context.Context, groupID, userID, newMemberID int64) error {
	// 验证群聊是否存在
	group, err := s.mysqlRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	if group == nil {
		return fmt.Errorf(ErrGroupNotFound)
	}

	// 验证操作者是否为群主或管理员
	if !s.isOwnerOrAdmin(ctx, groupID, userID) {
		return fmt.Errorf(ErrNotOwnerOrAdmin)
	}

	// 验证被邀请的用户是否存在
	user, err := s.mysqlRepo.GetUserByID(ctx, newMemberID)
	if err != nil {
		return fmt.Errorf("get member user: %w", err)
	}
	if user == nil {
		return fmt.Errorf(ErrUserNotFound)
	}

	// 验证操作者与被邀请者是好友关系
	isFriend, err := s.mysqlRepo.IsFriend(ctx, userID, newMemberID)
	if err != nil {
		return fmt.Errorf("check member friendship: %w", err)
	}
	if !isFriend {
		return fmt.Errorf(ErrMemberNotFriend)
	}

	// 检查群聊人数是否已满
	members, err := s.mysqlRepo.GetGroupMembers(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get group members: %w", err)
	}
	if len(members) >= maxGroupMembers {
		return fmt.Errorf(ErrGroupFull)
	}

	// 检查被邀请者是否已是群成员
	for _, m := range members {
		if m.UserID == newMemberID {
			return fmt.Errorf(ErrAlreadyMember)
		}
	}

	// 在 MySQL 中添加成员
	newMember := &model.GroupMember{
		GroupID: groupID,
		UserID:  newMemberID,
		Role:    0, // 普通成员
	}
	if err := s.mysqlRepo.AddGroupMember(ctx, newMember); err != nil {
		return fmt.Errorf("add group member: %w", err)
	}

	// 更新 Redis 缓存
	if err := s.redisRepo.AddGroupMemberRedis(ctx, groupID, newMemberID); err != nil {
		// Returning success here would leave the send-time Lua authorization
		// cache stale.  The caller must retry/repair instead of believing the
		// member can already send messages.
		return fmt.Errorf("sync group membership to redis: %w", err)
	}

	return nil
}

// RemoveMember 从群组中移除成员。
// userID 必须是群主/管理员，或者 removeMemberID==userID（自行退出）。
// 群主不能被移除。
func (s *GroupService) RemoveMember(ctx context.Context, groupID, userID, removeMemberID int64) error {
	// 验证群聊是否存在
	group, err := s.mysqlRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	if group == nil {
		return fmt.Errorf(ErrGroupNotFound)
	}

	// 查询所有群成员从中找出操作者（actor）和被移除者（target）
	members, err := s.mysqlRepo.GetGroupMembers(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get group members: %w", err)
	}
	var actor, target *model.GroupMember
	for i := range members {
		if members[i].UserID == userID {
			actor = &members[i]
		}
		if members[i].UserID == removeMemberID {
			target = &members[i]
		}
	}
	if actor == nil { // 操作者不是群成员
		return fmt.Errorf(ErrNotOwnerOrAdmin)
	}
	if target == nil { // 被移除者不是群成员
		return fmt.Errorf(ErrMemberNotFound)
	}

	// 操作者和被移除者不是同一个人，说明这是踢人操作，需检查操作者权限
	if userID != removeMemberID {
		if actor.Role != 1 && actor.Role != 2 { // 操作者不是管理员或群主
			return fmt.Errorf(ErrNotOwnerOrAdmin)
		}
		if actor.Role == 1 && target.Role != 0 { // 管理员只能移除普通成员
			return fmt.Errorf(ErrCannotRemovePeer)
		}
	}

	// 被移除者不能是群主
	if target.Role == 2 || removeMemberID == group.OwnerID {
		return fmt.Errorf(ErrCannotRemoveOwner)
	}

	// 先移除 Redis 中的群成员信息，再移除 MySQL 中的群成员信息
	// 只要 Redis 中的群成员信息被删除，这个人就已经不能在群内发消息了，即使后面的 MySQL 群成员信息删除失败也不影响最终的效果
	if err := s.redisRepo.RemoveGroupMemberRedis(ctx, groupID, removeMemberID); err != nil {
		return fmt.Errorf("sync group member removal to redis: %w", err)
	}
	if err := s.mysqlRepo.RemoveGroupMember(ctx, groupID, removeMemberID); err != nil {
		return fmt.Errorf("remove group member after redis authorization removal: %w", err)
	}

	return nil
}

// LeaveGroup 允许用户退出群组。群主不能退出（必须先转让群主或解散群组）。
func (s *GroupService) LeaveGroup(ctx context.Context, groupID, userID int64) error {
	// 验证群聊是否存在
	group, err := s.mysqlRepo.GetGroupByID(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get group: %w", err)
	}
	if group == nil {
		return fmt.Errorf(ErrGroupNotFound)
	}

	// 群主不能退出
	if userID == group.OwnerID {
		return fmt.Errorf(ErrCannotLeaveAsOwner)
	}

	// 在该群的群成员中找到自己
	members, err := s.mysqlRepo.GetGroupMembers(ctx, groupID)
	if err != nil {
		return fmt.Errorf("get group members: %w", err)
	}
	found := false
	for _, member := range members {
		if member.UserID == userID {
			found = true
			break
		}
	}
	if !found { // 找不到说明该用户不在这个群聊中
		return fmt.Errorf(ErrMemberNotFound)
	}

	// 先移除 Redis 中的群成员信息，再移除 MySQL 中的群成员信息
	if err := s.redisRepo.RemoveGroupMemberRedis(ctx, groupID, userID); err != nil {
		return fmt.Errorf("sync group leave to redis: %w", err)
	}
	if err := s.mysqlRepo.RemoveGroupMember(ctx, groupID, userID); err != nil {
		return fmt.Errorf("leave group after redis authorization removal: %w", err)
	}

	return nil
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
