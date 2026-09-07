package repository

import (
	"context"
	"database/sql"
	"fmt"
	"gochat/internal/model"
)

// MySQLRepo 定义了服务和消费者所需的所有 MySQL CRUD 操作。
type MySQLRepo interface {
	// —— 用户 ——
	GetUserByID(ctx context.Context, userID int64) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	CreateUser(ctx context.Context, user *model.User) error
	UpdateUser(ctx context.Context, user *model.User) error

	// ── 好友 ──
	CreateFriendRequest(ctx context.Context, req *model.FriendRequest) error
	UpdateFriendRequest(ctx context.Context, req *model.FriendRequest) error
	GetFriendRequestByID(ctx context.Context, id int64) (*model.FriendRequest, error)
	GetFriendRequestsByUser(ctx context.Context, userID int64) ([]model.FriendRequest, error)
	CreateFriendship(ctx context.Context, fs *model.Friendship) error
	DeleteFriendship(ctx context.Context, userID, friendID int64) error
	GetFriendList(ctx context.Context, userID int64) ([]model.Friendship, error)
	IsFriend(ctx context.Context, userID, friendID int64) (bool, error)
	CreateBlacklist(ctx context.Context, bl *model.Blacklist) error
	DeleteBlacklist(ctx context.Context, userID, blockedID int64) error
	IsBlocked(ctx context.Context, userID, blockedID int64) (bool, error)

	// ── 群组 ──
	CreateGroup(ctx context.Context, group *model.Group) (int64, error)
	UpdateGroup(ctx context.Context, group *model.Group) error
	GetGroupByID(ctx context.Context, groupID int64) (*model.Group, error)
	AddGroupMember(ctx context.Context, member *model.GroupMember) error
	RemoveGroupMember(ctx context.Context, groupID, userID int64) error
	GetGroupMembers(ctx context.Context, groupID int64) ([]model.GroupMember, error)
	UpdateGroupMemberRole(ctx context.Context, groupID, userID, role int) error
}

// MySQLRepoImpl — 基于 database/sql 的具体实现
type MySQLRepoImpl struct {
	db *sql.DB
}

func NewMySQLRepo(db *sql.DB) *MySQLRepoImpl {
	return &MySQLRepoImpl{db}
}

// —— 用户 ——
func (m *MySQLRepoImpl) GetUserByID(ctx context.Context, userID int64) (*model.User, error) {
	query := `SELECT id, username, password_hash, nickname, avatar_url, sign, gender, created_at, updated_at
	          FROM users WHERE id = ?`
	row := m.db.QueryRowContext(ctx, query, userID)
	var u model.User
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Nickname, &u.AvatarURL, &u.Sign, &u.Gender, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("按ID获取用户: %w", err)
	}
	return &u, nil
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

func (m *MySQLRepoImpl) UpdateUser(ctx context.Context, user *model.User) error {
	query := `UPDATE users SET username=?, password_hash=?, nickname=?, avatar_url=?, sign=?, gender=? WHERE id=?`
	_, err := m.db.ExecContext(ctx, query, user.Username, user.PasswordHash, user.Nickname, user.AvatarURL, user.Sign, user.Gender, user.ID)
	if err != nil {
		return fmt.Errorf("更新用户: %w", err)
	}
	return nil
}

// ── 好友 ──

func (m *MySQLRepoImpl) CreateFriendRequest(ctx context.Context, req *model.FriendRequest) error {
	query := `INSERT INTO friend_requests (from_user_id, to_user_id, message, status)
	          VALUES (?, ?, ?, ?)
	          ON DUPLICATE KEY UPDATE
	              id = LAST_INSERT_ID(id),
	              message = IF(status = 0, message, VALUES(message)),
	              created_at = IF(status = 0, created_at, NOW()),
	              updated_at = IF(status = 0, updated_at, NOW()),
	              status = IF(status = 0, status, VALUES(status))`
	result, err := m.db.ExecContext(ctx, query,
		req.FromUserID,
		req.ToUserID,
		req.Message,
		req.Status,
	)
	if err != nil {
		return fmt.Errorf("插入 friend_requests: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("插入 friend_requests 获取最后插入ID: %w", err)
	}
	req.ID = id
	return nil
}

// 处理好友请求：拒绝/同意好友请求
func (m *MySQLRepoImpl) UpdateFriendRequest(ctx context.Context, req *model.FriendRequest) error {
	query := `UPDATE friend_requests SET status=?, updated_at=NOW() WHERE id=?`
	_, err := m.db.ExecContext(ctx, query, req.Status, req.ID)
	if err != nil {
		return fmt.Errorf("更新 friend_requests: %w", err)
	}
	return nil
}

// 根据请求id获取好友请求
func (m *MySQLRepoImpl) GetFriendRequestByID(ctx context.Context, id int64) (*model.FriendRequest, error) {
	query := `SELECT id, from_user_id, to_user_id, message, status, created_at, updated_at
	          FROM friend_requests WHERE id = ?`
	row := m.db.QueryRowContext(ctx, query, id)
	var r model.FriendRequest
	err := row.Scan(&r.ID, &r.FromUserID, &r.ToUserID, &r.Message, &r.Status, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows { // 没找到这个请求id的好友请求
			return nil, nil
		}
		// Scan方法失败了
		return nil, fmt.Errorf("按ID获取好友请求: %w", err)
	}
	return &r, nil
}

// 根据用户id获取好友请求
func (m *MySQLRepoImpl) GetFriendRequestsByUser(ctx context.Context, userID int64) ([]model.FriendRequest, error) {
	query := `SELECT id, from_user_id, to_user_id, message, status, created_at, updated_at
	          FROM friend_requests
	          WHERE to_user_id = ? AND status = 0
	          ORDER BY created_at DESC`
	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("按用户获取好友请求: %w", err)
	}
	defer rows.Close()

	var results []model.FriendRequest
	for rows.Next() {
		var r model.FriendRequest
		if err := rows.Scan(&r.ID, &r.FromUserID, &r.ToUserID, &r.Message, &r.Status, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("扫描好友请求: %w", err)
		}
		results = append(results, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历好友请求: %w", err)
	}
	return results, nil
}

// 建立好友关系
func (m *MySQLRepoImpl) CreateFriendship(ctx context.Context, fs *model.Friendship) error {
	// 插入双向记录：user->friend 和 friend->user
	query := `INSERT INTO friendships (user_id, friend_id) VALUES (?, ?)`
	// 插入user->friend
	_, err := m.db.ExecContext(ctx, query, fs.UserID, fs.FriendID)
	if err != nil {
		return fmt.Errorf("插入好友关系 user->friend: %w", err)
	}
	// 插入friend->user
	_, err = m.db.ExecContext(ctx, query, fs.FriendID, fs.UserID)
	if err != nil {
		return fmt.Errorf("插入好友关系 friend->user: %w", err)
	}
	return nil
}

// 删除好友关系
func (m *MySQLRepoImpl) DeleteFriendship(ctx context.Context, userID, friendID int64) error {
	// 删除双向记录：user->friend 和 friend->user
	query := `DELETE FROM friendships WHERE (user_id = ? AND friend_id = ?) OR (user_id = ? AND friend_id = ?)`
	_, err := m.db.ExecContext(ctx, query, userID, friendID, friendID, userID)
	if err != nil {
		return fmt.Errorf("删除好友关系: %w", err)
	}
	return nil
}

// 获取好友列表
func (m *MySQLRepoImpl) GetFriendList(ctx context.Context, userID int64) ([]model.Friendship, error) {
	query := `SELECT f.id, f.user_id, f.friend_id, f.created_at
	          FROM friendships f
	          WHERE f.user_id = ?`
	rows, err := m.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("获取好友列表: %w", err)
	}
	defer rows.Close()

	var results []model.Friendship
	for rows.Next() {
		var fs model.Friendship
		if err := rows.Scan(&fs.ID, &fs.UserID, &fs.FriendID, &fs.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描好友关系: %w", err)
		}
		results = append(results, fs)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历好友关系: %w", err)
	}
	return results, nil
}

// 判断两人是否是朋友关系
func (m *MySQLRepoImpl) IsFriend(ctx context.Context, userID, friendID int64) (bool, error) {
	query := `SELECT COUNT(*) FROM friendships WHERE user_id = ? AND friend_id = ?`
	var count int
	err := m.db.QueryRowContext(ctx, query, userID, friendID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查是否为好友: %w", err)
	}
	return count > 0, nil
}

// 创建一条拉黑记录
func (m *MySQLRepoImpl) CreateBlacklist(ctx context.Context, bl *model.Blacklist) error {
	query := `INSERT INTO blacklist (user_id, blocked_id) VALUES (?, ?)`
	result, err := m.db.ExecContext(ctx, query, bl.UserID, bl.BlockedID)
	if err != nil {
		return fmt.Errorf("插入黑名单: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("插入黑名单 获取最后插入ID: %w", err)
	}
	bl.ID = id
	return nil
}

// 删除一条拉黑记录
func (m *MySQLRepoImpl) DeleteBlacklist(ctx context.Context, userID, blockedID int64) error {
	query := `DELETE FROM blacklist WHERE user_id = ? AND blocked_id = ?`
	_, err := m.db.ExecContext(ctx, query, userID, blockedID)
	if err != nil {
		return fmt.Errorf("删除黑名单: %w", err)
	}
	return nil
}

// 判断是否拉黑了对方或被对方拉黑
func (m *MySQLRepoImpl) IsBlocked(ctx context.Context, userID, blockedID int64) (bool, error) {
	query := `SELECT COUNT(*) FROM blacklist WHERE user_id = ? AND blocked_id = ?`
	var count int
	err := m.db.QueryRowContext(ctx, query, userID, blockedID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查是否已拉黑: %w", err)
	}
	return count > 0, nil
}

// ── 群组 ──

func (m *MySQLRepoImpl) CreateGroup(ctx context.Context, group *model.Group) (int64, error) {
	query := "INSERT INTO `groups` (name, notice, owner_id, max_members, created_at, updated_at) VALUES (?, ?, ?, 500, NOW(), NOW())"
	result, err := m.db.ExecContext(ctx, query,
		group.Name,
		group.Notice,
		group.OwnerID,
	)
	if err != nil {
		return 0, fmt.Errorf("创建群组: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("创建群组 获取最后插入ID: %w", err)
	}
	return id, nil
}

func (m *MySQLRepoImpl) UpdateGroup(ctx context.Context, group *model.Group) error {
	query := "UPDATE `groups` SET name=?, notice=?, owner_id=?, updated_at=NOW() WHERE id=?"
	_, err := m.db.ExecContext(ctx, query, group.Name, group.Notice, group.OwnerID, group.ID)
	if err != nil {
		return fmt.Errorf("更新群组: %w", err)
	}
	return nil
}

func (m *MySQLRepoImpl) GetGroupByID(ctx context.Context, groupID int64) (*model.Group, error) {
	query := "SELECT id, name, notice, owner_id, max_members, created_at, updated_at FROM `groups` WHERE id = ?"
	row := m.db.QueryRowContext(ctx, query, groupID)
	var g model.Group
	err := row.Scan(&g.ID, &g.Name, &g.Notice, &g.OwnerID, &g.MaxMembers, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		// 找不到这个群
		if err == sql.ErrNoRows {
			return nil, nil
		}
		// 查这个群产生了错误
		return nil, fmt.Errorf("按ID获取群组: %w", err)
	}
	return &g, nil
}

func (m *MySQLRepoImpl) AddGroupMember(ctx context.Context, member *model.GroupMember) error {
	query := `INSERT INTO group_members (group_id, user_id, role, muted_until, joined_at)
	          VALUES (?, ?, ?, ?, NOW())`
	_, err := m.db.ExecContext(ctx, query,
		member.GroupID,
		member.UserID,
		member.Role,
		member.MutedUntil,
	)
	if err != nil {
		return fmt.Errorf("添加群成员: %w", err)
	}
	return nil
}

func (m *MySQLRepoImpl) RemoveGroupMember(ctx context.Context, groupID, userID int64) error {
	query := `DELETE FROM group_members WHERE group_id=? AND user_id=?`
	_, err := m.db.ExecContext(ctx, query, groupID, userID)
	if err != nil {
		return fmt.Errorf("移除群成员: %w", err)
	}
	return nil
}

func (m *MySQLRepoImpl) GetGroupMembers(ctx context.Context, groupID int64) ([]model.GroupMember, error) {
	query := `SELECT id, group_id, user_id, role, muted_until, joined_at
	          FROM group_members WHERE group_id = ?`
	rows, err := m.db.QueryContext(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("获取群成员: %w", err)
	}
	defer rows.Close()

	members := make([]model.GroupMember, 0)
	for rows.Next() {
		var gm model.GroupMember
		err := rows.Scan(&gm.ID, &gm.GroupID, &gm.UserID, &gm.Role, &gm.MutedUntil, &gm.JoinedAt)
		if err != nil {
			return nil, fmt.Errorf("扫描群成员: %w", err)
		}
		members = append(members, gm)
	}
	// 检查 rows 是否被正常遍历完毕
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历群成员: %w", err)
	}
	return members, nil
}

func (m *MySQLRepoImpl) UpdateGroupMemberRole(ctx context.Context, groupID, userID, role int) error {
	query := `UPDATE group_members SET role=? WHERE group_id=? AND user_id=?`
	_, err := m.db.ExecContext(ctx, query, role, groupID, userID)
	if err != nil {
		return fmt.Errorf("更新群成员角色: %w", err)
	}
	return nil
}
