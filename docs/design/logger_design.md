# Logger Design 日志系统设计

基于 zap 库封装的高性能日志系统，支持应用日志和 GORM 日志记录。

## 文档更新记录

| Code | Module    | Date       | Author | PRI | Description                          |
|------|-----------|------------|--------|-----|--------------------------------------|
| 1    | base-log  | 2025-06-21 | AEnjoy | P0  | 初始设计文档创建,提供应用日志和GORM日志功能 |

## 设计原则

1. **性能优先**: 使用 uber-go/zap 库作为基础，确保高性能日志记录
2. **灵活配置**: 支持多环境配置（开发、测试、生产）
3. **分级记录**: 实现不同级别的日志记录（DEBUG, INFO, WARN, ERROR, FATAL）
4. **结构化输出**: 统一的JSON格式输出，便于日志采集和分析
5. **文件轮转**: 支持按大小或时间的日志文件轮转
6. **上下文跟踪**: 集成链路追踪ID，便于问题定位
7. **GORM集成**: 无缝对接GORM的日志系统

## 核心功能

1. P0-**应用日志记录**: 
   - 支持结构化日志输出
   - 支持日志级别控制
   - 支持日志文件轮转
   
2. P0-**GORM日志集成**:
   - SQL执行记录
   - 慢查询日志
   - 错误日志记录

3. P1-**性能指标**:
   - 日志写入延迟监控
   - 日志量统计
   
4. P2-**日志聚合**:
   - ELK集成支持
   - prometheus指标输出

## 实现示例

### 1. 基础日志器配置

```go
type LogConfig struct {
    Level      string // 日志级别
    Filename   string // 日志文件路径
    MaxSize    int    // 单个文件最大尺寸，单位MB
    MaxBackups int    // 最大保留文件数
    MaxAge     int    // 最大保留天数
    Compress   bool   // 是否压缩
}

func NewLogger(config LogConfig) (*zap.Logger, error) {
    // 配置实现
}
```

### 2. GORM日志适配器

```go
type GormLogger struct {
    ZapLogger *zap.Logger
    Config    *GormLogConfig
}

func NewGormLogger(zapLogger *zap.Logger) *GormLogger {
    // 实现GORM日志适配器
}
```

### 3. 使用示例

```go
// 初始化应用日志
logger := NewLogger(LogConfig{
    Level:      "info",
    Filename:   "logs/app.log",
    MaxSize:    100,
    MaxBackups: 3,
    MaxAge:     7,
    Compress:   true,
})

// 配置GORM日志
db.Logger = NewGormLogger(logger)

// 业务代码中使用
logger.Info("操作成功",
    zap.String("user", "admin"),
    zap.Int("affected", 1),
)
```

## 测试用例

1. **基础功能测试**:
   - 验证不同级别日志输出
   - 验证日志文件轮转
   - 验证JSON格式正确性

2. **GORM日志测试**:
   - SQL执行日志记录
   - 慢查询识别
   - 错误日志捕获

3. **性能测试**:
   - 并发写入性能
   - 文件IO性能
   - 内存使用情况

## 监控指标

1. **日志量指标**:
   - 每秒写入条数
   - 日志级别分布
   - 文件大小变化

2. **性能指标**:
   - 写入延迟
   - 内存使用
   - IO等待时间

## 最佳实践

1. 在生产环境中使用JSON格式输出
2. 合理设置日志级别，避免无用日志
3. 配置适当的文件轮转策略
4. 使用结构化字段，便于后续分析
5. 集成链路追踪ID
