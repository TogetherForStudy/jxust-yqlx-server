# 事故复盘：MySQL Error 1040 Too Many Connections

**日期**：2026-06-29  
**影响范围**：全站所有用户，所有接口返回 Error 1040  
**触发操作**：在生产环境新增一个题目 Project  
**根因类别**：级联故障（Cascading Failure）— Redis 键缺失 → DB 回退 → 连接耗尽 → Worker 重试风暴 → 全站崩溃

---

## 事故时间线

| 时间点 | 事件 |
|---|---|
| T0 | 管理员通过 Admin API 新增一个 `QuestionProject` |
| T0 + N | 小程序首页请求项目列表（`GET /api/v0/questions/projects`），部分用户开始遇到慢响应 |
| T0 + N+1 | MySQL 连接池耗尽，出现 `Error 1040: Too many connections` |
| T0 + N+2 | Sync Worker 写 DB 也失败，`task_retry_pushed` 日志持续刷屏，进入重试风暴 |
| T0 + N+3 | 全站所有接口返回 Error 1040 |
| T0 + N+4 | 尝试重启服务 → `InitProjectRedisData` 查询 MySQL 也失败 → 所有项目的 Redis 键都缺失 → 问题加剧 |

---

## 根因分析

### 完整故障链路

```
管理员新增 QuestionProject
    │
    ▼
CreateAdminQuestionProject 没有初始化 Redis 键
(project:usage:{new_id} 不存在)
    │
    ▼
用户请求 GET /api/v0/questions/projects
    │
    ▼
question_service.go:GetProjects
    │
    ├─ GetEnabledProjectFeatures (features LIKE 'review.project.%')  ← OK
    ├─ GetUserFeatures (JSON_CONTAINS 全表扫描 features 表)         ← 慢
    │
    └─ for each visible project:
         ├─ SCard(project:users:{id})   → key 不存在，返回 0 ✓
         └─ GetInt(project:usage:{id})  → key 不存在，返回 redis.Nil 错误 ✗
              │
              ▼
         错误处理: if err != nil → DB 回退查询
         SELECT COALESCE(SUM(study_count + practice_count), 0)
         FROM user_question_usages uq
         JOIN questions q ON q.id = uq.question_id
         WHERE q.project_id = ? AND q.is_active = ?
              │
              ▼
         每次请求触发额外 DB 查询 → 连接持有时间变长
              │
              ▼
         大量并发用户打开小程序首页 → 连接池耗尽
              │
              ▼
         MySQL Error 1040: Too many connections
              │
              ├──▶ Sync Worker 写 DB 也失败 → task_retry_pushed
              │    → 立即重试（无退避）→ 再次 1040 → 死循环
              │    → 连接压力持续无法恢复
              │
              └──▶ 全站所有接口返回 1040 → 服务完全不可用
```

### 三个根本原因

#### 1. `GetInt` 对不存在的 Redis 键返回错误（核心缺陷）

**文件**：`internal/pkg/cache/redis.go:92`

```go
// 修复前
func (r *redisCache) GetInt(ctx context.Context, key string) (int64, error) {
    val, err := r.cli.GetRedisCli().Get(ctx, key).Result()
    if err != nil {
        return 0, err  // redis.Nil 被当作故障传播
    }
    // ...
}
```

Redis `GET` 对不存在的键返回 `redis.Nil`，但 `GetInt` 的调用方（`GetProjects`）将所有错误都视为"Redis 不可用"并回退到 DB 查询。这与 `SCard` 的行为不一致——`SCard` 对不存在的集合返回 `(0, nil)`。

#### 2. 创建项目时未初始化 Redis 键（触发条件）

**文件**：`internal/services/question_admin_service.go:71`

```go
// 修复前
func (s *QuestionService) CreateAdminQuestionProject(...) {
    // ... DB insert ...
    // ❌ 没有初始化 Redis 键
    return &resp, nil
}
```

`InitProjectRedisData()` 仅在服务启动时运行一次。运行期间通过 API 创建的新项目，其 Redis 键永远不会被初始化。

#### 3. Worker 无退避重试（放大器）

**文件**：`internal/worker/worker.go:144`

```go
// 修复前：失败后立即 LPUSH 回队列，下次轮询（≤5s）马上重试
func (w *Worker) handleTaskError(task Task, taskData string, err error) {
    if retryCount < w.config.MaxRetries {
        task.IncrementRetry()
        w.queue.Push(...)  // 立即重入队，无延迟
    }
}
```

当 MySQL 连接耗尽时，Worker 的每次重试都立即失败并重新入队，形成**紧循环重试风暴**，阻断了 DB 连接池的自然恢复。

---

## 修复方案

### 修复 1：`GetInt` 对缺失键返回 0（消除根因）

**文件**：`internal/pkg/cache/redis.go`

```go
func (r *redisCache) GetInt(ctx context.Context, key string) (int64, error) {
    val, err := r.cli.GetRedisCli().Get(ctx, key).Result()
    if err != nil {
        // 键不存在时返回 0，避免调用方误将 redis.Nil 当作 Redis 故障而回退到 DB
        if errors.Is(err, rediscache.Nil) {
            return 0, nil
        }
        return 0, err
    }
    // ...
}
```

**效果**：新项目的 `project:usage:{id}` 键不存在时，`GetInt` 直接返回 0，不再穿透到 DB。同时保持了真正 Redis 故障时的错误传播。

### 修复 2：创建项目时初始化 Redis 键（防止复发）

**文件**：`internal/services/question_admin_service.go`

```go
func (s *QuestionService) CreateAdminQuestionProject(ctx context.Context, ...) {
    // ... DB insert ...

    // 初始化 Redis 中的项目数据，避免 GetProjects 因键缺失回退到 DB 查询
    s.initProjectRedisKeys(ctx, project.ID)
    // ...
}

func (s *QuestionService) initProjectRedisKeys(ctx context.Context, projectID uint) {
    if cache.GlobalCache == nil || projectID == 0 {
        return
    }
    noExpiration := time.Duration(0)
    usageKey := fmt.Sprintf("project:usage:%d", projectID)
    _ = cache.GlobalCache.Set(ctx, usageKey, "0", &noExpiration)
}
```

**效果**：新项目创建后立即拥有 Redis 键，不再依赖启动时的批量初始化。

### 修复 3：Worker 指数退避重试（切断死亡螺旋）

**文件**：`internal/worker/worker.go`

```go
func (w *Worker) handleTaskError(task Task, taskData string, err error) {
    if retryCount < w.config.MaxRetries {
        // 指数退避：1s → 2s → 4s（第 1/2/3 次重试）
        backoff := time.Duration(math.Pow(2, float64(retryCount))) * time.Second
        // ... 等待或 context 取消 ...
        select {
        case <-time.After(backoff):
        case <-w.ctx.Done():
            return
        }
        // 退避完成后再入队
        w.queue.Push(...)
    }
}
```

**效果**：重试间隔从 0s 变为 1s/2s/4s，给 DB 留出恢复窗口。日志标记改为 `task_retry_waiting`，携带 `backoff_secs` 字段便于监控。

### 修复 4：启动时 DB 查询加重试（启动容错）

**文件**：`internal/bootstrap/redis.go`

```go
func InitProjectRedisData(db *gorm.DB) {
    // ...
    const maxRetries = 3
    for i := 0; i < maxRetries; i++ {
        if err := db.Where("is_active = ?", true).Select("id").Find(&projects).Error; err != nil {
            if i < maxRetries-1 {
                time.Sleep(time.Duration(i+1) * time.Second)
                continue
            }
            // 3 次全部失败则降级（修复 1 兜底：GetInt 缺键返回 0）
            return
        }
        break
    }
    // ...
}
```

**效果**：启动时 DB 暂不可用会自动重试，避免一次性失败导致所有项目缺键。

---

## 多层防御体系

```
┌─────────────────────────────────────────────────────────────┐
│                     防御层 ①（根因消除）                       │
│  GetInt 缺键返回 0 → 永久消除 "缺键→DB回退" 的因果链           │
├─────────────────────────────────────────────────────────────┤
│                     防御层 ②（预防复发）                       │
│  创建项目时初始化 Redis 键 → 新项目不再缺键                     │
├─────────────────────────────────────────────────────────────┤
│                     防御层 ③（切断风暴）                       │
│  Worker 指数退避重试 → DB 故障时不会形成自激振荡                │
├─────────────────────────────────────────────────────────────┤
│                     防御层 ④（启动容错）                       │
│  InitProjectRedisData 重试 3 次 → 启动时短暂故障可自愈          │
│  失败后降级（依赖防御层①兜底）                                  │
└─────────────────────────────────────────────────────────────┘
```

---

## 后续建议

### 短期

1. **为 `features` 表的 `feature_key` 列确认索引存在** — `GetEnabledProjectFeatures` 使用 `LIKE 'review.project.%'`，需要 B-tree 索引支持前缀匹配
2. **监控 MySQL 连接数** — 添加 `SHOW PROCESSLIST` 的定时采集，连接数超过 80% 时告警
3. **确认 `user_question_usages.question_id` 和 `questions.project_id` 有索引** — 确保 DB 回退查询（即使很少触发）不会变成慢查询

### 中长期

4. **`GetUserFeatures` 的 `JSON_CONTAINS` 查询考虑加 MySQL 多值索引**（MySQL 8.0.17+）或改用关联表，消除全表扫描
5. **连接池配置外部化** — 将 `SetMaxOpenConns` / `SetMaxIdleConns` 放入配置文件，便于按环境调优
6. **为关键接口添加 Prometheus 指标** — 请求延迟、DB 连接等待时间、Redis 命中率
