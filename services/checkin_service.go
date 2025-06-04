package services

import (
	"campus-canvas-chat/database"
	"campus-canvas-chat/models"
	"errors"
	"time"

	"gorm.io/gorm"
)

type CheckInService struct {
	db *gorm.DB
}

func NewCheckInService() *CheckInService {
	return &CheckInService{
		db: database.GetDB(),
	}
}

// CreateCheckInTask 创建打卡任务
func (s *CheckInService) CreateCheckInTask(task *models.CheckInTask, operatorID int64) error {
	// 检查操作者是否是房主或管理员
	var member models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", task.ChatRoomID, operatorID).First(&member).Error; err != nil {
		return errors.New("用户不是该聊天室成员")
	}

	if member.Role != "OWNER" && member.Role != "ADMIN" {
		return errors.New("权限不足，只有房主和管理员可以创建打卡任务")
	}

	// 创建打卡任务
	return s.db.Create(task).Error
}

// GetCheckInTasks 获取聊天室的打卡任务列表
func (s *CheckInService) GetCheckInTasks(chatRoomID int64, isActive bool) ([]models.CheckInTask, error) {
	var tasks []models.CheckInTask

	query := s.db.Where("chat_room_id = ?", chatRoomID)
	if isActive {
		query = query.Where("is_active = ?", true)
	}

	err := query.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

// UpdateCheckInTask 更新打卡任务
func (s *CheckInService) UpdateCheckInTask(taskID int64, updates map[string]interface{}, operatorID int64) error {
	// 获取任务信息
	var task models.CheckInTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return errors.New("打卡任务不存在")
	}

	// 检查操作者权限
	var member models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", task.ChatRoomID, operatorID).First(&member).Error; err != nil {
		return errors.New("用户不是该聊天室成员")
	}

	if member.Role != "OWNER" && member.Role != "ADMIN" {
		return errors.New("权限不足")
	}

	// 更新任务
	return s.db.Model(&task).Updates(updates).Error
}

// DeleteCheckInTask 删除打卡任务
func (s *CheckInService) DeleteCheckInTask(taskID int64, operatorID int64) error {
	// 获取任务信息
	var task models.CheckInTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return errors.New("打卡任务不存在")
	}

	// 检查操作者权限
	var member models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", task.ChatRoomID, operatorID).First(&member).Error; err != nil {
		return errors.New("用户不是该聊天室成员")
	}

	if member.Role != "OWNER" && member.Role != "ADMIN" {
		return errors.New("权限不足")
	}

	// 软删除任务
	return s.db.Model(&task).Update("is_active", false).Error
}

// SubmitCheckIn 提交打卡记录
func (s *CheckInService) SubmitCheckIn(checkIn *models.CheckIn) error {
	// 检查用户是否是聊天室成员
	var member models.ChatRoomMember
	if err := s.db.Where("chat_room_id = ? AND user_id = ?", checkIn.ChatRoomID, checkIn.UserID).First(&member).Error; err != nil {
		return errors.New("用户不是该聊天室成员")
	}

	// 检查今天是否已经打卡
	today := time.Now().Format("2006-01-02")
	checkDate, _ := time.Parse("2006-01-02", today)

	var existingCheckIn models.CheckIn
	if err := s.db.Where("chat_room_id = ? AND user_id = ? AND check_date = ?",
		checkIn.ChatRoomID, checkIn.UserID, checkDate).First(&existingCheckIn).Error; err == nil {
		return errors.New("今天已经打卡过了")
	}

	// 设置打卡日期为今天
	checkIn.CheckDate = checkDate

	// 提交打卡记录
	return s.db.Create(checkIn).Error
}

// GetCheckInRecords 获取打卡记录
func (s *CheckInService) GetCheckInRecords(chatRoomID int64, userID *int64, startDate, endDate *time.Time, page, pageSize int) ([]models.CheckIn, int64, error) {
	var checkIns []models.CheckIn
	var total int64

	query := s.db.Model(&models.CheckIn{}).Where("chat_room_id = ?", chatRoomID)

	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	if startDate != nil {
		query = query.Where("check_date >= ?", *startDate)
	}

	if endDate != nil {
		query = query.Where("check_date <= ?", *endDate)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	err := query.Preload("User").
		Order("check_date DESC, created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&checkIns).Error

	return checkIns, total, err
}

// GetCheckInStats 获取打卡统计信息
func (s *CheckInService) GetCheckInStats(chatRoomID int64, startDate, endDate time.Time) (map[string]interface{}, error) {
	// 获取时间范围内的打卡记录
	var checkIns []models.CheckIn
	err := s.db.Where("chat_room_id = ? AND check_date >= ? AND check_date <= ?",
		chatRoomID, startDate, endDate).Preload("User").Find(&checkIns).Error
	if err != nil {
		return nil, err
	}

	// 统计每个用户的打卡次数
	userStats := make(map[int64]map[string]interface{})
	for _, checkIn := range checkIns {
		if userStats[checkIn.UserID] == nil {
			userStats[checkIn.UserID] = map[string]interface{}{
				"user":       checkIn.User,
				"count":      0,
				"last_check": checkIn.CheckDate,
			}
		}
		userStats[checkIn.UserID]["count"] = userStats[checkIn.UserID]["count"].(int) + 1
		if checkIn.CheckDate.After(userStats[checkIn.UserID]["last_check"].(time.Time)) {
			userStats[checkIn.UserID]["last_check"] = checkIn.CheckDate
		}
	}

	// 转换为排行榜格式
	type UserRank struct {
		User      models.User `json:"user"`
		Count     int         `json:"count"`
		LastCheck time.Time   `json:"last_check"`
	}

	var ranking []UserRank
	for _, stats := range userStats {
		ranking = append(ranking, UserRank{
			User:      stats["user"].(models.User),
			Count:     stats["count"].(int),
			LastCheck: stats["last_check"].(time.Time),
		})
	}

	// 按打卡次数排序
	for i := 0; i < len(ranking)-1; i++ {
		for j := i + 1; j < len(ranking); j++ {
			if ranking[i].Count < ranking[j].Count {
				ranking[i], ranking[j] = ranking[j], ranking[i]
			}
		}
	}

	// 计算总体统计
	totalCheckIns := len(checkIns)
	uniqueUsers := len(userStats)

	// 计算连续打卡天数（以当前日期为基准）
	continuousDays := s.calculateContinuousDays(chatRoomID)

	return map[string]interface{}{
		"total_check_ins": totalCheckIns,
		"unique_users":    uniqueUsers,
		"ranking":         ranking,
		"continuous_days": continuousDays,
	}, nil
}

// calculateContinuousDays 计算连续打卡天数
func (s *CheckInService) calculateContinuousDays(chatRoomID int64) int {
	continuousDays := 0
	currentDate := time.Now()

	for {
		dateStr := currentDate.Format("2006-01-02")
		checkDate, _ := time.Parse("2006-01-02", dateStr)

		var count int64
		s.db.Model(&models.CheckIn{}).Where("chat_room_id = ? AND check_date = ?", chatRoomID, checkDate).Count(&count)

		if count > 0 {
			continuousDays++
			currentDate = currentDate.AddDate(0, 0, -1)
		} else {
			break
		}
	}

	return continuousDays
}

// GetUserCheckInHistory 获取用户打卡历史
func (s *CheckInService) GetUserCheckInHistory(chatRoomID, userID int64, month string) ([]models.CheckIn, error) {
	var checkIns []models.CheckIn

	// 解析月份
	startDate, err := time.Parse("2006-01", month)
	if err != nil {
		return nil, errors.New("无效的月份格式")
	}

	endDate := startDate.AddDate(0, 1, 0).Add(-time.Second)

	err = s.db.Where("chat_room_id = ? AND user_id = ? AND check_date >= ? AND check_date <= ?",
		chatRoomID, userID, startDate, endDate).
		Order("check_date ASC").
		Find(&checkIns).Error

	return checkIns, err
}

// GetTodayCheckInStatus 获取今天的打卡状态
func (s *CheckInService) GetTodayCheckInStatus(chatRoomID, userID int64) (bool, *models.CheckIn, error) {
	today := time.Now().Format("2006-01-02")
	checkDate, _ := time.Parse("2006-01-02", today)

	var checkIn models.CheckIn
	err := s.db.Where("chat_room_id = ? AND user_id = ? AND check_date = ?",
		chatRoomID, userID, checkDate).First(&checkIn).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil, nil
		}
		return false, nil, err
	}

	return true, &checkIn, nil
}
