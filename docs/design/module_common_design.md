# Common Module Design 普通模块设计

普通模块设计是构建高内聚低耦合的应用程序的重要部分。

它包含了一些通用RestfulAPI接口和工具函数，旨在提供基础设施支持和通用功能。

## 文档更新记录

| Code | Module | Date       | Author | PRI | Description |
|------|--------|------------|--------|-----|-------------|
| 1    | init   | 2025-06-21 | AEnjoy | P0  | 初始设计文档创建    |


## 设计原则

1. **高内聚低耦合**: 模块之间通过接口和事件解耦，确保模块独立性(可以独立测试和部署)
   1. 对于独立的模块应用，可以提供清晰的接口定义(gRPC/HTTP API)，使得模块之间的交互简单明了
   2. 对于集成的模块应用，可以通过事件总线或消息队列实现模块间的异步通信，降低耦合度
2. **通用性**: 提供通用的RestfulAPI接口和工具函数，便于复用
3. **易扩展性**: 设计时考虑未来可能的扩展需求
4. **性能优化**: 优化常用操作的性能，减少不必要的开销
5. **安全性**: 确保接口和数据传输的安全性，防止未授权访问
6. **可测试性**: 提供易于测试的接口和工具函数，支持单元测试和集成测试
7. **代码共享**: 提供公共的代码库和工具函数，便于团队代码复用
   1. 通用的工具函数库位于 `pkg/utils` 目录下
   2. 通用的RestfulAPI接口位于 `pkg/api` 目录下
   3. 通用的配置和日志系统位于 `pkg/config` 和 `pkg/logger` 目录下
   4. 通用的错误处理位于 `pkg/errors` 目录下
   5. 通用的业务码位于 `pkg/code` 目录下

## 核心功能

1. P0-**通用RestfulAPI接口**:
   - 容器化应用的健康检查接口 /health
   - (P2)Prometheus追踪接口 /metrics 和 /healthz
   - 后端程序版本信息接口 /version
   - 上述接口不需要认证和授权，但仅`后端程序版本信息接口`提供公共访问
2. P0-**通用工具函数**:
   - 信号处理、加密解密、随机ID等常用工具函数
   - 配置加载和解析工具
   - 日志记录工具
3. P0-**业务错误码**:
    - 定义通用的业务错误码格式和错误处理机制
    - 提供统一的错误响应格式

## 接口定义

### 健康检查接口

健康检查接口是容器化应用的基础接口，用于检查应用是否正常运行。

一般只需要提供一个简单的 `/health` 接口，返回 HTTP 200 状态码即可表示应用正常。

```go
func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
    // w.Write([]byte(`{"msg":"OK"}`))
}
```

### Prometheus追踪接口

Prometheus追踪接口用于提供应用的性能指标和健康状态。

一般提供两个接口：`/metrics` 和 `/healthz`。

请使用 `github.com/prometheus/client_golang/prometheus` 包来实现。

```go
func MetricsHandler(w http.ResponseWriter, r *http.Request) {
    // 使用 Prometheus 提供的 Handler
    promhttp.Handler().ServeHTTP(w, r)
}
func HealthzHandler(w http.ResponseWriter, r *http.Request) {
    // 返回应用健康状态
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Healthy"))
}
```

### 版本信息接口

程序的版本信息接口用于提供应用的版本信息。

其包含的内容位于 `/pkg/version/version.go`

该接口返回应用的版本、Git提交哈希、特性标识、构建时间、Go版本、构建主机平台和平台版本等信息。

版本信息内容包含以下部分:

```go
const DefaultVersion = "dev"

var (
	ServiceName       string
	Version           = DefaultVersion
	GitCommit         = DefaultVersion
	Features          = DefaultVersion
	BuildTime         string
	GoVersion         string
	BuildHostPlatform string
	PlatformVersion   string
)
```

响应JSON示例:

```json
{
  "version":"dev",
  "commit": "c86b329",
  "features": "main",
  "build_time": "2025-06-21T12:00:00Z",
  "go_version": "go1.20.3",
  "build_host_platform": "linux/amd64",
  "platform_version": "Debian 12 Bookworm"
}
```

上述信息在编译时由CI/CD系统注入。

```dockerfile
ARG VERSION
ARG BUILD_TIME
ARG GIT_COMMIT
ARG FEATURES
ARG BUILD_HOST_PLATFORM
ARG PLATFORM_VERSION
ARG CGO_ENABLED

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY . .

WORKDIR /app/cmd/apiserver
RUN GO_VERSION=`go version | awk '{print $3}'` && CGO_ENABLED=$CGO_ENABLED && \
	go build -v -tags=sonic -tags=avx -ldflags "\
	-w -s \
	-X 'github.com/TogetherForStudy/jxust-yqlx-server/pkg/version.Version=${VERSION}' \
	-X 'github.com/TogetherForStudy/jxust-yqlx-server/pkg/version.BuildTime=${BUILD_TIME}' \
	-X 'github.com/TogetherForStudy/jxust-yqlx-server/pkg/version.GoVersion=$GO_VERSION' \
	-X 'github.com/TogetherForStudy/jxust-yqlx-server/pkg/version.GitCommit=${GIT_COMMIT}' \
	-X 'github.com/TogetherForStudy/jxust-yqlx-server/pkg/version.Features=${FEATURES}' \
	-X 'github.com/TogetherForStudy/jxust-yqlx-server/pkg/version.BuildHostPlatform=${BUILD_HOST_PLATFORM}' \
	-X 'github.com/TogetherForStudy/jxust-yqlx-server/pkg/version.PlatformVersion=${PLATFORM_VERSION}' \
    " -o /apiserver
```

### 工具函数

工具函数提供了一些常用的功能，如信号处理、加密解密、随机ID生成等。

### 业务错误码

业务错误码用于定义通用的业务错误码格式和错误处理机制。

业务错误码的设计应遵循以下原则：
1. **唯一性**: 每个错误码应唯一标识一个特定的错误类型
2. **可读性**: 错误码的错误解释内容应易于理解，便于开发人员和用户识别
3. **可扩展性**: 错误码应支持未来的扩展需求
4. **一致性**: 错误码的格式和命名应保持一致

错误码长度: 7位 XAABBCC (数字 如 1100001)

其中前三位表示模块编号(X表示占位(当前为1))，后四位表示具体错误码。

新增的错误码需按递增顺序排列(即新增的错误码应大于同组现有的最大错误码)。

AA表示模块编号 Code，BBCC表示具体错误码。

| Code | Module Name                      |
|------|----------------------------------|
| 11_  | System or Common Module          |
| 12_  | Async Job                        |
| 13_  | WeChat                           |
| 14_  | User   Module                    |
| 15_  | Course and Exam Module           |
| 16_  | Teacher Review Module            |
| 17_  | Academic management              |
| 18_  | Development planning             |
| 19_  | Further education and employment |
| 20_  | Academic Affairs System          |
| 30_  | Other Module                     |

```go
type ResCode int64

func (c ResCode) GetMsg() string {
    msg, ok := StatusMsgMap[c]
    if !ok {
        return StatusMsgMap[CommonErrorUnknown]
    }
    return msg
}

const EmptyValue ResCode = 0

// Common
const (
    CommonSuccess ResCode = 1100000 + iota
	_
    CommonErrorBadRequest
    CommonErrorNotFound
    CommonErrorInternalServerError
    CommonErrorUnknown
    CommonErrorIO
    CommonDeadLine
    CommonErrorPushTaskFailed
)

// Async Job
const (
    JobSuccess ResCode = 1200000 + iota
	JobErrorNotFound
)

// ...

var StatusMsgMap = map[ResCode]string{
    EmptyValue:"unknown",
	
    CommonSuccess: "操作成功",
    CommonErrorBadRequest: "请求参数错误",
    CommonErrorNotFound: "资源未找到",
    CommonErrorInternalServerError: "服务器内部错误",
    CommonErrorUnknown: "未知错误",
    CommonErrorIO: "IO错误",
    CommonDeadLine: "操作超时",
    CommonErrorPushTaskFailed: "任务推送失败",

    JobSuccess: "任务执行成功",
    JobErrorNotFound: "任务未找到",

    // 其他错误码...
}
```

## 测试Case
1. 不需要鉴权
2. 健康检查接口返回200
3. Prometheus追踪接口返回200
4. 版本信息接口返回正确格式
5. 工具函数正常工作
6. 业务错误码符合逻辑且正常工作
