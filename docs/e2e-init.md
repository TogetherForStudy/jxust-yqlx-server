# 数据库初始化说明

## RBAC 权限系统初始化

在运行E2E测试之前，需要先初始化数据库的RBAC权限系统数据。

### 方法一：使用SQL脚本初始化（推荐）

```bash
# 执行SQL初始化脚本
mysql -u your_username -p your_database < scripts/init_rbac.sql
```

### 方法二：程序启动时自动初始化

修改 `cmd/apiserver/main.go`，在数据库迁移后添加RBAC初始化：

```go
// 自动迁移数据库表
if err := database.AutoMigrate(db); err != nil {
    logger.Fatalf("Failed to migrate database: %v", err)
}

// 初始化RBAC权限系统
rbacService := services.NewRBACService(db, nil)
if err := rbacService.SeedDefaults(context.Background()); err != nil {
    logger.Warnf("Failed to seed RBAC defaults: %v", err)
}
```

## E2E测试

初始化完成后，运行E2E测试：

```bash
# 确保服务运行在非release模式（启用mock登录端点）
# 在 .env 中设置: GIN_MODE=debug

# 启动服务
go run cmd/apiserver/main.go

# 在另一个终端运行E2E测试
python3 scripts/e2e_test.py --base-url http://localhost:8085
```

## 角色说明

系统预置了以下角色：

- `user_basic`: 基本用户，默认角色，拥有常规读写权限
- `user_active`: 活跃用户，活跃度达标解锁，额外权限
- `user_verified`: 认证用户，完成校内身份认证
- `operator`: 运营，拥有内容管理权限
- `admin`: 管理员，拥有全部权限

## 测试用户类型

Mock登录支持的测试用户类型：

- `basic`: 基本用户
- `active`: 活跃用户  
- `verified`: 认证用户
- `operator`: 运营
- `admin`: 管理员
