# Message Queue Design 消息队列设计

基于 go-redis 实现的轻量级消息队列系统，支持竞争型消费者模式和发布订阅模式。

## 文档更新记录

| Code | Module    | Date       | Author | PRI | Description                            |
|------|-----------|------------|--------|-----|----------------------------------------|
| 1    | base-mq   | 2025-06-21 | AEnjoy | P0  | 初始设计文档创建,提供消息队列系统设计方案 |

## 设计原则

1. **可靠性**: 确保消息不丢失
2. **持久化**: 支持消息持久化存储
3. **有序性**: 保证消息顺序（同一队列内）
4. **幂等性**: 支持消息幂等处理
5. **可观测**: 完善的监控和追踪能力
6. **扩展性**: 支持水平扩展
7. **高性能**: 低延迟和高吞吐量

## 核心功能

1. P0-**竞争型消费者队列**:
   - 消息生产和消费
   - 消息确认机制
   - 死信队列支持
   
2. P0-**发布订阅系统**:
   - 主题订阅
   - 消息广播
   - 消息过滤

3. P1-**高级特性**:
   - 消息重试
   - 延迟队列
   - 优先级队列
   
4. P2-**运维功能**:
   - 队列监控
   - 性能统计
   - 消息追踪

## Topic定义:

所有的消息队列相关的 Topic 定义都集中在 `topics.go` 文件中，便于统一管理和维护。

[topics.go](../../pkg/constant/topics.go)

不同类型的topic使用不同的大`const`区块区分,并且每个区块都应包含相关的注释说明。

例如:

```go
// User相关的Topic定义
const (
    TopicUserCreated       = AppName + "/user/created"
	TopicUserUpdated       = AppName + "/user/updated"
)

// Order相关的Topic定义
const (
    TopicOrderCreated      = AppName + "/order/created"
    TopicOrderUpdated      = AppName + "/order/updated"
)
```

## 接口定义

### 1. 竞争型消费者接口

```go
type CompetingQueue interface {
    // 生产者接口
    Push(ctx context.Context, message Message) error
    BatchPush(ctx context.Context, messages []Message) error
    
    // 消费者接口
    Pop(ctx context.Context) (Message, error)
    BatchPop(ctx context.Context, count int) ([]Message, error)
    
    // 确认接口
    Ack(ctx context.Context, messageID string) error
    Nack(ctx context.Context, messageID string) error
}
```

### 2. 发布订阅接口

```go
type PubSubQueue interface {
    // 发布者接口
    Publish(ctx context.Context, topic string, message Message) error
    BatchPublish(ctx context.Context, topic string, messages []Message) error
    
    // 订阅者接口
    Subscribe(ctx context.Context, topic string) (<-chan Message, error)
    Unsubscribe(ctx context.Context, topic string) error
}
```

以上两个接口可以合并为一个消息队列大接口。

### 3. 消息结构

```go
type Message struct {
    ID        string
    Topic     string
    Body      []byte
    Timestamp time.Time
    Attempts  int
    Metadata  map[string]string
}
```

## 整体接口：

```go
// Mq defines the interface for a message queue.
type Mq interface {
	// Publish sends a message to a specific topic.
	// All active subscribers on that topic will receive the message.
	Publish(ctx context.Context, topic string, msg []byte) error

	// Subscribe creates a subscription to a topic.
	// It returns a read-only channel where received messages (as 'any', typically []byte) will be sent.
	// Each subscriber instance receives a copy of the message.
	Subscribe(ctx context.Context, topic string) (<-chan any, error)

	// Unsubscribe removes the subscription for the given topic.
	// The channel returned by Subscribe will be closed.
	Unsubscribe(ctx context.Context, topic string) error

	// QueuePublish sends a message to a topic associated with a queue group.
	// Functionally often the same as Publish on the publisher side.
	QueuePublish(ctx context.Context, topic string, msg []byte) error

	// QueueSubscribe creates a subscription to a topic within a queue group.
	// Only one subscriber within the same queue group will receive a given message.
	// It returns a read-only channel where received messages (as 'any', typically []byte) will be sent.
	// The specific queue group name might be derived from the topic or configured internally.
	QueueSubscribe(ctx context.Context, topic string) (<-chan any, error)

	// QueueUnsubscribe removes the queue subscription for the given topic.
	// The channel returned by QueueSubscribe will be closed.
	QueueUnsubscribe(ctx context.Context, topic string) error

	// Close cleans up all resources, unsubscribes from all topics, and closes the connection.
	Close()

	// SetConditions allows configuring parameters like channel buffer capacity.
	SetConditions(capacity int)
}
```


## 实现示例

### 1. 竞争型消费者实现

```go
type RedisCompetingQueue struct {
    client *redis.Client
    name   string
}

func NewRedisCompetingQueue(client *redis.Client, name string) CompetingQueue {
    return &RedisCompetingQueue{
        client: client,
        name:   name,
    }
}

func (q *RedisCompetingQueue) Push(ctx context.Context, message Message) error {
    // 实现消息入队
    return q.client.LPush(ctx, q.name, message).Err()
}

func (q *RedisCompetingQueue) Pop(ctx context.Context) (Message, error) {
    // 实现消息出队，带确认机制
    return q.client.BRPopLPush(ctx, q.name, q.getPendingKey(), 0).Result()
}
```

### 2. 发布订阅实现

```go
type RedisPubSubQueue struct {
    client *redis.Client
}

func NewRedisPubSubQueue(client *redis.Client) PubSubQueue {
    return &RedisPubSubQueue{client: client}
}

func (q *RedisPubSubQueue) Publish(ctx context.Context, topic string, message Message) error {
    // 实现消息发布
    return q.client.Publish(ctx, topic, message).Err()
}

func (q *RedisPubSubQueue) Subscribe(ctx context.Context, topic string) (<-chan Message, error) {
    // 实现消息订阅
    pubsub := q.client.Subscribe(ctx, topic)
    return q.createMessageChannel(ctx, pubsub), nil
}
```

## 消息处理

### 1. 消息重试机制

```go
type RetryStrategy interface {
    ShouldRetry(attempts int) bool
    GetBackoff(attempts int) time.Duration
}

type ExponentialBackoff struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay time.Duration
}

func (b *ExponentialBackoff) GetBackoff(attempts int) time.Duration {
    delay := b.InitialDelay * time.Duration(math.Pow(2, float64(attempts)))
    if delay > b.MaxDelay {
        return b.MaxDelay
    }
    return delay
}
```

### 2. 死信队列处理

```go
type DeadLetterQueue struct {
    queue CompetingQueue
    maxAttempts int
}

func (dlq *DeadLetterQueue) ProcessMessage(msg Message) error {
    if msg.Attempts >= dlq.maxAttempts {
        return dlq.moveToDeadLetter(msg)
    }
    return nil
}
```

## 测试用例

1. **功能测试**:
   - 消息投递测试
   - 消息消费测试
   - 重试机制测试

2. **性能测试**:
   - 吞吐量测试
   - 延迟测试
   - 并发性能测试

3. **可靠性测试**:
   - 故障恢复测试
   - 消息持久化测试
   - 负载均衡测试

## 监控指标

1. **队列指标**:
   - 队列长度
   - 消息处理率
   - 消息延迟

2. **性能指标**:
   - 处理延迟
   - 内存使用
   - CPU使用率

## 最佳实践

1. 实现消息幂等性处理
2. 设置合理的重试策略
3. 监控死信队列
4. 实现优雅降级
5. 定期清理过期消息
6. 使用批量操作提高性能
