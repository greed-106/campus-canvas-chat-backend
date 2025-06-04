package controllers

import (
	"campus-canvas-chat/models"
	"campus-canvas-chat/services"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type ChatRoomController struct {
	chatRoomService *services.ChatRoomService
}

func NewChatRoomController() *ChatRoomController {
	return &ChatRoomController{
		chatRoomService: services.NewChatRoomService(),
	}
}

// CreateChatRoom 创建聊天室
func (ctrl *ChatRoomController) CreateChatRoom(c *gin.Context) {
	var req struct {
		Name        string `json:"name" binding:"required,min=1,max=100"`
		Description string `json:"description" binding:"max=1000"`
		Category    string `json:"category" binding:"required,min=1,max=50"`
		MaxMembers  int    `json:"maxMembers" binding:"min=1,max=1000"`
		CreatorID   int64  `json:"creatorId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	chatRoom := &models.ChatRoom{
		Name:        req.Name,
		Description: req.Description,
		Category:    req.Category,
		CreatorID:   req.CreatorID,
		MaxMembers:  req.MaxMembers,
		IsActive:    true,
		IsApproved:  false, // 需要审核
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := ctrl.chatRoomService.CreateChatRoom(chatRoom); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "聊天室创建成功，等待审核",
		"data":    chatRoom.ID,
	})
}

// GetChatRoomList 获取聊天室列表
func (ctrl *ChatRoomController) GetChatRoomList(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	category := c.Query("category")
	isApproved := c.DefaultQuery("approved", "true") == "true"

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	rooms, total, err := ctrl.chatRoomService.GetChatRoomList(page, pageSize, category, isApproved)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"rooms":      rooms,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"total_page": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetChatRoomDetail 获取聊天室详情
func (ctrl *ChatRoomController) GetChatRoomDetail(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	room, err := ctrl.chatRoomService.GetChatRoomByID(roomID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "聊天室不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": room})
}

// JoinChatRoom 加入聊天室
func (ctrl *ChatRoomController) JoinChatRoom(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	var req struct {
		UserID int64 `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.chatRoomService.JoinChatRoom(roomID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成功加入聊天室"})
}

// LeaveChatRoom 离开聊天室
func (ctrl *ChatRoomController) LeaveChatRoom(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	var req struct {
		UserID int64 `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.chatRoomService.LeaveChatRoom(roomID, req.UserID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成功离开聊天室"})
}

// DeleteChatRoom 删除聊天室
func (ctrl *ChatRoomController) DeleteChatRoom(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	var req struct {
		UserID int64 `json:"userId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.chatRoomService.DeleteChatRoom(roomID, req.UserID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "聊天室删除成功"})
}

// GetUserChatRooms 获取用户加入的聊天室列表
func (ctrl *ChatRoomController) GetUserChatRooms(c *gin.Context) {
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	rooms, err := ctrl.chatRoomService.GetUserChatRooms(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rooms})
}

// ApproveChatRoom 审核聊天室（管理员功能）
func (ctrl *ChatRoomController) ApproveChatRoom(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	var req struct {
		Approved bool `json:"approved"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.chatRoomService.ApproveChatRoom(roomID, req.Approved); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	message := "聊天室审核通过"
	if !req.Approved {
		message = "聊天室审核拒绝"
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// UpdateMemberRole 更新成员角色
func (ctrl *ChatRoomController) UpdateMemberRole(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	var req struct {
		OperatorID   int64  `json:"operatorId" binding:"required"`
		TargetUserID int64  `json:"targetUserId" binding:"required"`
		NewRole      string `json:"newRole" binding:"required,oneof=MEMBER ADMIN"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.chatRoomService.UpdateMemberRole(roomID, req.OperatorID, req.TargetUserID, req.NewRole); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成员角色更新成功"})
}

// MuteMember 禁言成员
func (ctrl *ChatRoomController) MuteMember(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	var req struct {
		OperatorID   int64 `json:"operatorId" binding:"required"`
		TargetUserID int64 `json:"targetUserId" binding:"required"`
		Muted        bool  `json:"muted"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.chatRoomService.MuteMember(roomID, req.OperatorID, req.TargetUserID, req.Muted); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	message := "成员禁言成功"
	if !req.Muted {
		message = "成员解除禁言成功"
	}

	c.JSON(http.StatusOK, gin.H{"message": message})
}

// KickMember 踢出成员
func (ctrl *ChatRoomController) KickMember(c *gin.Context) {
	roomIDStr := c.Param("id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	var req struct {
		OperatorID   int64 `json:"operatorId" binding:"required"`
		TargetUserID int64 `json:"targetUserId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.chatRoomService.KickMember(roomID, req.OperatorID, req.TargetUserID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "成员踢出成功"})
}
