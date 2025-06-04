package services

import (
	"campus-canvas-chat/database"
	"campus-canvas-chat/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type ChatRoomService struct {
	db *gorm.DB
}

func NewChatRoomService() *ChatRoomService {
	return &ChatRoomService{
		db: database.GetDB(),
	}
}

// CreateChatRoom 创建聊天室
func (s *ChatRoomService) CreateChatRoom(room *models.ChatRoom) error {
	// 检查创建者是否存在
	var user models.User
	if err := s.db.First(&user, room.CreatorID).Error; err != nil {
		return errors.New("创建者不存在")
	}

	// 创建聊天室
	if err := s.db.Create(room).Error; err != nil {
		return err
	}

	// 将创建者添加为房主
	member := &models.ChatRoomMember{
		ChatRoomID: room.ID,
		UserID:     room.CreatorID,
		Role:       "OWNER",
		JoinedAt:   time.Now(),
	}

	return s.db.Create(member).Error
}

// GetChatRoomList 获取聊天室列表
func (s *ChatRoomService) GetChatRoomList(page, pageSize int, category string, isApproved bool) ([]models.ChatRoom, int64, error) {
	var rooms []models.ChatRoom
	var total int64

	query := s.db.Model(&models.ChatRoom{}).Where("is_active = ?", true)

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if isApproved {
		query = query.Where("is_approved = ?", true)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Preload("Creator").Offset(offset).Limit(pageSize).Find(&rooms).Error

	return rooms, total, err
}

// GetChatRoomByID 根据ID获取聊天室详情
func (s *ChatRoomService) GetChatRoomByID(roomID int64) (*models.ChatRoom, error) {
	var room models.ChatRoom
	err := s.db.Preload("Creator").Preload("Members", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, chat_room_id, user_id, role, is_muted, joined_at, updated_at")
	}).Preload("Members.User").First(&room, roomID).Error
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// JoinChatRoom 加入聊天室
func (s *ChatRoomService) JoinChatRoom(roomID, userID int64) error {
	// 检查聊天室是否存在且已审核
	var room models.ChatRoom
	if err := s.db.Where("id = ? AND is_active = ? AND is_approved = ?", roomID, true, true).First(&room).Error; err != nil {
		return errors.New("聊天室不存在或未审核")
	}

	// 检查用户是否已经是成员
	var existingMember models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, userID).First(&existingMember).Error; err == nil {
		return errors.New("用户已经是该聊天室成员")
	}

	// 检查房间人数限制
	var memberCount int64
	s.db.Model(&models.ChatRoomMember{}).Where("chat_room_id = ?", roomID).Count(&memberCount)
	if int(memberCount) >= room.MaxMembers {
		return errors.New("聊天室人数已满")
	}

	// 添加成员
	member := &models.ChatRoomMember{
		ChatRoomID: roomID,
		UserID:     userID,
		Role:       "MEMBER",
		JoinedAt:   time.Now(),
	}

	return s.db.Create(member).Error
}

// LeaveChatRoom 离开聊天室
func (s *ChatRoomService) LeaveChatRoom(roomID, userID int64) error {
	// 检查用户是否是房主
	var member models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, userID).First(&member).Error; err != nil {
		return errors.New("用户不是该聊天室成员")
	}

	if member.Role == "OWNER" {
		return errors.New("房主不能离开聊天室，请先转让房主权限或删除聊天室")
	}

	return s.db.Delete(&member).Error
}

// DeleteChatRoom 删除聊天室（仅房主可操作）
func (s *ChatRoomService) DeleteChatRoom(roomID, userID int64) error {
	// 检查用户是否是房主
	var member models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ? AND role = ?", roomID, userID, "OWNER").First(&member).Error; err != nil {
		return errors.New("只有房主可以删除聊天室")
	}

	// 软删除聊天室
	return s.db.Model(&models.ChatRoom{}).Where("id = ?", roomID).Updates(map[string]interface{}{
		"is_active":  false,
		"deleted_at": time.Now(),
	}).Error
}

// ApproveChatRoom 审核聊天室（管理员操作）
func (s *ChatRoomService) ApproveChatRoom(roomID int64, approved bool) error {
	return s.db.Model(&models.ChatRoom{}).Where("id = ?", roomID).Update("is_approved", approved).Error
}

// GetUserChatRooms 获取用户加入的聊天室列表
func (s *ChatRoomService) GetUserChatRooms(userID int64) ([]models.ChatRoom, error) {
	var rooms []models.ChatRoom

	err := s.db.Table("chat_room").Select("chat_room.*").
		Joins("JOIN chat_room_member ON chat_room.id = chat_room_member.chat_room_id").
		Where("chat_room_member.user_id = ? AND chat_room.is_active = ? AND chat_room.is_approved = ?", userID, true, true).
		Preload("Creator").
		Find(&rooms).Error

	return rooms, err
}

// UpdateMemberRole 更新成员角色（房主和管理员可操作）
func (s *ChatRoomService) UpdateMemberRole(roomID, operatorID, targetUserID int64, newRole string) error {
	// 检查操作者不能修改自己的角色
	if operatorID == targetUserID {
		return errors.New("不能修改自己的角色")
	}

	// 检查角色是否有效
	validRoles := map[string]bool{
		"OWNER":  true,
		"ADMIN":  true,
		"MEMBER": true,
	}
	if !validRoles[newRole] {
		return errors.New("无效的角色类型")
	}

	// 检查操作者权限
	var operatorMember models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, operatorID).First(&operatorMember).Error; err != nil {
		return errors.New("操作者不是该聊天室成员")
	}

	if operatorMember.Role != "OWNER" && operatorMember.Role != "ADMIN" {
		return errors.New("权限不足")
	}

	// 获取目标用户信息
	var targetMember models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, targetUserID).First(&targetMember).Error; err != nil {
		return errors.New("目标用户不是该聊天室成员")
	}

	// 管理员不能修改其他管理员的角色
	if operatorMember.Role == "ADMIN" && targetMember.Role == "ADMIN" {
		return errors.New("管理员不能修改其他管理员的角色")
	}

	// 只有房主可以设置或取消管理员角色
	if newRole == "ADMIN" && operatorMember.Role != "OWNER" {
		return errors.New("只有房主可以设置管理员")
	}

	// 不能修改房主角色
	if targetMember.Role == "OWNER" {
		return errors.New("不能修改房主的角色")
	}

	// 更新目标用户角色
	return s.db.Model(&models.ChatRoomMember{}).
		Where("chat_room_id = ? AND user_id = ?", roomID, targetUserID).
		Update("role", newRole).Error
}

// MuteMember 禁言成员
func (s *ChatRoomService) MuteMember(roomID, operatorID, targetUserID int64, muted bool) error {
	// 检查操作者权限
	var operatorMember models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, operatorID).First(&operatorMember).Error; err != nil {
		return errors.New("操作者不是该聊天室成员")
	}

	if operatorMember.Role != "OWNER" && operatorMember.Role != "ADMIN" {
		return errors.New("权限不足")
	}

	// 获取目标用户信息
	var targetMember models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, targetUserID).First(&targetMember).Error; err != nil {
		return errors.New("目标用户不是该聊天室成员")
	}

	// 不能禁言房主
	if targetMember.Role == "OWNER" {
		return errors.New("不能禁言房主")
	}

	// 管理员不能禁言其他管理员
	if operatorMember.Role == "ADMIN" && targetMember.Role == "ADMIN" {
		return errors.New("管理员不能禁言其他管理员")
	}

	// 更新禁言状态
	return s.db.Model(&models.ChatRoomMember{}).
		Where("chat_room_id = ? AND user_id = ?", roomID, targetUserID).
		Update("is_muted", muted).Error
}

// KickMember 踢出成员
func (s *ChatRoomService) KickMember(roomID, operatorID, targetUserID int64) error {
	// 检查操作者权限
	var operatorMember models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, operatorID).First(&operatorMember).Error; err != nil {
		return errors.New("操作者不是该聊天室成员")
	}

	if operatorMember.Role != "OWNER" && operatorMember.Role != "ADMIN" {
		return errors.New("权限不足")
	}

	// 检查目标用户
	var targetMember models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", roomID, targetUserID).First(&targetMember).Error; err != nil {
		return errors.New("目标用户不是该聊天室成员")
	}

	if targetMember.Role == "OWNER" {
		return errors.New("不能踢出房主")
	}

	// 删除成员
	return s.db.Delete(&targetMember).Error
}
