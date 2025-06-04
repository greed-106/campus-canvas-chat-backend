package controllers

import (
	"campus-canvas-chat/models"
	"campus-canvas-chat/services"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type CheckInController struct {
	checkInService *services.CheckInService
}

func NewCheckInController() *CheckInController {
	return &CheckInController{
		checkInService: services.NewCheckInService(),
	}
}

// CreateCheckInTask 创建打卡任务
func (ctrl *CheckInController) CreateCheckInTask(c *gin.Context) {
	var req struct {
		ChatRoomID  int64  `json:"chatRoomId" binding:"required"`
		Title       string `json:"title" binding:"required,min=1,max=200"`
		Description string `json:"description" binding:"max=1000"`
		Cycle       string `json:"cycle" binding:"required,oneof=DAILY WEEKLY MONTHLY"`
		StartDate   string `json:"startDate" binding:"required"`
		EndDate     string `json:"endDate"`
		OperatorID  int64  `json:"operatorId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 解析开始日期
	startDate, err := time.Parse("2006-01-02", req.StartDate)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式"})
		return
	}

	task := &models.CheckInTask{
		ChatRoomID:  req.ChatRoomID,
		Title:       req.Title,
		Description: req.Description,
		Cycle:       req.Cycle,
		IsActive:    true,
		StartDate:   startDate,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 解析结束日期（可选）
	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
			return
		}
		task.EndDate = &endDate
	}

	if err := ctrl.checkInService.CreateCheckInTask(task, req.OperatorID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "打卡任务创建成功",
		"data":    task,
	})
}

// GetCheckInTasks 获取打卡任务列表
func (ctrl *CheckInController) GetCheckInTasks(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	isActive := c.DefaultQuery("active", "true") == "true"

	tasks, err := ctrl.checkInService.GetCheckInTasks(roomID, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tasks})
}

// UpdateCheckInTask 更新打卡任务
func (ctrl *CheckInController) UpdateCheckInTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	var req struct {
		Title       string `json:"title" binding:"omitempty,min=1,max=200"`
		Description string `json:"description" binding:"omitempty,max=1000"`
		Cycle       string `json:"cycle" binding:"omitempty,oneof=DAILY WEEKLY MONTHLY"`
		IsActive    *bool  `json:"isActive"`
		EndDate     string `json:"endDate"`
		OperatorID  int64  `json:"operatorId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.Cycle != "" {
		updates["cycle"] = req.Cycle
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
			return
		}
		updates["end_date"] = endDate
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有需要更新的字段"})
		return
	}

	updates["updated_at"] = time.Now()

	if err := ctrl.checkInService.UpdateCheckInTask(taskID, updates, req.OperatorID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "打卡任务更新成功"})
}

// DeleteCheckInTask 删除打卡任务
func (ctrl *CheckInController) DeleteCheckInTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseInt(taskIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的任务ID"})
		return
	}

	var req struct {
		OperatorID int64 `json:"operatorId" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.checkInService.DeleteCheckInTask(taskID, req.OperatorID); err != nil {
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "打卡任务删除成功"})
}

// SubmitCheckIn 提交打卡记录
func (ctrl *CheckInController) SubmitCheckIn(c *gin.Context) {
	var req struct {
		ChatRoomID int64  `json:"chatRoomId" binding:"required"`
		UserID     int64  `json:"userId" binding:"required"`
		Content    string `json:"content" binding:"max=500"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	checkIn := &models.CheckIn{
		ChatRoomID: req.ChatRoomID,
		UserID:     req.UserID,
		Content:    req.Content,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if err := ctrl.checkInService.SubmitCheckIn(checkIn); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "打卡成功",
		"data":    checkIn,
	})
}

// GetCheckInRecords 获取打卡记录
func (ctrl *CheckInController) GetCheckInRecords(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// 可选的用户ID过滤
	var userID *int64
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if uid, err := strconv.ParseInt(userIDStr, 10, 64); err == nil {
			userID = &uid
		}
	}

	// 可选的日期范围过滤
	var startDate, endDate *time.Time
	if startDateStr := c.Query("start_date"); startDateStr != "" {
		if sd, err := time.Parse("2006-01-02", startDateStr); err == nil {
			startDate = &sd
		}
	}
	if endDateStr := c.Query("end_date"); endDateStr != "" {
		if ed, err := time.Parse("2006-01-02", endDateStr); err == nil {
			endDate = &ed
		}
	}

	checkIns, total, err := ctrl.checkInService.GetCheckInRecords(roomID, userID, startDate, endDate, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"checkIns":  checkIns,
			"total":     total,
			"page":      page,
			"pageSize":  pageSize,
			"totalPage": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetCheckInStats 获取打卡统计信息
func (ctrl *CheckInController) GetCheckInStats(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	// 默认统计最近30天
	startDateStr := c.DefaultQuery("start_date", time.Now().AddDate(0, 0, -30).Format("2006-01-02"))
	endDateStr := c.DefaultQuery("end_date", time.Now().Format("2006-01-02"))

	startDate, err := time.Parse("2006-01-02", startDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的开始日期格式"})
		return
	}

	endDate, err := time.Parse("2006-01-02", endDateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的结束日期格式"})
		return
	}

	stats, err := ctrl.checkInService.GetCheckInStats(roomID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": stats})
}

// GetUserCheckInHistory 获取用户打卡历史
func (ctrl *CheckInController) GetUserCheckInHistory(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	userIDStr := c.Param("user_id")

	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	// 默认查询当前月份
	month := c.DefaultQuery("month", time.Now().Format("2006-01"))

	checkIns, err := ctrl.checkInService.GetUserCheckInHistory(roomID, userID, month)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": checkIns})
}

// GetTodayCheckInStatus 获取今天的打卡状态
func (ctrl *CheckInController) GetTodayCheckInStatus(c *gin.Context) {
	roomIDStr := c.Param("room_id")
	userIDStr := c.Param("user_id")

	roomID, err := strconv.ParseInt(roomIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的聊天室ID"})
		return
	}

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的用户ID"})
		return
	}

	checkedIn, checkIn, err := ctrl.checkInService.GetTodayCheckInStatus(roomID, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"checkedIn": checkedIn,
			"checkIn":   checkIn,
		},
	})
}
