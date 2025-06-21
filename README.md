# GoJxust - 校园服务微信小程序后端

基于Go语言开发的校园服务微信小程序后端系统，主要提供教师点评功能，支持未来功能扩展。

## 原始需求

实现一个Go语言开发的校园服务微信小程序后端；

使用Go语言及相关技术栈，遵循go语言的开发规范；

数据库设计需要满足微信小程序后端需求，业务方面暂时涉及用户、选课功能，保留可扩展能力；

功能需求：

实现：老师点评功能，用户可以点评老师，输入老师的名字和教授的课程，200字以内的评语进行点评，管理员可以进行审核和增删改查评语；用户可以通过输入老师的名字查询老师的评价；

预留以下功能的可扩展性，不需要实现它们：

学业：课表、挂科率、选课、校历、期末资料、学分（学历学位条件）

发展：转专业、竞赛、考证、交换生、指引

科研：助理招募

未来：保研时间点和要做的事、毕业去向查询

## 功能特性

### 已实现功能
- ✅ 微信小程序登录认证
- ✅ 用户管理（注册、资料更新）
- ✅ 教师信息管理
- ✅ 教师点评功能（创建、查询、审核）
- ✅ 管理员权限控制
- ✅ RESTful API设计
- ✅ JWT Token认证
- ✅ 数据库设计与迁移

### 预留扩展功能
- 🔄 学业管理（课表、挂科率、选课、校历、期末资料、学分）
- 🔄 发展规划（转专业、竞赛、考证、交换生、指引）
- 🔄 科研助理招募
- 🔄 升学就业（保研时间点、毕业去向查询）

## 技术栈

- **语言**: Go 1.23+
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

- Go 1.23+
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

    a. 管理员会随时审核您的代码,除非你的PR设置为`draft`状态或标题内包含`WIP`/`Work In Progress`/🚧字样。

## 许可证

MIT License
