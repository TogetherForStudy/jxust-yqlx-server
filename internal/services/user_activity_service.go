package services

import (
	"context"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	"gorm.io/gorm"
)

type UserActivityService struct {
	db          *gorm.DB
	rbacService *RBACService
}

func NewUserActivityService(db *gorm.DB, rbacService *RBACService) *UserActivityService {
	return &UserActivityService{
		db:          db,
		rbacService: rbacService,
	}
}

// UpdateActiveUserRoles 更新活跃用户角色
// 规则：100天内有25天访问次数，授予活跃角色；不满足条件则取消活跃角色
func (s *UserActivityService) UpdateActiveUserRoles(ctx context.Context) error {
	logger.Infof("开始执行活跃用户角色更新任务...")

	// 计算100天前的日期
	now := time.Now()
	hundredDaysAgo := now.AddDate(0, 0, -100)

	// 查询所有在100天内有活动的用户
	var activeUsers []struct {
		UserID     uint
		ActiveDays int64
	}

	// 统计每个用户在100天内的活跃天数
	err := s.db.WithContext(ctx).
		Model(&models.UserActivity{}).
		Select("user_id, COUNT(DISTINCT date) as active_days").
		Where("date >= ?", hundredDaysAgo.Format("2006-01-02")).
		Group("user_id").
		Having("active_days >= ?", 25).
		Scan(&activeUsers).Error

	if err != nil {
		return err
	}

	logger.Infof("找到 %d 个满足活跃条件的用户", len(activeUsers))

	// 获取活跃角色ID
	var activeRole models.Role
	if err := s.db.WithContext(ctx).
		Where("role_tag = ?", models.RoleTagUserActive).
		First(&activeRole).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			logger.Warnf("活跃角色不存在，跳过角色授予")
			return nil
		}
		return err
	}

	// 获取所有当前拥有活跃角色的用户
	var currentActiveUsers []models.UserRole
	if err := s.db.WithContext(ctx).
		Where("role_id = ?", activeRole.ID).
		Find(&currentActiveUsers).Error; err != nil {
		return err
	}

	currentActiveUserMap := make(map[uint]bool)
	for _, ur := range currentActiveUsers {
		currentActiveUserMap[ur.UserID] = true
	}

	// 需要授予角色的用户（增量）
	usersToGrant := make(map[uint]bool)
	for _, user := range activeUsers {
		if !currentActiveUserMap[user.UserID] {
			usersToGrant[user.UserID] = true
		}
	}

	// 需要撤销角色的用户
	usersToRevoke := make(map[uint]bool)
	for userID := range currentActiveUserMap {
		found := false
		for _, user := range activeUsers {
			if user.UserID == userID {
				found = true
				break
			}
		}
		if !found {
			usersToRevoke[userID] = true
		}
	}

	// 授予活跃角色（增量）
	grantCount := 0
	for userID := range usersToGrant {
		if err := s.rbacService.GrantRole(ctx, userID, activeRole.ID); err != nil {
			logger.Warnf("授予用户 %d 活跃角色失败: %v", userID, err)
			continue
		}
		grantCount++
	}

	// 撤销活跃角色
	revokeCount := 0
	for userID := range usersToRevoke {
		if err := s.rbacService.RevokeRole(ctx, userID, activeRole.ID); err != nil {
			logger.Warnf("撤销用户 %d 活跃角色失败: %v", userID, err)
			continue
		}
		revokeCount++
	}

	logger.Infof("活跃用户角色更新完成: 授予 %d 个用户，撤销 %d 个用户", grantCount, revokeCount)
	return nil
}
