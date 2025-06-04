package models

import (
	"time"

	"gorm.io/gorm"
)

// User 用户表（已存在）
type User struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Username    string    `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Password    string    `gorm:"size:100;not null" json:"-"`
	Email       string    `gorm:"size:50;uniqueIndex;not null" json:"email"`
	Bio         string    `gorm:"size:2000" json:"bio"`
	AvatarURL   string    `gorm:"size:255" json:"avatarUrl"`
	CreatedTime time.Time `gorm:"type:datetime;default:CURRENT_TIMESTAMP;not null" json:"createdTime"`
	Status      string    `gorm:"type:enum('ACTIVE','DISABLED','DELETED');default:'ACTIVE'" json:"status"`
}

// ChatRoom 聊天室表
type ChatRoom struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string         `gorm:"size:100;not null" json:"name"`
	Description string         `gorm:"size:1000" json:"description"`
	Category    string         `gorm:"size:50;not null" json:"category"` // 技术、艺术、运动等
	CreatorID   int64          `gorm:"not null" json:"creatorId"`
	Creator     User           `gorm:"foreignKey:CreatorID" json:"creator"`
	MaxMembers  int            `gorm:"default:100" json:"maxMembers"`
	IsActive    bool           `gorm:"default:true" json:"isActive"`
	IsApproved  bool           `gorm:"default:false" json:"isApproved"` // 需要审核
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联
	Members  []ChatRoomMember `gorm:"foreignKey:ChatRoomID" json:"members,omitempty"`
	Messages []Message        `gorm:"foreignKey:ChatRoomID" json:"messages,omitempty"`
	CheckIns []CheckIn        `gorm:"foreignKey:ChatRoomID" json:"checkIns,omitempty"`
}

// ChatRoomMember 聊天室成员表
type ChatRoomMember struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatRoomID int64     `gorm:"not null" json:"chatRoomId"`
	UserID     int64     `gorm:"not null" json:"userId"`
	Role       string    `gorm:"type:enum('MEMBER','ADMIN','OWNER');default:'MEMBER'" json:"role"`
	IsMuted    bool      `gorm:"default:false" json:"isMuted"`
	JoinedAt   time.Time `json:"joinedAt"`
	UpdatedAt  time.Time `json:"updatedAt"`

	// 关联
	ChatRoom ChatRoom `gorm:"foreignKey:ChatRoomID" json:"-"` // Prevent ChatRoom from being serialized to avoid circular dependency
	User     User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Message 群聊消息表（持久化存储）
type Message struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatRoomID int64     `gorm:"not null;index" json:"chatRoomId"`
	UserID     int64     `gorm:"not null;index" json:"userId"`
	Content    string    `gorm:"type:text;not null" json:"content"`
	CreatedAt  time.Time `gorm:"index" json:"createdAt"`

	// 群聊消息持久化存储，不维护已读未读状态
}

// PrivateMessage 私聊消息表（持久化存储）
type PrivateMessage struct {
	ID         int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	SenderID   int64          `gorm:"not null;index" json:"senderId"`
	ReceiverID int64          `gorm:"not null;index" json:"receiverId"`
	Content    string         `gorm:"type:text;not null" json:"content"`
	CreatedAt  time.Time      `gorm:"index" json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联字段已移除，减少数据传输冗余
	// 如需用户信息，请通过 SenderID 和 ReceiverID 单独查询
}

// ConversationUnreadCount 会话未读消息计数表
type ConversationUnreadCount struct {
	ID             int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ConversationID int64     `gorm:"not null;index" json:"conversationId"`
	UserID         int64     `gorm:"not null;index" json:"userId"`
	UnreadCount    int64     `gorm:"default:0" json:"unreadCount"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`

	// 关联
	Conversation Conversation `gorm:"foreignKey:ConversationID" json:"conversation,omitempty"`
	User         User         `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// Admin 管理员表
type Admin struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"uniqueIndex;not null" json:"userId"`
	Role      string    `gorm:"type:enum('SUPER_ADMIN','MODERATOR');default:'MODERATOR'" json:"role"`
	IsActive  bool      `gorm:"default:true" json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	// 关联
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// CheckIn 打卡表
type CheckIn struct {
	ID         int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatRoomID int64     `gorm:"not null;index" json:"chatRoomId"`
	UserID     int64     `gorm:"not null;index" json:"userId"`
	Content    string    `gorm:"size:500" json:"content"`
	CheckDate  time.Time `gorm:"type:date;not null;index" json:"checkDate"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`

	// 关联
	ChatRoom ChatRoom `gorm:"foreignKey:ChatRoomID" json:"-"` // Prevent ChatRoom from being serialized to avoid circular dependency
	User     User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// CheckInTask 打卡任务表
type CheckInTask struct {
	ID          int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	ChatRoomID  int64      `gorm:"not null;index" json:"chatRoomId"`
	Title       string     `gorm:"size:200;not null" json:"title"`
	Description string     `gorm:"size:1000" json:"description"`
	Cycle       string     `gorm:"type:enum('DAILY','WEEKLY','MONTHLY');default:'DAILY'" json:"cycle"`
	IsActive    bool       `gorm:"default:true" json:"isActive"`
	StartDate   time.Time  `gorm:"type:date;not null" json:"startDate"`
	EndDate     *time.Time `gorm:"type:date" json:"endDate"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`

	// 关联
	ChatRoom ChatRoom `gorm:"foreignKey:ChatRoomID" json:"-"` // Prevent ChatRoom from being serialized to avoid circular dependency
}

// Conversation 会话表（用于私聊会话管理）
type Conversation struct {
	ID              int64      `gorm:"primaryKey;autoIncrement" json:"id"`
	User1ID         int64      `gorm:"not null;index" json:"user1Id"`
	User2ID         int64      `gorm:"not null;index" json:"user2Id"`
	LastMessageID   *int64     `gorm:"index" json:"lastMessageId"`
	LastMessageTime *time.Time `json:"lastMessageTime"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`

	// 关联字段已移除，减少数据传输冗余
	// 如需用户信息，请通过 User1ID 和 User2ID 单独查询
	// 如需最后一条消息详情，请通过 LastMessageID 单独查询
	LastMessage *PrivateMessage `gorm:"foreignKey:LastMessageID" json:"lastMessage,omitempty"`
}

// TableName 设置表名
func (User) TableName() string {
	return "user"
}

func (ChatRoom) TableName() string {
	return "chatroom"
}

func (ChatRoomMember) TableName() string {
	return "chatroom_member"
}

func (Message) TableName() string {
	return "message"
}

func (PrivateMessage) TableName() string {
	return "private_message"
}

func (Conversation) TableName() string {
	return "conversation"
}

func (Admin) TableName() string {
	return "admin"
}

func (CheckIn) TableName() string {
	return "checkin"
}

func (CheckInTask) TableName() string {
	return "checkin_task"
}

func (ConversationUnreadCount) TableName() string {
	return "conversation_unread_count"
}
