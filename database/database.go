package database

import (
	"campus-canvas-chat/config"
	"campus-canvas-chat/models"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// InitDatabase 初始化数据库连接
func InitDatabase(cfg *config.Config) error {
	var err error

	// 连接数据库
	DB, err = gorm.Open(mysql.Open(cfg.GetDSN()), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return err
	}

	// 自动迁移表结构
	err = AutoMigrate()
	if err != nil {
		return err
	}

	log.Println("数据库连接成功")
	return nil
}

// AutoMigrate 自动迁移表结构
func AutoMigrate() error {
	return DB.AutoMigrate(
		&models.User{},
		&models.ChatRoom{},
		&models.ChatRoomMember{},
		&models.Message{},
		&models.Admin{},
		&models.CheckIn{},
		&models.CheckInTask{},
		&models.Conversation{},
		&models.PrivateMessage{},
		&models.ConversationUnreadCount{},
	)
}

// GetDB 获取数据库实例
func GetDB() *gorm.DB {
	return DB
}
