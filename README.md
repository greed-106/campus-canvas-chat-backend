# Campus Canvas Chat - 校园聊天室后端

基于 Golang + Gin + GORM + WebSocket + Redis 实现的聊天室社群后端系统，支持学生自由创建/加入学习社群。

## 功能特性

### 🏠 小组管理服务
- ✅ 用户可创建/删除兴趣小组（需审核）
- ✅ 设置小组名称、介绍、分类（如技术、艺术、运动等）
- ✅ 用户可查看小组列表
- ✅ 用户可加入/退出小组
- ✅ 管理员可管理成员（禁言、踢出）
- ✅ 独立的管理员账户信息表

### 💬 实时消息服务
- ✅ 小组成员可发送文本消息
- ✅ 在线用户通过WebSocket实时接收消息
- ✅ 消息搜索功能

### 📅 打卡功能
- ✅ 小组可开启周期性打卡任务（每日/每周/每月）
- ✅ 成员提交打卡记录
- ✅ 统计小组内打卡排行榜

## 技术栈

- **后端框架**: Gin (Go Web Framework)
- **ORM**: GORM
- **数据库**: MySQL
- **缓存**: Redis
- **实时通信**: WebSocket

## 项目结构

```
CampusCanvasChat/
├── config/          # 配置管理
├── controllers/     # 控制器层
├── database/        # 数据库连接和初始化
├── models/          # 数据模型
├── redis/           # Redis连接和操作
├── routes/          # 路由配置
├── services/        # 业务逻辑层
├── websocket/       # WebSocket管理
├── main.go          # 程序入口
├── go.mod           # Go模块文件
├── .env.example     # 环境变量示例
└── README.md        # 项目说明
```

## 快速开始

### 1. 环境要求
- Golang
- MySQL
- Redis

### 2. 安装依赖
```bash
go mod tidy
```

### 3. 配置环境变量
```bash
cp .env.example .env
# 编辑 .env 文件，配置数据库和Redis连接信息
```

### 4. 运行项目
```bash
go run main.go
```

服务器将在 `http://localhost:<你指定的端口>` 启动