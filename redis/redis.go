package redis

import (
	"campus-canvas-chat/config"
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

var Client *redis.Client
var ctx = context.Background()

// InitRedis 初始化Redis连接
func InitRedis(cfg *config.Config) error {
	Client = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})

	// 测试连接
	_, err := Client.Ping(ctx).Result()
	if err != nil {
		return err
	}

	log.Println("Redis连接成功")
	return nil
}

// GetClient 获取Redis客户端
func GetClient() *redis.Client {
	return Client
}

// SetUserOnline 设置用户在线状态
func SetUserOnline(userID int64) error {
	key := fmt.Sprintf("user:online:%d", userID)
	return Client.Set(ctx, key, "1", 24*time.Hour).Err()
}

// SetUserOffline 设置用户离线状态
func SetUserOffline(userID int64) error {
	key := fmt.Sprintf("user:online:%d", userID)
	return Client.Del(ctx, key).Err()
}

// IsUserOnline 检查用户是否在线
func IsUserOnline(userID int64) bool {
	key := fmt.Sprintf("user:online:%d", userID)
	result := Client.Exists(ctx, key)
	return result.Val() > 0
}

// AddUserToRoom 将用户添加到房间
func AddUserToRoom(roomID, userID int64) error {
	key := fmt.Sprintf("room:users:%d", roomID)
	return Client.SAdd(ctx, key, userID).Err()
}

// RemoveUserFromRoom 从房间移除用户
func RemoveUserFromRoom(roomID, userID int64) error {
	key := fmt.Sprintf("room:users:%d", roomID)
	return Client.SRem(ctx, key, userID).Err()
}

// GetRoomUsers 获取房间内的用户列表
func GetRoomUsers(roomID int64) ([]string, error) {
	key := fmt.Sprintf("room:users:%d", roomID)
	return Client.SMembers(ctx, key).Result()
}

// CacheMessage 缓存消息（用于离线消息）
func CacheMessage(userID int64, messageData string) error {
	key := fmt.Sprintf("offline:messages:%d", userID)
	// 使用列表存储离线消息，设置7天过期时间
	pipe := Client.Pipeline()
	pipe.LPush(ctx, key, messageData)
	pipe.Expire(ctx, key, 7*24*time.Hour)
	_, err := pipe.Exec(ctx)
	return err
}

// GetOfflineMessages 获取离线消息
func GetOfflineMessages(userID int64) ([]string, error) {
	key := fmt.Sprintf("offline:messages:%d", userID)
	messages, err := Client.LRange(ctx, key, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	// 获取后清空离线消息
	Client.Del(ctx, key)
	return messages, nil
}