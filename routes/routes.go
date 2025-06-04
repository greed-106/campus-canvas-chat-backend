package routes

import (
	"campus-canvas-chat/controllers"
	"campus-canvas-chat/services"
	"campus-canvas-chat/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置路由
func SetupRoutes(hub *websocket.Hub) *gin.Engine {
	r := gin.Default()

	// 配置CORS
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Type", "Accept", "Authorization"}
	r.Use(cors.New(config))

	// 初始化服务
	messageService := services.NewMessageService()

	// 初始化控制器
	chatRoomController := controllers.NewChatRoomController()
	messageController := controllers.NewMessageController(messageService, hub)
	checkInController := controllers.NewCheckInController()

	// API版本分组
	v1 := r.Group("/campus-canvas/api")
	{
		// 聊天室相关路由
		chatRooms := v1.Group("/chatrooms")
		{
			// 基础CRUD操作
			chatRooms.POST("", chatRoomController.CreateChatRoom)       // 创建聊天室
			chatRooms.GET("", chatRoomController.GetChatRoomList)       // 获取聊天室列表
			chatRooms.GET("/:id", chatRoomController.GetChatRoomDetail) // 获取聊天室详情
			chatRooms.DELETE("/:id", chatRoomController.DeleteChatRoom) // 删除聊天室

			// 成员管理
			chatRooms.POST("/:id/join", chatRoomController.JoinChatRoom)            // 加入聊天室
			chatRooms.POST("/:id/leave", chatRoomController.LeaveChatRoom)          // 离开聊天室
			chatRooms.PUT("/:id/members/role", chatRoomController.UpdateMemberRole) // 更新成员角色
			chatRooms.PUT("/:id/members/mute", chatRoomController.MuteMember)       // 禁言/解禁成员
			chatRooms.DELETE("/:id/members/kick", chatRoomController.KickMember)    // 踢出成员

			// 管理员功能
			chatRooms.PUT("/:id/approve", chatRoomController.ApproveChatRoom) // 审核聊天室
		}

		// 用户相关路由
		users := v1.Group("/users")
		{
			users.GET("/:user_id/chatrooms", chatRoomController.GetUserChatRooms) // 获取用户加入的聊天室
		}

		// 群聊消息路由
		groupMessages := v1.Group("/group-messages")
		{
			groupMessages.POST("/send", messageController.SendGroupMessage)
			groupMessages.GET("/chatroom/:chatRoomId", messageController.GetGroupMessages)
		}

		// 私聊消息路由
		privateMessages := v1.Group("/private-messages")
		{
			privateMessages.POST("/send", messageController.SendPrivateMessage)
			privateMessages.GET("/with/:user_id", messageController.GetPrivateMessages)
			privateMessages.GET("/conversations", messageController.GetConversations)
			privateMessages.GET("/unread/count", messageController.GetUserTotalUnreadCount)
			privateMessages.POST("/clear-unread", messageController.ClearConversationUnreadCount)
			privateMessages.GET("/search/:user_id", messageController.SearchPrivateMessages)
			privateMessages.DELETE("/:message_id", messageController.DeletePrivateMessage)
		}

		// 打卡相关路由
		checkIns := v1.Group("/checkins")
		{
			// 打卡任务管理
			tasks := checkIns.Group("/tasks")
			{
				tasks.POST("", checkInController.CreateCheckInTask)            // 创建打卡任务
				tasks.GET("/room/:room_id", checkInController.GetCheckInTasks) // 获取聊天室打卡任务
				tasks.PUT("/:id", checkInController.UpdateCheckInTask)         // 更新打卡任务
				tasks.DELETE("/:id", checkInController.DeleteCheckInTask)      // 删除打卡任务
			}

			// 打卡记录
			checkIns.POST("", checkInController.SubmitCheckIn)                      // 提交打卡
			checkIns.GET("/room/:room_id", checkInController.GetCheckInRecords)     // 获取打卡记录
			checkIns.GET("/room/:room_id/stats", checkInController.GetCheckInStats) // 获取打卡统计

			// 用户打卡历史
			checkIns.GET("/room/:room_id/user/:user_id/history", checkInController.GetUserCheckInHistory) // 获取用户打卡历史
			checkIns.GET("/room/:room_id/user/:user_id/today", checkInController.GetTodayCheckInStatus)   // 获取今天打卡状态
		}
	}

	// WebSocket路由
	r.GET("/ws", hub.HandleWebSocket)

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "campus-canvas-chat",
			"version": "1.0.0",
		})
	})

	// 404处理
	r.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"error": "接口不存在",
			"path":  c.Request.URL.Path,
		})
	})

	return r
}
