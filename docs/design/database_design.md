# Database Design 数据库设计

基于 GORM 的数据库访问层设计，采用接口化设计实现高内聚低耦合。

## 文档更新记录

| Code | Module        | Date       | Author | PRI | Description                                |
|------|---------------|------------|--------|-----|--------------------------------------------|
| 1    | base-database | 2025-06-21 | AEnjoy | P0  | 初始设计文档创建,提供数据库访问层设计和接口定义 |

## 设计原则

1. **接口分离**: 所有数据库操作都通过接口定义，便于测试和扩展
2. **仓储模式**: 采用Repository模式封装数据访问层
3. **事务支持**: 统一的事务管理机制
4. **连接池优化**: 合理配置连接池参数
5. **错误处理**: 统一的错误处理和返回机制
6. **性能监控**: 支持SQL执行监控和性能分析
7. **迁移管理**: 数据库结构版本控制和迁移

## 核心功能

1. P0-**基础接口定义**:
   - CRUD操作接口
   - 分页查询支持
   - 软删除实现
   
2. P0-**事务管理**:
   - 事务接口定义
   - 事务传播机制
   - 事务回滚处理

3. P1-**性能优化**:
   - 连接池管理
   - 查询缓存
   - 批量操作支持
   
4. P2-**运维特性**:
   - 慢查询监控
   - 连接池状态
   - SQL执行统计

## 接口定义

### 1. 封装风格1:基础Repository接口

```go
type Repository[T any] interface {
    // 基础CRUD
    Create(ctx context.Context, entity *T) error
    Update(ctx context.Context, entity *T) error
    Delete(ctx context.Context, id uint) error
    FindByID(ctx context.Context, id uint) (*T, error)
    
    // 查询接口
    FindAll(ctx context.Context, page, size int) ([]T, int64, error)
    FindByCondition(ctx context.Context, condition *QueryCondition) ([]T, error)
    
    // 批量操作
    BatchCreate(ctx context.Context, entities []*T) error
    BatchUpdate(ctx context.Context, entities []*T) error
    BatchDelete(ctx context.Context, ids []uint) error
}
```

### 2. 封装风格2: 大型Repository接口(推荐)

```go
type IDb interface {
    Begin() (txn *gorm.DB)
    Commit(txn *gorm.DB)
    Rollback(txn *gorm.DB)

    // Common:
    Create(ctx context.Context,txn *gorm.DB, entity any) error
	Delete(ctx context.Context, txn *gorm.DB, entity any) error
	Update(ctx context.Context, txn *gorm.DB, entity any) error
    FindByID(ctx context.Context, txn *gorm.DB, id uint, entity any) error
    FindAll(ctx context.Context, txn *gorm.DB, page, size int, entities any) (int64, error)

    AddUser(ctx context.Context, txn *gorm.DB, user *User) error
	AddProfile(ctx context.Context, txn *gorm.DB, profile *Profile) error
	BandUserProfile(ctx context.Context, txn *gorm.DB, userID uint, profileID uint) error
    UnbindUserProfile(ctx context.Context, txn *gorm.DB, userID uint, profileID uint) error

    AddCourse(ctx context.Context, txn *gorm.DB, course *Course) error
	FindCourseByID(ctx context.Context, txn *gorm.DB, id uint) (*Course, error)
	SearchCourses(ctx context.Context, txn *gorm.DB, courseName string, page, size int) ([]Course, int64, error)

    AddOrder(ctx context.Context, txn *gorm.DB, order *Order) error
	// ...
}	
```

### 3. 事务接口

```go
type Transaction interface {
    // 事务操作
    Begin() error
    Commit() error
    Rollback() error
    
    // 事务传播
    InTransaction() bool
    GetTransaction() *gorm.DB
}
```

### 4. 数据库配置

```go
type DBConfig struct {
    Host         string
    Port         int
    Username     string
    Password     string
    Database     string
    MaxIdleConns int
    MaxOpenConns int
    MaxLifetime  time.Duration
    Debug        bool
}
```

## 实现示例

```go
type IUserRepository interface {
    Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	// ... 其他用户相关方法
}
// 用户仓储实现
type UserRepository struct {
    db *gorm.DB
}

func NewUserRepository(db *gorm.DB/* db object or txn object*/) Repository[User] {
    return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *User) error {
    return r.db.WithContext(ctx).Create(user).Error
}

// 事务示例
func (s *UserService) CreateUserWithProfile(ctx context.Context, user *User, profile *Profile) error {
    return s.transaction.Transaction(func(tx *gorm.DB) error {
        if err := s.userRepo.Create(ctx, user); err != nil {
            return err
        }
        profile.UserID = user.ID
        return s.profileRepo.Create(ctx, profile)
    })
}
```

## 测试用例

1. **CRUD测试**:
   - 基础操作验证
   - 事务操作验证
   - 并发操作测试

2. **性能测试**:
   - 连接池性能
   - 批量操作性能
   - 查询性能

3. **集成测试**:
   - 与业务层集成
   - 与缓存层集成
   - 错误处理验证

## 监控指标

1. **数据库指标**:
   - 连接池使用情况
   - 查询响应时间
   - 事务成功率

2. **性能指标**:
   - 慢查询统计
   - 查询命中率
   - 连接池等待时间

## 最佳实践

1. 使用**接口**进行**依赖注入**(方便测试和替换)
2. 统一错误处理和返回
3. 合理使用事务
4. 优化查询性能
5. 实现单元测试
6. 定期监控和优化
