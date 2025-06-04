package main

import (
	"campus-canvas-chat/config"
	"campus-canvas-chat/database"
	"campus-canvas-chat/redis"
	"campus-canvas-chat/routes"
	"campus-canvas-chat/websocket"
	"log"
)

func main() {
	// 加载配置
	cfg := config.LoadConfig()

	// 初始化数据库
	if err := database.InitDatabase(cfg); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}

	// 初始化Redis
	if err := redis.InitRedis(cfg); err != nil {
		log.Fatalf("Redis初始化失败: %v", err)
	}

	// 创建WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// 设置路由
	r := routes.SetupRoutes(hub)

	// 启动服务器
	log.Printf("服务器启动在端口: %s", cfg.Server.Port)
	if err := r.Run(":" + cfg.Server.Port); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
