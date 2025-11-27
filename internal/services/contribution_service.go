package services

import (
	"context"
	"errors"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"

	json "github.com/bytedance/sonic"
	"gorm.io/gorm"
)

type ContributionService struct {
	db                  *gorm.DB
	pointsService       *PointsService
	notificationService *NotificationService
}

func NewContributionService(db *gorm.DB) *ContributionService {
	return &ContributionService{
		db:                  db,
		pointsService:       NewPointsService(db),
		notificationService: NewNotificationService(db),
	}
}

// CreateContribution 创建投稿
func (s *ContributionService) CreateContribution(ctx context.Context, userID uint, req *request.CreateContributionRequest) error {

	// 序列化分类
	categoriesJSON, err := json.Marshal(req.Categories)
	if err != nil {
		return err
	}

	// 创建投稿
	contribution := models.UserContribution{
		UserID:     userID,
		Title:      req.Title,
		Content:    req.Content,
		Categories: categoriesJSON,
		Status:     models.UserContributionStatusPending,
	}

	if err := s.db.WithContext(ctx).Create(&contribution).Error; err != nil {
		return err
	}

	return nil
}

// GetContributions 获取投稿列表
func (s *ContributionService) GetContributions(ctx context.Context, userID uint, userRole models.UserRole, req *request.GetContributionsRequest) (*response.PageResponse, error) {
	var contributions []models.UserContribution
	var total int64

	// 构建查询
	query := s.db.WithContext(ctx).Model(&models.UserContribution{})

	// 普通用户只能看自己的投稿
	if userRole == models.UserRoleNormal {
		query = query.Where("user_id = ?", userID)
	} else if req.UserID != nil {
		// 管理员可以按用户ID过滤
		query = query.Where("user_id = ?", *req.UserID)
	}

	// 状态过滤
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询并预加载关联关系
	offset := (req.Page - 1) * req.Size
	if err := query.Order("created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, nickname")
		}).
		Preload("Reviewer", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, nickname")
		}).
		Preload("Notification").
		Find(&contributions).Error; err != nil {
		return nil, err
	}
	// 转换为响应格式
	var contributionResponses []response.ContributionResponse
	for _, contribution := range contributions {
		contributionResponse, err := s.convertToResponse(ctx, &contribution)
		if err != nil {
			return nil, err
		}
		contributionResponses = append(contributionResponses, *contributionResponse)
	}

	return &response.PageResponse{
		Data:  contributionResponses,
		Total: total,
		Page:  req.Page,
		Size:  req.Size,
	}, nil
}

// GetContributionByID 根据ID获取投稿详情
func (s *ContributionService) GetContributionByID(ctx context.Context, contributionID uint, userID uint, userRole models.UserRole) (*response.ContributionResponse, error) {
	var contribution models.UserContribution
	if err := s.db.Preload("User", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, nickname")
	}).Preload("Reviewer", func(db *gorm.DB) *gorm.DB {
		return db.Select("id, nickname")
	}).Preload("Notification").WithContext(ctx).First(&contribution, contributionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("投稿不存在")
		}
		return nil, err
	}

	// 权限检查：普通用户只能查看自己的投稿
	if contribution.UserID != userID && userRole == models.UserRoleNormal {
		return nil, errors.New("无权限")
	}

	return s.convertToResponse(ctx, &contribution)
}

// ReviewContribution 审核投稿
func (s *ContributionService) ReviewContribution(ctx context.Context, contributionID uint, reviewerID uint, reviewerRole models.UserRole, req *request.ReviewContributionRequest) error {

	// 查找投稿
	var contribution models.UserContribution
	if err := s.db.WithContext(ctx).First(&contribution, contributionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("投稿不存在")
		}
		return err
	}

	// 检查状态
	if contribution.Status != models.UserContributionStatusPending {
		return errors.New("只能审核待审核状态的投稿")
	}

	// 开启事务
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()

		// 更新投稿状态
		updates := map[string]interface{}{
			"status":      models.UserContributionStatus(req.Status),
			"reviewer_id": reviewerID,
			"review_note": req.ReviewNote,
			"reviewed_at": &now,
		}

		if req.Status == 2 { // 采纳
			updates["points_awarded"] = req.Points

			// 创建通知
			notificationReq := &request.CreateNotificationRequest{
				Title:   req.Title,
				Content: req.Content,
			}
			if len(req.Categories) > 0 {
				notificationReq.Categories = req.Categories
			} else {
				// 使用原分类
				var originalCategories []int
				if err := json.Unmarshal(contribution.Categories, &originalCategories); err != nil {
					return err
				}
				notificationReq.Categories = originalCategories
			}

			// 创建通知（使用投稿类型）
			categoriesJSON, err := json.Marshal(notificationReq.Categories)
			if err != nil {
				return err
			}

			notification := models.Notification{
				Title:         notificationReq.Title,
				Content:       notificationReq.Content,
				PublisherID:   reviewerID,
				PublisherType: models.NotificationPublisherUser,
				ContributorID: &contribution.UserID,
				Categories:    categoriesJSON,
				Status:        models.NotificationStatusPending, // 新投稿转换的通知需要审核
				PublishedAt:   nil,
			}

			if err := tx.Create(&notification).Error; err != nil {
				return err
			}

			updates["notification_id"] = notification.ID

			// 奖励积分
			if req.Points > 0 {
				if err := s.pointsService.AddPoints(ctx, tx, contribution.UserID, int(req.Points),
					models.PointsTransactionSourceContribution, "投稿被采纳", &contributionID); err != nil {
					return err
				}
			}
		}

		return tx.Model(&contribution).Updates(updates).Error
	})
}

// GetUserContributionStats 获取用户投稿统计
func (s *ContributionService) GetUserContributionStats(ctx context.Context, userID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总投稿数
	var totalCount int64
	if err := s.db.WithContext(ctx).Model(&models.UserContribution{}).Where("user_id = ?", userID).Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats["total_count"] = totalCount

	// 待审核数
	var pendingCount int64
	if err := s.db.WithContext(ctx).Model(&models.UserContribution{}).
		Where("user_id = ? AND status = ?", userID, models.UserContributionStatusPending).
		Count(&pendingCount).Error; err != nil {
		return nil, err
	}
	stats["pending_count"] = pendingCount

	// 已采纳数
	var approvedCount int64
	if err := s.db.WithContext(ctx).Model(&models.UserContribution{}).
		Where("user_id = ? AND status = ?", userID, models.UserContributionStatusApproved).
		Count(&approvedCount).Error; err != nil {
		return nil, err
	}
	stats["approved_count"] = approvedCount

	// 已拒绝数
	var rejectedCount int64
	if err := s.db.WithContext(ctx).Model(&models.UserContribution{}).
		Where("user_id = ? AND status = ?", userID, models.UserContributionStatusRejected).
		Count(&rejectedCount).Error; err != nil {
		return nil, err
	}
	stats["rejected_count"] = rejectedCount

	// 总获得积分
	var totalPoints int64
	if err := s.db.WithContext(ctx).Model(&models.UserContribution{}).
		Where("user_id = ? AND status = ?", userID, models.UserContributionStatusApproved).
		Select("COALESCE(SUM(points_awarded), 0)").
		Scan(&totalPoints).Error; err != nil {
		return nil, err
	}
	stats["total_points"] = totalPoints

	return stats, nil
}

// GetAdminContributionStats 获取管理员投稿统计（全系统）
func (s *ContributionService) GetAdminContributionStats(ctx context.Context) (*response.AdminContributionStatsResponse, error) {
	stats := &response.AdminContributionStatsResponse{}

	// 总投稿数
	if err := s.db.WithContext(ctx).Model(&models.UserContribution{}).Count(&stats.TotalCount).Error; err != nil {
		return nil, err
	}

	// 待审核数
	if err := s.db.WithContext(ctx).Model(&models.UserContribution{}).
		Where("status = ?", models.UserContributionStatusPending).
		Count(&stats.PendingCount).Error; err != nil {
		return nil, err
	}

	// 已采纳数
	if err := s.db.WithContext(ctx).Model(&models.UserContribution{}).
		Where("status = ?", models.UserContributionStatusApproved).
		Count(&stats.ApprovedCount).Error; err != nil {
		return nil, err
	}

	// 已拒绝数
	if err := s.db.WithContext(ctx).Model(&models.UserContribution{}).
		Where("status = ?", models.UserContributionStatusRejected).
		Count(&stats.RejectedCount).Error; err != nil {
		return nil, err
	}

	// 总发放积分
	if err := s.db.WithContext(ctx).Model(&models.UserContribution{}).
		Where("status = ?", models.UserContributionStatusApproved).
		Select("COALESCE(SUM(points_awarded), 0)").
		Scan(&stats.TotalPoints).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// 辅助方法：转换为响应格式
func (s *ContributionService) convertToResponse(ctx context.Context, contribution *models.UserContribution) (*response.ContributionResponse, error) {
	// 解析分类
	var categoryIDs []uint
	if len(contribution.Categories) > 0 {
		if err := json.Unmarshal(contribution.Categories, &categoryIDs); err != nil {
			return nil, err
		}
	}

	categories, err := s.getCategoriesByIDs(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}

	// 使用预加载的用户信息
	var user *response.UserSimpleResponse
	if contribution.User != nil {
		user = &response.UserSimpleResponse{
			ID:       contribution.User.ID,
			Nickname: contribution.User.Nickname,
		}
	}

	// 使用预加载的审核者信息
	var reviewer *response.UserSimpleResponse
	if contribution.Reviewer != nil {
		reviewer = &response.UserSimpleResponse{
			ID:       contribution.Reviewer.ID,
			Nickname: contribution.Reviewer.Nickname,
		}
	}

	// 使用预加载的通知信息
	var notification *response.NotificationSimpleResponse
	if contribution.Notification != nil {
		// 解析日程数据
		var scheduleData *models.ScheduleData
		if contribution.Notification.Schedule != nil {
			var schedule models.ScheduleData
			if err := json.Unmarshal(contribution.Notification.Schedule, &schedule); err == nil {
				scheduleData = &schedule
			}
		}

		notification = &response.NotificationSimpleResponse{
			ID:          contribution.Notification.ID,
			Title:       contribution.Notification.Title,
			Categories:  categories,
			PublisherID: contribution.Notification.PublisherID,
			Schedule:    scheduleData,
			Status:      contribution.Notification.Status,
			ViewCount:   contribution.Notification.ViewCount,
			PublishedAt: contribution.Notification.PublishedAt,
			CreatedAt:   contribution.Notification.CreatedAt,
		}
	}

	return &response.ContributionResponse{
		ID:             contribution.ID,
		UserID:         contribution.UserID,
		User:           user,
		Title:          contribution.Title,
		Content:        contribution.Content,
		Categories:     categories,
		Status:         contribution.Status,
		ReviewerID:     contribution.ReviewerID,
		Reviewer:       reviewer,
		ReviewNote:     contribution.ReviewNote,
		NotificationID: contribution.NotificationID,
		Notification:   notification,
		PointsAwarded:  contribution.PointsAwarded,
		ReviewedAt:     contribution.ReviewedAt,
		CreatedAt:      contribution.CreatedAt,
		UpdatedAt:      contribution.UpdatedAt,
	}, nil
}

// 辅助方法：根据分类ID获取分类信息
func (s *ContributionService) getCategoriesByIDs(ctx context.Context, categoryIDs []uint) ([]response.NotificationCategoryResponse, error) {
	// 如果分类ID列表为空，直接返回空结果
	if len(categoryIDs) == 0 {
		return []response.NotificationCategoryResponse{}, nil
	}

	// 将uint8切片转换为interface{}切片，避免GORM将其当作二进制数据处理
	var interfaceIDs []interface{}
	for _, id := range categoryIDs {
		interfaceIDs = append(interfaceIDs, id)
	}

	var categories []models.NotificationCategory
	if err := s.db.WithContext(ctx).Where("id IN ?", interfaceIDs).Find(&categories).Error; err != nil {
		return nil, err
	}

	var responses []response.NotificationCategoryResponse
	for _, category := range categories {
		responses = append(responses, response.NotificationCategoryResponse{
			ID:        category.ID,
			Name:      category.Name,
			Sort:      category.Sort,
			IsActive:  category.IsActive,
			CreatedAt: category.CreatedAt,
			UpdatedAt: category.UpdatedAt,
		})
	}

	return responses, nil
}
