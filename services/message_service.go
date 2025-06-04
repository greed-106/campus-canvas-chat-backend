package services

import (
	"campus-canvas-chat/database"
	"campus-canvas-chat/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type MessageService struct {
	db *gorm.DB
}

func NewMessageService() *MessageService {
	return &MessageService{
		db: database.GetDB(),
	}
}

// SendGroupMessage 发送群聊消息（持久化存储）
func (s *MessageService) SendGroupMessage(chatRoomID, userID int64, content string) (*models.Message, error) {
	// 检查用户是否是聊天室成员且未被禁言
	var member models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", chatRoomID, userID).First(&member).Error; err != nil {
		return nil, errors.New("用户不是该聊天室成员")
	}

	if member.IsMuted {
		return nil, errors.New("用户已被禁言")
	}

	// 获取用户名
	var username string
	if err := s.db.Model(&models.User{}).Where("id = ?", userID).Select("username").Scan(&username).Error; err != nil {
		return nil, errors.New("用户不存在")
	}

	// 创建并保存消息到数据库
	message := &models.Message{
		ChatRoomID: chatRoomID,
		UserID:     userID,
		Content:    content,
		CreatedAt:  time.Now(),
	}

	if err := s.db.Create(message).Error; err != nil {
		return nil, errors.New("发送消息失败")
	}

	return message, nil
}

// GetGroupMessages 获取群聊消息列表
func (s *MessageService) GetGroupMessages(chatRoomID int64, page, pageSize int) ([]models.Message, int64, error) {
	var messages []models.Message
	var total int64

	// 检查聊天室是否存在
	var chatRoom models.ChatRoom
	if err := s.db.First(&chatRoom, chatRoomID).Error; err != nil {
		return nil, 0, errors.New("聊天室不存在")
	}

	// 获取总数
	if err := s.db.Model(&models.Message{}).Where("chat_room_id = ?", chatRoomID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询消息
	offset := (page - 1) * pageSize
	err := s.db.Where("chat_room_id = ?", chatRoomID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&messages).Error

	return messages, total, err
}

// SendPrivateMessage 发送私聊消息（持久化存储）
func (s *MessageService) SendPrivateMessage(senderID, receiverID int64, content string) (*models.PrivateMessage, error) {
	// 检查发送者和接收者是否存在
	var sender, receiver models.User
	if err := s.db.First(&sender, senderID).Error; err != nil {
		return nil, errors.New("发送者不存在")
	}
	if err := s.db.First(&receiver, receiverID).Error; err != nil {
		return nil, errors.New("接收者不存在")
	}

	// 创建私聊消息
	message := &models.PrivateMessage{
		SenderID:   senderID,
		ReceiverID: receiverID,
		Content:    content,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	// 保存到数据库
	if err := s.db.Create(message).Error; err != nil {
		return nil, err
	}

	// 重新查询消息以获取最新数据
	if err := s.db.First(message, message.ID).Error; err != nil {
		return nil, err
	}

	// 更新或创建会话记录
	conversationID := s.updateConversation(senderID, receiverID, message.ID, message.CreatedAt)

	// 增加接收者在该会话的未读消息计数
	if conversationID > 0 {
		s.incrementConversationUnreadCount(conversationID, receiverID)
	}

	return message, nil
}

// updateConversation 更新会话记录，返回会话ID
func (s *MessageService) updateConversation(user1ID, user2ID, messageID int64, messageTime time.Time) int64 {
	// 确保user1ID < user2ID，保持会话记录的一致性
	if user1ID > user2ID {
		user1ID, user2ID = user2ID, user1ID
	}

	// 查找现有会话
	var conversation models.Conversation
	err := s.db.Where("user1_id = ? AND user2_id = ?", user1ID, user2ID).First(&conversation).Error

	if err != nil {
		// 创建新会话
		conversation = models.Conversation{
			User1ID:         user1ID,
			User2ID:         user2ID,
			LastMessageID:   &messageID,
			LastMessageTime: &messageTime,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		}
		if err := s.db.Create(&conversation).Error; err != nil {
			return 0
		}
	} else {
		// 更新现有会话
		s.db.Model(&conversation).Updates(map[string]interface{}{
			"last_message_id":   messageID,
			"last_message_time": messageTime,
			"updated_at":        time.Now(),
		})
	}

	return conversation.ID
}

// GetPrivateMessages 获取私聊消息列表
func (s *MessageService) GetPrivateMessages(user1ID, user2ID int64, page, pageSize int) ([]models.PrivateMessage, int64, error) {
	var messages []models.PrivateMessage
	var total int64

	// 构建查询条件：双向消息
	query := s.db.Model(&models.PrivateMessage{}).Where(
		"(sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)",
		user1ID, user2ID, user2ID, user1ID,
	)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，按时间倒序
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&messages).Error

	return messages, total, err
}

// ConversationWithUnreadCount 会话信息包含未读计数
type ConversationWithUnreadCount struct {
	models.Conversation
	UnreadCount int64 `json:"unreadCount"`
}

// GetConversations 获取用户的所有会话列表（包含未读计数）
func (s *MessageService) GetConversations(userID int64) ([]ConversationWithUnreadCount, error) {
	var conversations []models.Conversation

	err := s.db.Where("user1_id = ? OR user2_id = ?", userID, userID).
		Preload("LastMessage").
		Order("last_message_time DESC").
		Find(&conversations).Error

	if err != nil {
		return nil, err
	}

	// 为每个会话添加未读计数
	var result []ConversationWithUnreadCount
	for _, conv := range conversations {
		unreadCount, _ := s.GetConversationUnreadCount(conv.ID, userID)
		result = append(result, ConversationWithUnreadCount{
			Conversation: conv,
			UnreadCount:  unreadCount,
		})
	}

	return result, nil
}

// GetConversationUnreadCount 获取指定会话的未读消息数量
func (s *MessageService) GetConversationUnreadCount(conversationID, userID int64) (int64, error) {
	var unreadCount models.ConversationUnreadCount
	err := s.db.Where("conversation_id = ? AND user_id = ?", conversationID, userID).First(&unreadCount).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, err
	}
	return unreadCount.UnreadCount, nil
}

// GetUserTotalUnreadCount 获取用户所有会话的未读消息总数
func (s *MessageService) GetUserTotalUnreadCount(userID int64) (int64, error) {
	var totalCount int64
	err := s.db.Model(&models.ConversationUnreadCount{}).
		Where("user_id = ?", userID).
		Select("COALESCE(SUM(unread_count), 0)").
		Scan(&totalCount).Error
	return totalCount, err
}

// ClearConversationUnreadCount 清零指定会话的未读消息计数
func (s *MessageService) ClearConversationUnreadCount(conversationID, userID int64) error {
	return s.db.Model(&models.ConversationUnreadCount{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("unread_count", 0).Error
}

// incrementConversationUnreadCount 增加指定会话的未读消息计数
func (s *MessageService) incrementConversationUnreadCount(conversationID, userID int64) error {
	// 尝试更新现有记录
	result := s.db.Model(&models.ConversationUnreadCount{}).
		Where("conversation_id = ? AND user_id = ?", conversationID, userID).
		Update("unread_count", gorm.Expr("unread_count + ?", 1))

	if result.Error != nil {
		return result.Error
	}

	// 如果没有找到记录，创建新记录
	if result.RowsAffected == 0 {
		unreadCount := &models.ConversationUnreadCount{
			ConversationID: conversationID,
			UserID:         userID,
			UnreadCount:    1,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		return s.db.Create(unreadCount).Error
	}

	return nil
}

// SearchPrivateMessages 搜索私聊消息
func (s *MessageService) SearchPrivateMessages(user1ID, user2ID int64, keyword string, page, pageSize int) ([]models.PrivateMessage, int64, error) {
	var messages []models.PrivateMessage
	var total int64

	query := s.db.Model(&models.PrivateMessage{}).Where(
		"((sender_id = ? AND receiver_id = ?) OR (sender_id = ? AND receiver_id = ?)) AND content LIKE ?",
		user1ID, user2ID, user2ID, user1ID, "%"+keyword+"%",
	)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&messages).Error

	return messages, total, err
}

// DeletePrivateMessage 删除私聊消息（软删除）
func (s *MessageService) DeletePrivateMessage(messageID, userID int64) error {
	// 只有发送者可以删除消息
	var message models.PrivateMessage
	if err := s.db.First(&message, messageID).Error; err != nil {
		return errors.New("消息不存在")
	}

	if message.SenderID != userID {
		return errors.New("只能删除自己发送的消息")
	}

	return s.db.Delete(&message).Error
}

// CleanupOldPrivateMessages 清理旧的私聊消息
func (s *MessageService) CleanupOldPrivateMessages(daysToKeep int) error {
	if daysToKeep <= 0 {
		return errors.New("保留天数必须大于0")
	}

	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)
	return s.db.Where("created_at < ?", cutoffDate).Delete(&models.PrivateMessage{}).Error
}
