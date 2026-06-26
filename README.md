# GoJxust V1.4.0

GoJxust 是专为江西理工大学学生设计的开源自托管校园服务平台，现本项目为江理一起来学小程序的后端服务。从学习与生活中的常见需求出发，提供一系列提高效率、降低信息差的服务，帮助大学生节约时间、提升自我。

## 功能亮点

### 核心架构

| 模块                 | 说明                                                                   |
| -------------------- | ---------------------------------------------------------------------- |
| 🔐 **幂等性保障**    | 基于 Redis 分布式锁的幂等性中间件，支持严格/宽松双模式，防止重复提交   |
| 🎯 **RBAC 权限系统** | 六级角色（guest → admin）+ 功能白名单 + 功能开关，灵活应对复杂权限场景 |
| ⚡ **智能缓存加速**  | Redis 分布式缓存：幂等性响应缓存、权限快照缓存、在线人数统计等         |
| 🔒 **双 Token 认证** | JWT Access + Refresh Token，微信小程序登录，请求 ID 全链路追踪         |
| 📊 **实时在线统计**  | Redis Sorted Set 实现系统级/项目级在线人数，管理端单接口聚合查询       |

### 校园服务

| 服务            | 说明                             |
| --------------- | -------------------------------- |
| 📅 **课程表**   | 自定义课表管理，支持导入/重置    |
| 📝 **教师评价** | 选课评价、教师评分、文本评价     |
| 📉 **挂科率**   | 课程挂科率查询与统计分析         |
| 📖 **词汇学习** | 四六级等词汇学习与自测           |
| 🍅 **番茄钟**   | 专注计时 + 排行 + 学习统计       |
| 📋 **学习任务** | 待办事项管理与跟踪               |
| ⏳ **倒数日**   | 考试/活动倒计时提醒              |
| 📁 **资料共享** | MinIO 对象存储，文件上传下载     |
| ✍️ **刷题**     | 在线刷题 + 答题统计              |
| 🏫 **组织管理** | 班级/社团等组织成员管理          |
| 💎 **积分体系** | 每日签到奖励、积分消费、交易记录 |

### 管理与运维

| 功能                | 说明                                                  |
| ------------------- | ----------------------------------------------------- |
| 🖥️ **后台管理**     | Web 管理界面登录，课表/挂科率/刷题统计等管理接口      |
| 📄 **OpenAPI 文档** | 自动生成 OpenAPI 规范文档                             |
| 💾 **GPA 备份**     | GPA 数据备份与恢复                                    |
| 🤖 **MCP 协议**     | 原生支持 Model Context Protocol，提供校园服务工具接口 |

## 技术栈

| 类别     | 技术                  |
| -------- | --------------------- |
| 语言     | Go 1.26.3+            |
| Web 框架 | Gin                   |
| 数据库   | MySQL 8.0+            |
| 缓存     | Redis                 |
| ORM      | GORM                  |
| 对象存储 | MinIO（S3 兼容）      |
| 认证     | JWT（golang-jwt/jwt） |
| 日志     | Zap                   |
| 配置     | 环境变量 + YAML       |

## 项目结构

```
goJxust/
├── cmd/apiserver/          # 应用入口
│   ├── main.go             # 主程序
│   ├── .env.example        # 环境变量模板
│   └── Dockerfile
├── internal/
│   ├── bootstrap/          # 启动初始化
│   ├── config/             # 配置管理
│   ├── database/           # 数据库连接与迁移
│   ├── handlers/           # HTTP 处理器（Controller）
│   ├── services/           # 业务逻辑层
│   ├── models/             # 数据模型（GORM）
│   ├── dto/                # 请求/响应 DTO
│   ├── middleware/         # 中间件（认证、RBAC、幂等等）
│   ├── router/             # 路由注册
│   ├── scheduler/          # 定时任务
│   ├── worker/             # 异步任务
│   └── pkg/                # 内部公共包
├── pkg/
│   ├── constant/           # 全局常量
│   ├── logger/             # 日志封装
│   ├── minio/              # MinIO 客户端
│   └── utils/              # 工具函数
├── docs/                   # 设计文档与 API 文档
├── scripts/                # SQL 初始化、E2E 测试等脚本
├── deployment/             # Kubernetes 部署配置
├── Makefile
└── docker-compose.yml
```

## 快速开始

### 环境要求

- Go 1.26+
- MySQL 8.0+
- Redis（幂等性、缓存等功能依赖）
- MinIO（文件存储，可选）

### 1. 克隆项目

```bash
git clone https://github.com/TogetherForStudy/jxust-yqlx-server.git
cd jxust-yqlx-server
```

### 2. 环境配置

```bash
cp cmd/apiserver/.env.example .env
vim .env    # 填写数据库、Redis、微信小程序等配置
```

### 3. 安装依赖

```bash
go mod tidy
```

### 4. 启动服务

```bash
go run cmd/apiserver/main.go
```

应用默认在 `http://localhost:8080` 启动。

## 部署

### Docker

```bash
make docker-build                   # 构建镜像
docker run -p 8080:8080 gojxust     # 运行容器
```

### Docker Compose

```bash
docker-compose up -d
```

### Kubernetes

配置文件位于 `deployment/` 目录。

## 开发指南

### 常用命令

```bash
make build-apiserver    # 构建 Linux 二进制
make test               # 运行单元测试
make test-coverage      # 测试 + 覆盖率
make clean              # 清理构建产物
```

### E2E 测试

```bash
# 初始化 RBAC 数据（首次）
mysql -u user -p database < scripts/init_rbac.sql

# 安装依赖
pip install httpx

# 运行（需先启动服务，且 GIN_MODE 非 release）
python scripts/e2e_test.py
```

### 添加新接口

1. `internal/dto/` — 定义请求/响应结构体
2. `internal/models/` — 添加数据模型（如需）
3. `internal/services/` — 实现业务逻辑
4. `internal/handlers/` — 编写 Handler
5. `internal/router/router.go` — 注册路由与中间件
6. 按需在 RBAC 中配置权限标签

### 换行符规范

项目统一使用 **LF** 换行符，已通过 `.gitattributes` 强制配置。

## 贡献指南

1. Fork 本仓库
2. 基于 `main` 创建特性分支
3. 遵循 [Conventional Commits](https://www.conventionalcommits.org/) 提交
4. 推送分支并创建 Pull Request

   a. 管理员会随时审核您的代码,除非你的PR设置为`draft`状态或标题内包含`WIP`/`Work In Progress`/🚧字样，当然，你也可以手动at管理员/召唤gemini来review你的代码

## 许可证

MIT License
