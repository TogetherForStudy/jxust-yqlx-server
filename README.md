# GoJxust - 校园服务微信小程序后端

基于Go语言开发的校园服务微信小程序后端系统，主要提供选课参考功能，支持未来功能扩展。

## V1.1.0

- 自动登录：同意条款
- 选课助手：输入姓名查询老师评价；点评老师；审核评价；客服反馈
- 课程表：查看课程表，基于全量导出的数据；编辑课程表，自由基于已有数据进行添加、编辑和删除
- 挂科率：数据查询
- 英雄榜：增删改查
- 系统配置：增删改查，提供前端读取一些必要配置
- 倒数日：增删改查
- 学习清单：简易TODO
- 通知公告：运营账号发布；用户投稿；审核机制

## 未来计划

请参考设计文档。

## 功能特性

- ✅ 微信小程序登录认证
- ✅ 用户管理（注册、资料更新）
- ✅ 用户权限控制
- ✅ RESTful API设计
- ✅ JWT Token认证
- ✅ 数据库设计与迁移

## 技术栈

- **语言**: Go 1.24.1+
- **框架**: Gin Web Framework
- **数据库**: MySQL 8.0+
- **ORM**: GORM
- **认证**: JWT (golang-jwt/jwt)
- **配置**: 环境变量 + godotenv

## 项目结构

```
goJxust/
├── main.go                 # 应用入口
├── go.mod                 # Go模块文件
├── .env.example           # 环境变量模板
├── .gitignore            # Git忽略文件
├── readme.md             # 项目说明
├── internal/             # 内部包
│   ├── config/           # 配置管理
│   │   └── config.go
│   ├── database/         # 数据库连接
│   │   └── database.go
│   ├── models/           # 数据模型
│   │   └── models.go
│   ├── services/         # 业务逻辑层
│   │   ├── auth_service.go
│   │   ├── teacher_service.go
│   │   └── review_service.go
│   ├── handlers/         # 控制器层
│   │   ├── auth_handler.go
│   │   ├── teacher_handler.go
│   │   ├── review_handler.go
│   │   └── admin_handler.go
│   ├── middleware/       # 中间件
│   │   └── middleware.go
│   ├── utils/           # 工具函数
│   │   ├── response.go
│   │   └── auth.go
│   └── router/          # 路由配置
│       └── router.go
└── scripts/             # 脚本文件
    └── init.sql         # 数据库初始化脚本
```

## 快速开始

### 1. 环境准备

- Go 1.24+
- MySQL 8.0+
- 微信小程序开发者账号

### 2. 项目配置

```bash
# 复制环境变量模板
cp .env.example .env

# 编辑环境变量
vim .env
```

配置示例：
```env
# 数据库配置
DB_HOST=localhost
DB_PORT=3306
DB_USERNAME=root
DB_PASSWORD=your_password
DB_NAME=gojxust

# JWT密钥
JWT_SECRET=your_jwt_secret_key_here

# 服务器配置
SERVER_PORT=8080

# 微信小程序配置
WECHAT_APP_ID=your_wechat_app_id
WECHAT_APP_SECRET=your_wechat_app_secret

# minio 对象存储
MINIO_ENDPOINT=localhost:9000
MINIO_ACCESS_KEY=minioadmin
MINIO_SECRET_KEY=minioadmin
MINIO_USE_SSL=false
BUCKET_NAME=yqlx

# 主机配置（用于确保minio反向代理时签名匹配）
HOST=localhost:8085
SCHEME=http

# Redis 配置：Redis暂未启用，可以先不配置
```

### 3. 安装依赖

```bash
go mod tidy
```

### 4. 运行应用

```bash
go run main.go
```

应用将在 `http://localhost:8080` 启动

## 部署说明

### 使用Docker部署

1. 创建Dockerfile
2. 构建镜像: `make docker-build`
3. 运行容器: `docker run -p 8080:8080 gojxust`

### 生产环境配置

1. 设置 `GIN_MODE=release`
2. 使用反向代理 (Nginx)
3. 配置HTTPS证书
4. 设置数据库连接池
5. 配置日志文件
6. 监控和报警

## 开发规范

### 代码规范
- 遵循Go官方代码规范
- 使用gofmt格式化代码
- 添加必要的注释
- 错误处理规范

### API设计规范
- RESTful API设计
- 统一的响应格式
- 合理的HTTP状态码
- 参数验证和错误处理

### 数据库规范
- 表名使用复数形式
- 字段名使用下划线命名
- 添加必要的索引
- 软删除支持

## 贡献指南

1. Fork项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建Pull Request

    a. 管理员会随时审核您的代码,除非你的PR设置为`draft`状态或标题内包含`WIP`/`Work In Progress`/🚧字样，当然，你也可以手动at管理员/召唤gemini来review你的代码

## 许可证

MIT License
