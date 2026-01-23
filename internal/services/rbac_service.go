package services

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"
	json "github.com/bytedance/sonic"
	"gorm.io/gorm"
)

// RBACService 提供角色、权限、用户授权的核心能力
type RBACService struct {
	db    *gorm.DB
	cache cache.Cache
}

// UserPermissionSnapshot 缓存中的用户权限快照
type UserPermissionSnapshot struct {
	RoleTags       []string  `json:"role_tags"`
	PermissionTags []string  `json:"permission_tags"`
	IsAdmin        bool      `json:"is_admin"`
	CachedAt       time.Time `json:"cached_at"`
}

// NewRBACService 创建 RBAC 服务
func NewRBACService(db *gorm.DB) *RBACService {
	return &RBACService{
		db:    db,
		cache: cache.GlobalCache,
	}
}

// cacheKey 构建用户权限缓存 key
func (s *RBACService) cacheKey(userID uint) string {
	return fmt.Sprintf("rbac:user:%d:permissions", userID)
}

// SeedDefaults 初始化设计文档中预置的角色/权限与绑定关系
func (s *RBACService) SeedDefaults(ctx context.Context) error {
	roleSeeds := []models.Role{
		{RoleTag: models.RoleTagUserBasic, Name: "基本用户", Description: "默认角色"},
		{RoleTag: models.RoleTagUserActive, Name: "活跃用户", Description: "活跃度达标解锁"},
		{RoleTag: models.RoleTagUserVerified, Name: "认证用户", Description: "完成校内身份认证"},
		{RoleTag: models.RoleTagOperator, Name: "运营", Description: "运营"},
		{RoleTag: models.RoleTagAdmin, Name: "管理", Description: "管理"},
	}

	permissionSeeds := []models.Permission{
		{PermissionTag: models.PermissionUserGet, Name: "用户资料查看", Description: ""},
		{PermissionTag: models.PermissionUserUpdate, Name: "用户资料修改", Description: ""},
		{PermissionTag: models.PermissionOSSTokenGet, Name: "获取OSS Token", Description: ""},
		{PermissionTag: models.PermissionReviewCreate, Name: "发布点评", Description: ""},
		{PermissionTag: models.PermissionReviewGetSelf, Name: "查看本人点评", Description: ""},
		{PermissionTag: models.PermissionCourseTableGet, Name: "查看课表", Description: ""},
		{PermissionTag: models.PermissionCourseTableClassSearch, Name: "搜索班级", Description: ""},
		{PermissionTag: models.PermissionCourseTableClassUpdate, Name: "更新本人班级", Description: ""},
		{PermissionTag: models.PermissionCourseTableClassUpdateAll, Name: "管理员更新班级", Description: ""},
		{PermissionTag: models.PermissionCourseTableUpdate, Name: "更新个人课表", Description: ""},
		{PermissionTag: models.PermissionFailRate, Name: "挂科率查询", Description: ""},
		{PermissionTag: models.PermissionPointGet, Name: "积分查看", Description: ""},
		{PermissionTag: models.PermissionPointSpend, Name: "积分消费", Description: ""},
		{PermissionTag: models.PermissionPointManage, Name: "积分管理", Description: ""},
		{PermissionTag: models.PermissionContributionGet, Name: "投稿查看", Description: ""},
		{PermissionTag: models.PermissionContributionCreate, Name: "投稿创建", Description: ""},
		{PermissionTag: models.PermissionCountdown, Name: "倒数日", Description: ""},
		{PermissionTag: models.PermissionStudyTask, Name: "学习任务", Description: ""},
		{PermissionTag: models.PermissionMaterialGet, Name: "资料查看", Description: ""},
		{PermissionTag: models.PermissionMaterialRate, Name: "资料评分", Description: ""},
		{PermissionTag: models.PermissionMaterialDownload, Name: "资料下载", Description: ""},
		{PermissionTag: models.PermissionMaterialCategoryGet, Name: "资料分类查看", Description: ""},
		{PermissionTag: models.PermissionQuestion, Name: "刷题访问", Description: ""},
		{PermissionTag: models.PermissionPomodoro, Name: "番茄钟", Description: ""},
		{PermissionTag: models.PermissionDictionary, Name: "每日一词", Description: ""},

		{PermissionTag: models.PermissionReviewManage, Name: "点评管理", Description: ""},
		{PermissionTag: models.PermissionCourseTableManage, Name: "课表管理", Description: ""},
		{PermissionTag: models.PermissionHeroManage, Name: "英雄榜管理", Description: ""},
		{PermissionTag: models.PermissionConfigManage, Name: "配置管理", Description: ""},
		{PermissionTag: models.PermissionContributionManage, Name: "投稿管理", Description: ""},
		{PermissionTag: models.PermissionNotificationGet, Name: "通知后台查看", Description: ""},
		{PermissionTag: models.PermissionNotificationCreate, Name: "通知创建", Description: ""},
		{PermissionTag: models.PermissionNotificationPublish, Name: "通知发布", Description: ""},
		{PermissionTag: models.PermissionNotificationUpdate, Name: "通知更新", Description: ""},
		{PermissionTag: models.PermissionNotificationApprove, Name: "通知审核", Description: ""},
		{PermissionTag: models.PermissionNotificationSchedule, Name: "通知排期", Description: ""},
		{PermissionTag: models.PermissionNotificationPin, Name: "通知置顶/撤销", Description: ""},
		{PermissionTag: models.PermissionNotificationDelete, Name: "通知删除", Description: ""},
		{PermissionTag: models.PermissionNotificationPublishAdmin, Name: "通知直发", Description: ""},
		{PermissionTag: models.PermissionNotificationCategoryManage, Name: "通知分类管理", Description: ""},
		{PermissionTag: models.PermissionFeatureManage, Name: "功能管理", Description: ""},
		{PermissionTag: models.PermissionUserManage, Name: "用户管理", Description: ""},
		{PermissionTag: models.PermissionMaterialManage, Name: "资料管理", Description: ""},
	}

	roleBindings := map[string][]string{
		// 基础用户：常规读写
		models.RoleTagUserBasic: {
			models.PermissionUserGet,
			models.PermissionUserUpdate,
			models.PermissionOSSTokenGet,
			models.PermissionReviewCreate,
			models.PermissionReviewGetSelf,
			models.PermissionCourseTableGet,
			models.PermissionCourseTableClassSearch,
			models.PermissionCourseTableClassUpdate,
			models.PermissionCourseTableUpdate,
			models.PermissionFailRate,
			models.PermissionPointGet,
			models.PermissionPointSpend,
			models.PermissionContributionGet,
			models.PermissionContributionCreate,
			models.PermissionCountdown,
			models.PermissionStudyTask,
			models.PermissionMaterialGet,
			models.PermissionMaterialRate,
			models.PermissionMaterialDownload,
			models.PermissionMaterialCategoryGet,
			models.PermissionQuestion,
			models.PermissionPomodoro,
			models.PermissionDictionary,
		},
		models.RoleTagUserActive: {
			models.PermissionCourseTableClassUpdateAll,
		},
		// 运营：
		models.RoleTagOperator: {
			models.PermissionContributionManage,
			models.PermissionNotificationGet,
			models.PermissionNotificationCreate,
			models.PermissionNotificationPublish,
			models.PermissionNotificationUpdate,
			models.PermissionNotificationApprove,
			models.PermissionNotificationSchedule,
		},
	}

	// 创建/更新角色
	for _, role := range roleSeeds {
		var existing models.Role
		err := s.db.WithContext(ctx).Where("role_tag = ?", role.RoleTag).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := s.db.WithContext(ctx).Create(&role).Error; err != nil {
				return fmt.Errorf("初始化角色失败: %w", err)
			}
			continue
		}
		if err != nil {
			return err
		}
		if existing.Name != role.Name || existing.Description != role.Description {
			if err := s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
				"name":        role.Name,
				"description": role.Description,
			}).Error; err != nil {
				return fmt.Errorf("更新角色信息失败: %w", err)
			}
		}
	}

	// 创建/更新权限
	for _, perm := range permissionSeeds {
		var existing models.Permission
		err := s.db.WithContext(ctx).Where("permission_tag = ?", perm.PermissionTag).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			if err := s.db.WithContext(ctx).Create(&perm).Error; err != nil {
				return fmt.Errorf("初始化权限失败: %w", err)
			}
			continue
		}
		if err != nil {
			return err
		}
		if existing.Name != perm.Name || existing.Description != perm.Description {
			if err := s.db.WithContext(ctx).Model(&existing).Updates(map[string]any{
				"name":        perm.Name,
				"description": perm.Description,
			}).Error; err != nil {
				return fmt.Errorf("更新权限信息失败: %w", err)
			}
		}
	}

	// 绑定角色权限
	return s.bindRolePermissions(ctx, roleBindings)
}

// bindRolePermissions 按照绑定关系创建角色-权限关联
func (s *RBACService) bindRolePermissions(ctx context.Context, bindings map[string][]string) error {
	for roleTag, permTags := range bindings {
		var role models.Role
		if err := s.db.WithContext(ctx).Where("role_tag = ?", roleTag).First(&role).Error; err != nil {
			return fmt.Errorf("查询角色失败[%s]: %w", roleTag, err)
		}

		var perms []models.Permission
		if len(permTags) > 0 {
			if err := s.db.WithContext(ctx).
				Where("permission_tag IN ?", permTags).
				Find(&perms).Error; err != nil {
				return fmt.Errorf("查询权限失败: %w", err)
			}
		}

		if err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			if err := tx.Where("role_id = ?", role.ID).Delete(&models.RolePermission{}).Error; err != nil {
				return err
			}
			for _, perm := range perms {
				rp := models.RolePermission{
					RoleID:       role.ID,
					PermissionID: perm.ID,
				}
				if err := tx.Where("role_id = ? AND permission_id = ?", role.ID, perm.ID).
					FirstOrCreate(&rp).Error; err != nil {
					return err
				}
			}
			return nil
		}); err != nil {
			return fmt.Errorf("绑定角色权限失败[%s]: %w", roleTag, err)
		}
	}
	return nil
}

// ListRoles 获取角色列表
func (s *RBACService) ListRoles(ctx context.Context) ([]models.Role, error) {
	var roles []models.Role
	if err := s.db.WithContext(ctx).Find(&roles).Error; err != nil {
		return nil, err
	}
	return roles, nil
}

// ListRolesWithUsers 获取角色列表及其用户信息
func (s *RBACService) ListRolesWithUsers(ctx context.Context) ([]models.Role, map[uint]int, map[uint][]uint, error) {
	// 获取所有角色
	var roles []models.Role
	if err := s.db.WithContext(ctx).Find(&roles).Error; err != nil {
		return nil, nil, nil, err
	}

	// 构建角色ID到用户数量和用户ID列表的映射
	roleUserCountMap := make(map[uint]int)
	roleUserIDsMap := make(map[uint][]uint)

	// 为每个角色统计用户数量
	for _, role := range roles {
		var count int64
		if err := s.db.WithContext(ctx).
			Model(&models.UserRole{}).
			Where("role_id = ?", role.ID).
			Count(&count).Error; err != nil {
			return nil, nil, nil, err
		}
		roleUserCountMap[role.ID] = int(count)

		// 只为非 basic_user 角色获取用户ID列表
		if role.RoleTag != models.RoleTagUserBasic {
			var userIDs []uint
			if err := s.db.WithContext(ctx).
				Model(&models.UserRole{}).
				Where("role_id = ?", role.ID).
				Pluck("user_id", &userIDs).Error; err != nil {
				return nil, nil, nil, err
			}
			roleUserIDsMap[role.ID] = userIDs
		}
	}

	return roles, roleUserCountMap, roleUserIDsMap, nil
}

// ListPermissions 获取权限列表
func (s *RBACService) ListPermissions(ctx context.Context) ([]models.Permission, error) {
	var perms []models.Permission
	if err := s.db.WithContext(ctx).Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

// GetRolesWithPermissions 获取所有角色及其对应的权限列表
func (s *RBACService) GetRolesWithPermissions(ctx context.Context) ([]models.Role, map[uint][]models.Permission, error) {
	// 获取所有角色
	var roles []models.Role
	if err := s.db.WithContext(ctx).Find(&roles).Error; err != nil {
		return nil, nil, err
	}

	// 获取所有角色-权限关联
	var rolePermissions []models.RolePermission
	if err := s.db.WithContext(ctx).Find(&rolePermissions).Error; err != nil {
		return nil, nil, err
	}

	// 获取所有权限
	var permissions []models.Permission
	if err := s.db.WithContext(ctx).Find(&permissions).Error; err != nil {
		return nil, nil, err
	}

	// 构建权限ID到权限对象的映射
	permMap := make(map[uint]models.Permission)
	for _, perm := range permissions {
		permMap[perm.ID] = perm
	}

	// 构建角色ID到权限列表的映射
	rolePermMap := make(map[uint][]models.Permission)
	for _, rp := range rolePermissions {
		if perm, exists := permMap[rp.PermissionID]; exists {
			rolePermMap[rp.RoleID] = append(rolePermMap[rp.RoleID], perm)
		}
	}

	return roles, rolePermMap, nil
}

// CreateRole 创建角色
func (s *RBACService) CreateRole(ctx context.Context, role *models.Role) error {
	return s.db.WithContext(ctx).Create(role).Error
}

// UpdateRole 更新角色
func (s *RBACService) UpdateRole(ctx context.Context, id uint, updates map[string]any) error {
	return s.db.WithContext(ctx).Model(&models.Role{}).Where("id = ?", id).Updates(updates).Error
}

// DeleteRole 删除角色并清理关联
func (s *RBACService) DeleteRole(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var userIDs []uint
		if err := tx.Table("user_roles").
			Select("user_id").
			Where("role_id = ?", id).
			Find(&userIDs).Error; err != nil {
			return err
		}

		if err := tx.Where("role_id = ?", id).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		if err := tx.Where("role_id = ?", id).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		if err := tx.Delete(&models.Role{}, id).Error; err != nil {
			return err
		}
		for _, uid := range userIDs {
			s.invalidateUserCache(uid)
		}
		return nil
	})
}

// CreatePermission 创建权限
func (s *RBACService) CreatePermission(ctx context.Context, perm *models.Permission) error {
	return s.db.WithContext(ctx).Create(perm).Error
}

// UpdateRolePermissions 重置角色拥有的权限列表
func (s *RBACService) UpdateRolePermissions(ctx context.Context, roleID uint, permissionIDs []uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&models.RolePermission{}).Error; err != nil {
			return err
		}
		for _, pid := range permissionIDs {
			rp := models.RolePermission{
				RoleID:       roleID,
				PermissionID: pid,
			}
			if err := tx.Create(&rp).Error; err != nil {
				return err
			}
		}

		// 失效关联用户的缓存
		var userIDs []uint
		if err := tx.Table("user_roles").
			Select("user_id").
			Where("role_id = ?", roleID).
			Find(&userIDs).Error; err == nil {
			for _, uid := range userIDs {
				s.invalidateUserCache(uid)
			}
		}
		return nil
	})
}

// UpdateUserRoles 更新用户角色列表
func (s *RBACService) UpdateUserRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		for _, rid := range roleIDs {
			rel := models.UserRole{
				UserID: userID,
				RoleID: rid,
			}
			if err := tx.Create(&rel).Error; err != nil {
				return err
			}
		}
		s.invalidateUserCache(userID)
		return nil
	})
}

// EnsureUserHasRoleByTag 确保用户拥有指定角色（用于新用户默认授权）
func (s *RBACService) EnsureUserHasRoleByTag(ctx context.Context, userID uint, roleTag string) error {
	var role models.Role
	if err := s.db.WithContext(ctx).Where("role_tag = ?", roleTag).First(&role).Error; err != nil {
		return err
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rel models.UserRole
		err := tx.Where("user_id = ? AND role_id = ?", userID, role.ID).First(&rel).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rel = models.UserRole{
				UserID: userID,
				RoleID: role.ID,
			}
			if err := tx.Create(&rel).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
		s.invalidateUserCache(userID)
		return nil
	})
}

// GetUserPermissionSnapshot 获取用户有效权限（含缓存）
func (s *RBACService) GetUserPermissionSnapshot(ctx context.Context, userID uint) (*UserPermissionSnapshot, error) {
	if s.cache != nil {
		if cached, err := s.cache.Get(ctx, s.cacheKey(userID)); err == nil && cached != "" {
			var snap UserPermissionSnapshot
			if err := json.Unmarshal([]byte(cached), &snap); err == nil {
				return &snap, nil
			}
		}
	}

	var roleTags []string
	if err := s.db.WithContext(ctx).
		Table("roles").
		Select("roles.role_tag").
		Joins("JOIN user_roles ur ON ur.role_id = roles.id").
		Where("ur.user_id = ?", userID).
		Find(&roleTags).Error; err != nil {
		return nil, err
	}

	isAdmin := false
	for _, tag := range roleTags {
		if tag == models.RoleTagAdmin {
			isAdmin = true
			break
		}
	}

	var permissionTags []string
	if err := s.db.WithContext(ctx).
		Table("permissions").
		Select("DISTINCT permissions.permission_tag").
		Joins("JOIN role_permissions rp ON rp.permission_id = permissions.id").
		Joins("JOIN user_roles ur ON ur.role_id = rp.role_id").
		Where("ur.user_id = ?", userID).
		Find(&permissionTags).Error; err != nil {
		return nil, err
	}

	snap := &UserPermissionSnapshot{
		RoleTags:       roleTags,
		PermissionTags: permissionTags,
		IsAdmin:        isAdmin,
		CachedAt:       time.Now(),
	}

	if s.cache != nil {
		if data, err := json.Marshal(snap); err == nil {
			_ = s.cache.Set(ctx, s.cacheKey(userID), string(data), nil)
		}
	}

	return snap, nil
}

// CheckPermission 判断用户是否拥有指定权限
func (s *RBACService) CheckPermission(ctx context.Context, userID uint, permissionTag string) (bool, error) {
	snap, err := s.GetUserPermissionSnapshot(ctx, userID)
	if err != nil {
		return false, err
	}
	if snap.IsAdmin {
		return true, nil
	}
	for _, tag := range snap.PermissionTags {
		if tag == permissionTag {
			return true, nil
		}
	}
	return false, nil
}

// GetUserRoleTags 获取用户角色标签
func (s *RBACService) GetUserRoleTags(ctx context.Context, userID uint) ([]string, error) {
	snap, err := s.GetUserPermissionSnapshot(ctx, userID)
	if err != nil {
		return nil, err
	}
	return snap.RoleTags, nil
}

// CheckUserRole 检查用户是否拥有指定角色
func (s *RBACService) CheckUserRole(ctx context.Context, userID uint, role string) bool {
	userRoleTags, err := s.GetUserRoleTags(ctx, userID)
	if err != nil {
		return false
	}
	return slices.Contains(userRoleTags, role)
}

// GetUserPermissions 返回用户权限标识列表
func (s *RBACService) GetUserPermissions(ctx context.Context, userID uint) ([]string, error) {
	snap, err := s.GetUserPermissionSnapshot(ctx, userID)
	if err != nil {
		return nil, err
	}
	return snap.PermissionTags, nil
}

// GetUsersByRoleTags 根据角色标签列表获取用户数组
func (s *RBACService) GetUsersByRoleTags(ctx context.Context, roleTags []string) ([]models.User, error) {
	if len(roleTags) == 0 {
		return []models.User{}, nil
	}

	var users []models.User
	err := s.db.WithContext(ctx).
		Table("users").
		Select("DISTINCT users.*").
		Joins("JOIN user_roles ur ON ur.user_id = users.id").
		Joins("JOIN roles r ON r.id = ur.role_id").
		Where("r.role_tag IN ?", roleTags).
		Find(&users).Error

	if err != nil {
		return nil, err
	}

	return users, nil
}

// GrantRole 授予用户角色（如果不存在）
func (s *RBACService) GrantRole(ctx context.Context, userID uint, roleID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var rel models.UserRole
		err := tx.Where("user_id = ? AND role_id = ?", userID, roleID).First(&rel).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rel = models.UserRole{
				UserID: userID,
				RoleID: roleID,
			}
			if err := tx.Create(&rel).Error; err != nil {
				return err
			}
			s.invalidateUserCache(userID)
		} else if err != nil {
			return err
		}
		return nil
	})
}

// RevokeRole 撤销用户角色
func (s *RBACService) RevokeRole(ctx context.Context, userID uint, roleID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}
		s.invalidateUserCache(userID)
		return nil
	})
}

// invalidateUserCache 清理用户权限缓存
func (s *RBACService) invalidateUserCache(userID uint) {
	if s.cache == nil {
		return
	}
	ctx := context.Background()
	if err := s.cache.Delete(ctx, s.cacheKey(userID)); err != nil {
		logger.ErrorCtx(ctx, map[string]any{
			"action":         "invalidate_user_cache",
			"message":        "清理用户权限缓存失败",
			"error":          err.Error(),
			"target_user_id": userID,
		})
	}
}
