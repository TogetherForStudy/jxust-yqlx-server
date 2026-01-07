# GoJxust

GoJxust 是专为在校大学生设计的开源自托管服务平台，基于学习与生活中常见的需求出发，提供一系列提高效率、降低信息差的服务，让大学生能够节约有限时间，提高自我认知，创造无限价值。

## 亮点

🔐 **强幂等性保障**：基于Redis分布式锁的幂等性中间件，支持严格/宽松双模式，保障数据一致性

🎯 **精细权限控制**：RBAC角色权限系统 + 功能白名单双重保障，灵活应对复杂权限场景

⚡ **智能缓存加速**：Redis分布式缓存支持，幂等性响应缓存、权限快照缓存、在线人数统计等

🤖 **MCP协议支持**：原生支持Model Context Protocol，提供9+校园服务工具

📊 **实时在线统计**：基于Redis Sorted Set的在线人数统计，支持系统级和项目级统计

💎 **完整积分体系**：每日登录自动奖励、积分消费、交易记录、统计分析，激励用户活跃度

🌐 **RESTful API设计**：标准RESTful接口设计，统一响应格式，完善的错误处理，支持API版本控制

🔒 **安全认证机制**：JWT Token认证、微信小程序登录、请求ID追踪、CORS跨域支持，保障接口安全

📚 **丰富校园服务**：课程表、教师评价、挂科率、学习任务、倒数日、资料、刷题，一站式校园服务

## 技术栈

- **语言**: Go 1.24.1+
- **框架**: Gin Web Framework
- **数据库**: MySQL 8.0+、Redis
- **ORM**: GORM
- **认证**: JWT (golang-jwt/jwt)
- **配置**: 环境变量 + godotenv
- **日志**：Zap

## 快速部署

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

### Docker

1. 构建镜像: `make docker-build`
2. 运行容器: `docker run -p 8080:8080 gojxust`

### Docker Compose

创建一个新目录并将 docker-compose.yml 文件放入其中
在该目录下执行以下命令启动服务：
```docker-compose up -d```

## 贡献指南

1. Fork项目
2. 创建特性分支
3. 提交更改
4. 推送到分支
5. 创建Pull Request

    a. 管理员会随时审核您的代码,除非你的PR设置为`draft`状态或标题内包含`WIP`/`Work In Progress`/🚧字样，当然，你也可以手动at管理员/召唤gemini来review你的代码

## 许可证

MIT License
