package controllers

import (
	"campus-canvas-chat/services"
	"campus-canvas-chat/websocket"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type MessageController struct {
	messageService *services.MessageService
	webSocketHub   *websocket.Hub
}

func NewMessageController(messageService *services.MessageService, webSocketHub *websocket.Hub) *MessageController {
	return &MessageController{
		messageService: messageService,
		webSocketHub:   webSocketHub,
	}
}

// SendGroupMessage 发送群聊消息
func (mc *MessageController) SendGroupMessage(c *gin.Context) {
	type SendGroupMessageRequest struct {
		ChatRoomId int64  `json:"chatRoomId" binding:"required"`
		UserId     int64  `json:"userId" binding:"required"`
		Content    string `json:"content" binding:"required"`
	}

	var req SendGroupMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 发送群聊消息（持久化存储）
	message, err := mc.messageService.SendGroupMessage(req.ChatRoomId, req.UserId, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 通过WebSocket广播消息给聊天室内的在线用户
	messageData, _ := json.Marshal(map[string]interface{}{
		"type":    "group_message",
		"message": message,
	})
	mc.webSocketHub.BroadcastToRoom(req.ChatRoomId, messageData)

	c.JSON(http.StatusOK, gin.H{
		"message": "群聊消息发送成功",
		"data":    message,
	})
}

// GetGroupMessages 获取群聊消息列表
func (mc *MessageController) GetGroupMessages(c *gin.Context) {
	// 获取路径参数
	chatRoomIdStr := c.Param("chatRoomId")
	chatRoomId, err := strconv.ParseInt(chatRoomIdStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "聊天室ID格式错误"})
		return
	}

	// 获取查询参数
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "页码格式错误"})
		return
	}

	pageSize, err := strconv.Atoi(pageSizeStr)
	if err != nil || pageSize < 1 || pageSize > 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "页面大小格式错误或超出范围(1-100)"})
		return
	}

	// 获取群聊消息
	messages, total, err := mc.messageService.GetGroupMessages(chatRoomId, page, pageSize)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 计算总页数
	totalPage := (total + int64(pageSize) - 1) / int64(pageSize)

	c.JSON(http.StatusOK, gin.H{
		"message": "获取群聊消息成功",
		"data": gin.H{
			"messages":  messages,
			"page":      page,
			"pageSize":  pageSize,
			"total":     total,
			"totalPage": totalPage,
		},
	})
}

// SendPrivateMessage 发送私聊消息
func (mc *MessageController) SendPrivateMessage(c *gin.Context) {
	type SendPrivateMessageRequest struct {
		SenderId   int64  `json:"senderId" binding:"required"`
		ReceiverId int64  `json:"receiverId" binding:"required"`
		Content    string `json:"content" binding:"required"`
	}

	var req SendPrivateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 发送私聊消息（持久化存储）
	message, err := mc.messageService.SendPrivateMessage(req.SenderId, req.ReceiverId, req.Content)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 通过WebSocket推送给接收者（如果在线）
	privateMessageData, _ := json.Marshal(map[string]interface{}{
		"type":      "private_message",
		"content":   message.Content,
		"createdAt": message.CreatedAt,
		"senderId":  message.SenderID,
	})
	mc.webSocketHub.SendToUser(req.ReceiverId, privateMessageData)

	c.JSON(http.StatusOK, gin.H{
		"message":   "私聊消息发送成功",
		"createdAt": message.CreatedAt,
	})
}

// GetPrivateMessages 获取私聊消息列表
func (mc *MessageController) GetPrivateMessages(c *gin.Context) {
	// 从路径参数获取对方用户ID
	otherUserIDStr := c.Param("user_id")
	otherUserID, err := strconv.ParseInt(otherUserIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 从查询参数获取当前用户ID
	userIDStr := c.Query("userId")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少用户ID参数"})
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 获取私聊消息
	messages, total, err := mc.messageService.GetPrivateMessages(userID, otherUserID, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取消息失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"messages":   messages,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetConversations 获取用户的所有会话列表
func (mc *MessageController) GetConversations(c *gin.Context) {
	// 从查询参数获取用户ID
	userIDStr := c.Query("userId")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少用户ID参数"})
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// 获取会话列表
	conversations, err := mc.messageService.GetConversations(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取会话列表失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "获取会话列表成功",
		"conversations": conversations,
		"total":         len(conversations),
		"page":          page,
		"limit":         limit,
	})
}

// GetUserTotalUnreadCount 获取用户所有会话的未读消息总数
func (mc *MessageController) GetUserTotalUnreadCount(c *gin.Context) {
	// 从查询参数获取用户ID
	userIDStr := c.Query("userId")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少用户ID参数"})
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 获取用户所有会话的未读消息总数
	count, err := mc.messageService.GetUserTotalUnreadCount(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "获取未读消息数量失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"unreadCount": count,
		},
	})
}

// ClearConversationUnreadCount 清零指定会话的未读消息计数
func (mc *MessageController) ClearConversationUnreadCount(c *gin.Context) {
	type ClearConversationUnreadRequest struct {
		ConversationId int64 `json:"conversationId" binding:"required"`
		UserId         int64 `json:"userId" binding:"required"`
	}

	var req ClearConversationUnreadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 清零会话未读计数
	err := mc.messageService.ClearConversationUnreadCount(req.ConversationId, req.UserId)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "清零未读计数失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "清零未读计数成功",
	})
}

// SearchPrivateMessages 搜索私聊消息
func (mc *MessageController) SearchPrivateMessages(c *gin.Context) {
	// 从路径参数获取对方用户ID
	otherUserIDStr := c.Param("user_id")
	otherUserID, err := strconv.ParseInt(otherUserIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户ID格式错误"})
		return
	}

	// 从查询参数获取用户ID
	userIDStr := c.Query("userId")
	if userIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少用户ID参数"})
		return
	}
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 获取搜索关键词
	keyword := c.Query("keyword")
	if keyword == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "搜索关键词不能为空"})
		return
	}

	// 获取分页参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 搜索消息
	messages, total, err := mc.messageService.SearchPrivateMessages(userID, otherUserID, keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "搜索消息失败: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"messages":   messages,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
			"keyword":    keyword,
		},
	})
}

// DeletePrivateMessage 软删除私聊消息
func (mc *MessageController) DeletePrivateMessage(c *gin.Context) {
	type DeletePrivateMessageRequest struct {
		UserId    int64 `json:"userId" binding:"required"`
		MessageId int64 `json:"messageId" binding:"required"`
	}

	var req DeletePrivateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误: " + err.Error()})
		return
	}

	// 删除私聊消息
	err := mc.messageService.DeletePrivateMessage(req.MessageId, req.UserId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "消息删除成功",
	})
}
