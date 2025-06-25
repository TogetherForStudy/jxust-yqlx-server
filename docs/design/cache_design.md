# Cache Design 缓存系统设计

基于 go-redis 库实现的高性能缓存系统，支持幂等性请求缓存加速。

## 文档更新记录

| Code | Module      | Date       | Author | PRI | Description                        |
|------|-------------|------------|--------|-----|------------------------------------|
| 1    | base-cache  | 2025-06-21 | AEnjoy | P0  | 初始设计文档创建,提供缓存系统设计方案 |

## 设计原则

1. **透明访问**: 缓存层对业务层透明
2. **一致性**: 确保缓存与数据库的数据一致性
3. **过期策略**: 灵活的缓存过期策略
4. **防击穿**: 使用互斥锁防止缓存击穿
5. **防穿透**: 通过空值缓存防止缓存穿透
6. **防雪崩**: 采用随机过期时间防止缓存雪崩
7. **监控**: 完善的缓存监控指标

## 核心功能

1. P0-**基础缓存**:
   - GET/SET操作
   - 过期时间设置
   - 批量操作支持
   
2. P1-**高级特性**:
   - 分布式锁
   - 原子操作
   - Pipeline支持

3. P2-**缓存策略**:
   - LRU淘汰
   - 主动更新
   - 被动更新
   
4. P3-**监控运维**:
   - 命中率统计
   - 容量监控
   - 性能分析

## 接口定义

### 1. 缓存接口

```go
type Cache interface {
    // 基础操作
    Get(ctx context.Context, key string) (string, error)
    Set(ctx context.Context, key string, value any, expiration time.Duration) error
    Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
    
    // 高级特性
    Lock(key string, expiration time.Duration) (bool, error)
    Unlock(key string) error
    
    // 计数器
    Incr(key string) (int64, error)
    Decr(key string) (int64, error)
}
```

### 2. 缓存配置

```go
type RedisConfig struct {
    Host            string
    Port            int
    Password        string
    DB              int
    PoolSize        int
    MinIdleConns    int
    MaxRetries      int
    ConnMaxLifetime time.Duration
}
```

## 实现示例

### 1. Redis缓存实现

```go
type RedisCache struct {
    client *redis.Client
}

func NewRedisCache(config *CacheConfig) Cache {
    client := redis.NewClient(&redis.Options{
        Addr:         fmt.Sprintf("%s:%d", config.Host, config.Port),
        Password:     config.Password,
        DB:           config.DB,
        PoolSize:     config.PoolSize,
        MinIdleConns: config.MinIdleConns,
    })
    return &RedisCache{client: client}
}

// 实现接口方法
func (c *RedisCache) Get(ctx context.Context, key string) (string, error) {
    return c.client.Get(ctx, key).Result()
}
```

### 2. 缓存装饰器（可选）

```go
type CacheDecorator struct {
    cache Cache
    next  Service
}

func (d *CacheDecorator) GetUser(ctx context.Context, id uint) (*User, error) {
    // 尝试从缓存获取
    if user, err := d.getFromCache(ctx, id); err == nil {
        return user, nil
    }
    
    // 缓存未命中，从服务获取
    user, err := d.next.GetUser(ctx, id)
    if err != nil {
        return nil, err
    }
    
    // 写入缓存
    d.setToCache(ctx, id, user)
    return user, nil
}
```

## 缓存策略

1. **缓存更新策略**:
   ```go
   // 更新策略接口
   type UpdateStrategy interface {
       Update(key string, value interface{}) error
   }
   
   // Cache-Aside策略
   type CacheAsideStrategy struct {
       cache Cache
       db    *gorm.DB
   }
   
   // Write-Through策略
   type WriteThroughStrategy struct {
       cache Cache
       db    *gorm.DB
   }
   ```

2. **过期策略**:
   ```go
   // 随机过期时间，防止缓存雪崩
   func getRandomExpiration(baseExpiration time.Duration) time.Duration {
       delta := time.Duration(rand.Int63n(int64(baseExpiration) / 4))
       return baseExpiration + delta
   }
   ```

## 测试用例

1. **功能测试**:
   - 基础操作测试
   - 并发访问测试
   - 过期策略测试

2. **性能测试**:
   - 吞吐量测试
   - 延迟测试
   - 内存使用测试

3. **可靠性测试**:
   - 故障恢复测试
   - 压力测试
   - 一致性测试

## 监控指标

1. **性能指标**:
   - 请求延迟
   - 命中率
   - QPS

2. **资源指标**:
   - 内存使用
   - 连接数
   - 网络流量

## 最佳实践

1. 设置合理的过期时间
2. 使用批量操作提高性能
3. 实现优雅降级机制
4. 定期清理过期数据
5. 监控缓存状态
6. 使用Pipeline减少网络往返
