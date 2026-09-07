package model

import "time"

type Group struct {
	ID         int64     `json:"id"`          // 群id
	Name       string    `json:"name"`        // 群名称
	Notice     string    `json:"notice"`      // 群公告
	OwnerID    int64     `json:"owner_id"`    // 群主id
	MaxMembers int       `json:"max_members"` // 最大群成员人数
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type GroupMember struct {
	ID         int64      `json:"id"`                    // 仅用来标记行编号
	GroupID    int64      `json:"group_id"`              // 加入的群id
	UserID     int64      `json:"user_id"`               // 用户id
	Role       int        `json:"role"`                  // 0=普通成员, 1=管理员, 2=群主
	MutedUntil *time.Time `json:"muted_until,omitempty"` // 禁言截止时间
	JoinedAt   time.Time  `json:"joined_at"`             // 什么时候加的群
}
