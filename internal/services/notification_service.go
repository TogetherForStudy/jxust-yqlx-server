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

type NotificationService struct {
	db *gorm.DB
}

func NewNotificationService(db *gorm.DB) *NotificationService {
	return &NotificationService{
		db: db,
	}
}

// CreateNotification 创建通知（管理员专用）
func (s *NotificationService) CreateNotification(ctx context.Context, userID uint, userRole models.UserRole, req *request.CreateNotificationRequest) (*response.NotificationResponse, error) {

	// 序列化分类
	categoriesJSON, err := json.Marshal(req.Categories)
	if err != nil {
		return nil, err
	}

	// 创建通知
	notification := models.Notification{
		Title:         req.Title,
		Content:       req.Content,
		PublisherID:   userID,
		PublisherType: models.NotificationPublisherOperator,
		Categories:    categoriesJSON,
		Status:        models.NotificationStatusDraft,
	}

	if err := s.db.WithContext(ctx).Create(&notification).Error; err != nil {
		return nil, err
	}

	return s.GetNotificationAdminByID(ctx, notification.ID)
}

// UpdateNotification 更新通知
func (s *NotificationService) UpdateNotification(ctx context.Context, notificationID uint, userID uint, userRole models.UserRole, req *request.UpdateNotificationRequest) (*response.NotificationResponse, error) {
	// 查找通知
	var notification models.Notification
	if err := s.db.WithContext(ctx).First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("通知不存在")
		}
		return nil, err
	}

	// 检查状态
	if notification.Status == models.NotificationStatusDeleted {
		return nil, errors.New("已删除的通知不能修改")
	}

	// 权限校验：管理员无限制，运营人员只能修改草稿状态且是自己创建的通知
	if userRole == models.UserRoleOperator {
		// 运营人员的限制
		if notification.Status != models.NotificationStatusDraft {
			return nil, errors.New("运营人员只能修改草稿状态的通知")
		}
		if notification.PublisherID != userID {
			return nil, errors.New("运营人员只能修改自己创建的通知")
		}
	}
	// 管理员（UserRoleAdmin）无限制，不需要额外检查

	// 更新字段
	updates := make(map[string]interface{})
	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Content != "" {
		updates["content"] = req.Content
	}
	if len(req.Categories) > 0 {
		// 序列化分类
		categoriesJSON, err := json.Marshal(req.Categories)
		if err != nil {
			return nil, err
		}
		updates["categories"] = categoriesJSON
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&notification).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetNotificationAdminByID(ctx, notificationID)
}

// PublishNotification 发布通知
func (s *NotificationService) PublishNotification(ctx context.Context, notificationID uint, userID uint, userRole models.UserRole) error {

	// 查找通知
	var notification models.Notification
	if err := s.db.WithContext(ctx).First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("通知不存在")
		}
		return err
	}

	// 检查状态
	if notification.Status != models.NotificationStatusDraft {
		return errors.New("只能发布草稿状态的通知")
	}

	// 更新状态
	now := time.Now()
	return s.db.WithContext(ctx).Model(&notification).Updates(map[string]interface{}{
		"status":       models.NotificationStatusPending,
		"publisher_id": userID,
		"published_at": &now,
	}).Error
}

// PublishNotificationAdmin 管理员直接发布通知（跳过审核流程）
func (s *NotificationService) PublishNotificationAdmin(ctx context.Context, notificationID uint, userID uint, userRole models.UserRole) error {

	// 查找通知
	var notification models.Notification
	if err := s.db.WithContext(ctx).First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("通知不存在")
		}
		return err
	}

	// 直接更新为已发布状态，跳过审核流程
	now := time.Now()
	return s.db.WithContext(ctx).Model(&notification).Updates(map[string]interface{}{
		"status":       models.NotificationStatusPublished,
		"publisher_id": userID,
		"published_at": &now,
	}).Error
}

// GetNotifications 获取通知列表
func (s *NotificationService) GetNotifications(ctx context.Context, req *request.GetNotificationsRequest) (*response.PageResponse, error) {
	var notifications []models.Notification
	var total int64

	// 构建查询
	query := s.db.WithContext(ctx).Model(&models.Notification{}).Where("status = ?", models.NotificationStatusPublished)

	// 分类过滤
	if len(req.Categories) > 0 {
		// 将uint8切片转换为JSON格式的字符串
		categoriesJSON, err := json.Marshal(req.Categories)
		if err != nil {
			return nil, err
		}
		query = query.Where("JSON_OVERLAPS(categories, ?)", string(categoriesJSON))
	}

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询，排序规则：置顶优先（按置顶时间倒序），然后非置顶按发布时间倒序
	offset := (req.Page - 1) * req.Size
	if err := query.Order("is_pinned DESC, pinned_at DESC, published_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&notifications).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	var notificationResponses []response.NotificationSimpleResponse
	for _, notification := range notifications {
		// 解析分类
		var categoryIDs []uint8
		if err := json.Unmarshal(notification.Categories, &categoryIDs); err != nil {
			continue
		}

		categories, _ := s.getCategoriesByIDs(ctx, categoryIDs)

		// 解析日程数据
		var scheduleData *models.ScheduleData
		if notification.Schedule != nil {
			var schedule models.ScheduleData
			if err := json.Unmarshal(notification.Schedule, &schedule); err == nil {
				scheduleData = &schedule
			}
		}

		notificationResponses = append(notificationResponses, response.NotificationSimpleResponse{
			ID:          notification.ID,
			Title:       notification.Title,
			Categories:  categories,
			Status:      notification.Status,
			Schedule:    scheduleData,
			ViewCount:   notification.ViewCount,
			IsPinned:    notification.IsPinned,
			PinnedAt:    notification.PinnedAt,
			PublishedAt: notification.PublishedAt,
			CreatedAt:   notification.CreatedAt,
		})
	}

	return &response.PageResponse{
		Data:  notificationResponses,
		Total: total,
		Page:  req.Page,
		Size:  req.Size,
	}, nil
}

// GetNotificationByID 根据ID获取通知详情
func (s *NotificationService) GetNotificationByID(ctx context.Context, notificationID uint) (*response.NotificationResponse, error) {
	var notification models.Notification
	if err := s.db.WithContext(ctx).First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("通知不存在")
		}
		return nil, err
	}
	if notification.Status != models.NotificationStatusPublished {
		return nil, errors.New("通知未发布")
	}

	// 如果是已发布的通知，增加查看次数
	if notification.Status == models.NotificationStatusPublished {
		s.db.Model(&notification).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))
	}

	// 解析分类
	var categoryIDs []uint8
	if err := json.Unmarshal(notification.Categories, &categoryIDs); err != nil {
		return nil, err
	}

	categories, err := s.getCategoriesByIDs(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}

	// 解析日程数据
	var scheduleData *models.ScheduleData
	if notification.Schedule != nil {
		var schedule models.ScheduleData
		if err := json.Unmarshal(notification.Schedule, &schedule); err == nil {
			scheduleData = &schedule
		}
	}

	var publisher *response.UserSimpleResponse
	var contributor *response.UserSimpleResponse

	return &response.NotificationResponse{
		ID:            notification.ID,
		Title:         notification.Title,
		Content:       notification.Content,
		PublisherID:   notification.PublisherID,
		PublisherType: notification.PublisherType,
		Publisher:     publisher,
		ContributorID: notification.ContributorID,
		Contributor:   contributor,
		Categories:    categories,
		Status:        notification.Status,
		Schedule:      scheduleData,
		ViewCount:     notification.ViewCount,
		IsPinned:      notification.IsPinned,
		PinnedAt:      notification.PinnedAt,
		PublishedAt:   notification.PublishedAt,
		CreatedAt:     notification.CreatedAt,
		UpdatedAt:     notification.UpdatedAt,
	}, nil
}

// GetNotificationByID 根据ID获取通知详情
func (s *NotificationService) GetNotificationAdminByID(ctx context.Context, notificationID uint) (*response.NotificationResponse, error) {
	var notification models.Notification
	if err := s.db.WithContext(ctx).First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("通知不存在")
		}
		return nil, err
	}

	// 如果是已发布的通知，增加查看次数
	if notification.Status == models.NotificationStatusPublished {
		s.db.WithContext(ctx).Model(&notification).UpdateColumn("view_count", gorm.Expr("view_count + ?", 1))
	}

	// 解析分类
	var categoryIDs []uint8
	if err := json.Unmarshal(notification.Categories, &categoryIDs); err != nil {
		return nil, err
	}

	categories, err := s.getCategoriesByIDs(ctx, categoryIDs)
	if err != nil {
		return nil, err
	}

	// 解析日程数据
	var scheduleData *models.ScheduleData
	if notification.Schedule != nil {
		var schedule models.ScheduleData
		if err := json.Unmarshal(notification.Schedule, &schedule); err == nil {
			scheduleData = &schedule
		}
	}

	var publisher *response.UserSimpleResponse
	var contributor *response.UserSimpleResponse
	var approvals []response.NotificationApprovalResponse
	var approvalSummary *response.NotificationApprovalSummary

	// 获取发布者信息
	var publisherUser models.User
	if err := s.db.WithContext(ctx).Select("id, nickname").First(&publisherUser, notification.PublisherID).Error; err == nil {
		publisher = &response.UserSimpleResponse{
			ID:       publisherUser.ID,
			Nickname: publisherUser.Nickname,
		}
	}

	// 获取投稿者信息
	if notification.ContributorID != nil {
		var contributorUser models.User
		if err := s.db.WithContext(ctx).Select("id, nickname").First(&contributorUser, *notification.ContributorID).Error; err == nil {
			contributor = &response.UserSimpleResponse{
				ID:       contributorUser.ID,
				Nickname: contributorUser.Nickname,
			}
		}
	}

	// 获取审核记录
	if notification.Status == models.NotificationStatusPending || notification.Status == models.NotificationStatusPublished {
		var approvalRecords []models.NotificationApproval
		if err := s.db.WithContext(ctx).Where("notification_id = ?", notification.ID).Find(&approvalRecords).Error; err == nil {
			for _, approval := range approvalRecords {
				var reviewer models.User
				if err := s.db.WithContext(ctx).Select("id, nickname").First(&reviewer, approval.ReviewerID).Error; err == nil {
					approvals = append(approvals, response.NotificationApprovalResponse{
						ID: approval.ID,
						Reviewer: response.UserSimpleResponse{
							ID:       reviewer.ID,
							Nickname: reviewer.Nickname,
						},
						Status:    approval.Status,
						Note:      approval.Note,
						CreatedAt: approval.CreatedAt,
					})
				}
			}
		}

		// 生成审核进度汇总
		if summary, err := s.generateApprovalSummary(ctx, notification.ID); err == nil {
			approvalSummary = summary
		}
	}

	return &response.NotificationResponse{
		ID:              notification.ID,
		Title:           notification.Title,
		Content:         notification.Content,
		PublisherID:     notification.PublisherID,
		PublisherType:   notification.PublisherType,
		Publisher:       publisher,
		ContributorID:   notification.ContributorID,
		Contributor:     contributor,
		Categories:      categories,
		Status:          notification.Status,
		Schedule:        scheduleData,
		ViewCount:       notification.ViewCount,
		IsPinned:        notification.IsPinned,
		PinnedAt:        notification.PinnedAt,
		PublishedAt:     notification.PublishedAt,
		CreatedAt:       notification.CreatedAt,
		UpdatedAt:       notification.UpdatedAt,
		Approvals:       approvals,
		ApprovalSummary: approvalSummary,
	}, nil
}

// ConvertToSchedule 转换通知为日程
func (s *NotificationService) ConvertToSchedule(ctx context.Context, notificationID uint, req *request.ConvertToScheduleRequest) error {

	// 查找通知
	var notification models.Notification
	if err := s.db.WithContext(ctx).First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("通知不存在")
		}
		return err
	}

	// 创建日程数据
	scheduleData := models.ScheduleData{
		Title:       req.Title,
		Description: req.Description,
		TimeSlots:   make([]models.ScheduleTimeSlot, len(req.TimeSlots)),
	}

	for i, timeSlot := range req.TimeSlots {
		scheduleData.TimeSlots[i] = models.ScheduleTimeSlot{
			Name:      timeSlot.Name,
			StartDate: timeSlot.StartDate,
			EndDate:   timeSlot.EndDate,
			StartTime: timeSlot.StartTime,
			EndTime:   timeSlot.EndTime,
			IsAllDay:  timeSlot.IsAllDay,
		}
	}

	// 序列化日程数据
	scheduleJSON, err := json.Marshal(scheduleData)
	if err != nil {
		return err
	}

	// 更新通知的日程字段
	return s.db.WithContext(ctx).Model(&notification).Update("schedule", scheduleJSON).Error
}

// CreateCategory 创建分类
func (s *NotificationService) CreateCategory(ctx context.Context, req *request.CreateCategoryRequest) (*response.NotificationCategoryResponse, error) {
	category := models.NotificationCategory{
		Name:     req.Name,
		Sort:     req.Sort,
		IsActive: req.IsActive,
	}

	if err := s.db.WithContext(ctx).Create(&category).Error; err != nil {
		return nil, err
	}

	return &response.NotificationCategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		Sort:      category.Sort,
		IsActive:  category.IsActive,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}, nil
}

// GetCategories 获取所有分类
func (s *NotificationService) GetCategories(ctx context.Context) ([]response.NotificationCategoryResponse, error) {
	var categories []models.NotificationCategory
	if err := s.db.WithContext(ctx).Where("is_active = ?", true).Order("sort ASC, id ASC").Find(&categories).Error; err != nil {
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

// UpdateCategory 更新分类
func (s *NotificationService) UpdateCategory(ctx context.Context, categoryID uint8, req *request.UpdateCategoryRequest) (*response.NotificationCategoryResponse, error) {
	var category models.NotificationCategory
	if err := s.db.WithContext(ctx).First(&category, categoryID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("分类不存在")
		}
		return nil, err
	}

	// 更新字段
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Sort != 0 {
		updates["sort"] = req.Sort
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) > 0 {
		if err := s.db.WithContext(ctx).Model(&category).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return &response.NotificationCategoryResponse{
		ID:        category.ID,
		Name:      category.Name,
		Sort:      category.Sort,
		IsActive:  category.IsActive,
		CreatedAt: category.CreatedAt,
		UpdatedAt: category.UpdatedAt,
	}, nil
}

// DeleteNotification 删除通知（软删除）
func (s *NotificationService) DeleteNotification(ctx context.Context, notificationID uint, userID uint, userRole models.UserRole) error {

	// 查找通知
	var notification models.Notification
	if err := s.db.WithContext(ctx).First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("通知不存在")
		}
		return err
	}

	// 软删除
	return s.db.Delete(&notification).Error
}

// ApproveNotification 审核通知
func (s *NotificationService) ApproveNotification(ctx context.Context, notificationID uint, reviewerID uint, userRole models.UserRole, req *request.ApproveNotificationRequest) error {
	// 查找通知
	var notification models.Notification
	if err := s.db.WithContext(ctx).First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("通知不存在")
		}
		return err
	}

	// 检查通知状态
	if notification.Status != models.NotificationStatusPending {
		return errors.New("只能审核待审核状态的通知")
	}

	// 检查是否已经审核过
	var existingApproval models.NotificationApproval
	if err := s.db.WithContext(ctx).Where("notification_id = ? AND reviewer_id = ?", notificationID, reviewerID).First(&existingApproval).Error; err == nil {
		return errors.New("您已经审核过该通知")
	}

	// 创建审核记录
	approval := models.NotificationApproval{
		NotificationID: notificationID,
		ReviewerID:     reviewerID,
		Status:         models.NotificationApprovalStatus(req.Status),
		Note:           req.Note,
	}

	// 开始事务
	tx := s.db.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 保存审核记录
	if err := tx.Create(&approval).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 如果是同意，检查是否达到发布条件
	if req.Status == request.NotificationApprovalStatusRequestApproved {
		// 获取管理员和运营人员总数
		totalAdminCount, err := s.getAdminAndOperatorCount(ctx)
		if err != nil {
			tx.Rollback()
			return err
		}

		// 获取已通过审核的人数
		approvedCount, err := s.getApprovedCount(ctx, notificationID)
		if err != nil {
			tx.Rollback()
			return err
		}
		approvedCount++

		// 检查是否达到50%通过率
		approvalRate := float64(approvedCount) / float64(totalAdminCount)
		if approvalRate >= 0.5 {
			// 达到50%通过率，正式发布
			now := time.Now()
			if err := tx.Model(&notification).Updates(map[string]interface{}{
				"status":       models.NotificationStatusPublished,
				"published_at": &now,
			}).Error; err != nil {
				tx.Rollback()
				return err
			}
		}
		// 未达到50%通过率，保持待审核状态，等待更多审核
	}

	// 拒绝操作只记录审核结果，不改变通知状态
	// 通知会继续保持待审核状态，等待其他管理员审核

	return tx.Commit().Error
}

// getAdminAndOperatorCount 获取管理员和运营人员总数
func (s *NotificationService) getAdminAndOperatorCount(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&models.User{}).Where("role IN ?", []models.UserRole{models.UserRoleAdmin, models.UserRoleOperator}).Count(&count).Error
	return count, err
}

// getApprovedCount 获取某通知已通过审核的人数
func (s *NotificationService) getApprovedCount(ctx context.Context, notificationID uint) (int64, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&models.NotificationApproval{}).Where("notification_id = ? AND status = ?", notificationID, models.NotificationApprovalStatusApproved).Count(&count).Error
	return count, err
}

// generateApprovalSummary 生成审核进度汇总
func (s *NotificationService) generateApprovalSummary(ctx context.Context, notificationID uint) (*response.NotificationApprovalSummary, error) {
	// 获取管理员和运营人员总数
	totalReviewers, err := s.getAdminAndOperatorCount(ctx)
	if err != nil {
		return nil, err
	}

	// 获取已通过审核的人数
	approvedCount, err := s.getApprovedCount(ctx, notificationID)
	if err != nil {
		return nil, err
	}

	// 获取已拒绝审核的人数
	var rejectedCount int64
	if err := s.db.WithContext(ctx).Model(&models.NotificationApproval{}).
		Where("notification_id = ? AND status = ?", notificationID, models.NotificationApprovalStatusRejected).
		Count(&rejectedCount).Error; err != nil {
		return nil, err
	}

	// 计算待审核人数
	pendingCount := totalReviewers - approvedCount - rejectedCount

	// 计算通过率
	var approvalRate float64
	if totalReviewers > 0 {
		approvalRate = float64(approvedCount) / float64(totalReviewers)
	}

	// 判断是否可以发布
	canPublish := approvalRate >= 0.5

	// 获取所有审核记录
	var approvals []models.NotificationApproval
	if err := s.db.WithContext(ctx).Where("notification_id = ?", notificationID).
		Find(&approvals).Error; err != nil {
		return nil, err
	}

	// 构建已通过和已拒绝的用户列表
	var approvedUsers []response.UserSimpleResponse
	var rejectedUsers []response.UserSimpleResponse
	reviewedUserIDs := make(map[uint]bool)

	for _, approval := range approvals {
		var user models.User
		if err := s.db.WithContext(ctx).First(&user, approval.ReviewerID).Error; err == nil {
			userInfo := response.UserSimpleResponse{
				ID:       user.ID,
				Nickname: user.Nickname,
			}

			if approval.Status == models.NotificationApprovalStatusApproved {
				approvedUsers = append(approvedUsers, userInfo)
			} else if approval.Status == models.NotificationApprovalStatusRejected {
				rejectedUsers = append(rejectedUsers, userInfo)
			}

			reviewedUserIDs[approval.ReviewerID] = true
		}
	}

	// 获取所有管理员和运营人员
	var allReviewers []models.User
	if err := s.db.WithContext(ctx).Where("role IN ?", []models.UserRole{models.UserRoleAdmin, models.UserRoleOperator}).
		Find(&allReviewers).Error; err != nil {
		return nil, err
	}

	// 构建未审核用户列表
	var pendingUsers []response.UserSimpleResponse
	for _, reviewer := range allReviewers {
		if !reviewedUserIDs[reviewer.ID] {
			pendingUsers = append(pendingUsers, response.UserSimpleResponse{
				ID:       reviewer.ID,
				Nickname: reviewer.Nickname,
			})
		}
	}

	return &response.NotificationApprovalSummary{
		TotalReviewers: totalReviewers,
		ApprovedCount:  approvedCount,
		RejectedCount:  rejectedCount,
		PendingCount:   pendingCount,
		ApprovalRate:   approvalRate,
		RequiredRate:   0.5,
		CanPublish:     canPublish,
		ApprovedUsers:  approvedUsers,
		RejectedUsers:  rejectedUsers,
		PendingUsers:   pendingUsers,
	}, nil
}

// GetAdminNotifications 获取管理员通知列表（包括所有状态的通知）
func (s *NotificationService) GetAdminNotifications(ctx context.Context, userRole models.UserRole, req *request.GetNotificationsRequest) (*response.PageResponse, error) {
	var notifications []models.Notification
	var total int64

	// 构建查询
	query := s.db.WithContext(ctx).Model(&models.Notification{})

	// 状态过滤
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	} else {
		// 如果没有指定状态，显示所有非删除状态的通知
		query = query.Where("status != ?", models.NotificationStatusDeleted)
	}

	// 分类过滤
	if len(req.Categories) > 0 {
		// 将uint8切片转换为JSON格式的字符串
		categoriesJSON, err := json.Marshal(req.Categories)
		if err != nil {
			return nil, err
		}
		query = query.Where("JSON_OVERLAPS(categories, ?)", string(categoriesJSON))
	}

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询，排序规则：置顶优先（按置顶时间倒序），然后非置顶按发布时间倒序，最后按更新时间倒序
	offset := (req.Page - 1) * req.Size
	if err := query.Order("is_pinned DESC, pinned_at DESC, published_at DESC, updated_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&notifications).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	var notificationResponses []response.NotificationSimpleResponse
	for _, notification := range notifications {
		// 解析分类
		var categoryIDs []uint8
		if err := json.Unmarshal(notification.Categories, &categoryIDs); err != nil {
			continue
		}

		categories, _ := s.getCategoriesByIDs(ctx, categoryIDs)

		// 解析日程数据
		var scheduleData *models.ScheduleData
		if notification.Schedule != nil {
			var schedule models.ScheduleData
			if err := json.Unmarshal(notification.Schedule, &schedule); err == nil {
				scheduleData = &schedule
			}
		}

		// 为待审核状态的通知生成审核进度汇总
		var approvalSummary *response.NotificationApprovalSummary
		if notification.Status == models.NotificationStatusPending {
			if summary, err := s.generateApprovalSummary(ctx, notification.ID); err == nil {
				approvalSummary = summary
			}
		}

		notificationResponses = append(notificationResponses, response.NotificationSimpleResponse{
			ID:              notification.ID,
			Title:           notification.Title,
			Categories:      categories,
			Status:          notification.Status,
			Schedule:        scheduleData,
			ViewCount:       notification.ViewCount,
			IsPinned:        notification.IsPinned,
			PinnedAt:        notification.PinnedAt,
			PublishedAt:     notification.PublishedAt,
			CreatedAt:       notification.CreatedAt,
			ApprovalSummary: approvalSummary,
		})
	}

	return &response.PageResponse{
		Data:  notificationResponses,
		Total: total,
		Page:  req.Page,
		Size:  req.Size,
	}, nil
}

// GetNotificationStats 获取通知统计信息
func (s *NotificationService) GetNotificationStats(ctx context.Context) (*response.NotificationStatsResponse, error) {
	stats := &response.NotificationStatsResponse{}

	// 统计总数量（排除软删除）
	var totalCount int64
	if err := s.db.WithContext(ctx).Model(&models.Notification{}).Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats.TotalCount = totalCount

	// 按状态统计
	var draftCount, pendingCount, publishedCount int64

	// 草稿数量
	if err := s.db.WithContext(ctx).Model(&models.Notification{}).
		Where("status = ?", models.NotificationStatusDraft).
		Count(&draftCount).Error; err != nil {
		return nil, err
	}
	stats.DraftCount = draftCount

	// 待审核数量
	if err := s.db.WithContext(ctx).Model(&models.Notification{}).
		Where("status = ?", models.NotificationStatusPending).
		Count(&pendingCount).Error; err != nil {
		return nil, err
	}
	stats.PendingCount = pendingCount

	// 已发布数量
	if err := s.db.WithContext(ctx).Model(&models.Notification{}).
		Where("status = ?", models.NotificationStatusPublished).
		Count(&publishedCount).Error; err != nil {
		return nil, err
	}
	stats.PublishedCount = publishedCount

	return stats, nil
}

// 辅助方法：根据分类ID获取分类信息
func (s *NotificationService) getCategoriesByIDs(ctx context.Context, categoryIDs []uint8) ([]response.NotificationCategoryResponse, error) {
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

// PinNotification 置顶通知（管理员专用）
func (s *NotificationService) PinNotification(ctx context.Context, notificationID uint, userRole models.UserRole) error {
	// 查找通知
	var notification models.Notification
	if err := s.db.WithContext(ctx).First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("通知不存在")
		}
		return err
	}

	// 检查状态，只有已发布的通知才能置顶
	if notification.Status != models.NotificationStatusPublished {
		return errors.New("只有已发布的通知才能置顶")
	}

	// 检查是否已经置顶
	if notification.IsPinned {
		return errors.New("通知已经置顶")
	}

	// 更新置顶状态
	now := time.Now()
	return s.db.WithContext(ctx).Model(&notification).Updates(map[string]interface{}{
		"is_pinned":  true,
		"pinned_at":  &now,
		"updated_at": now,
	}).Error
}

// UnpinNotification 取消置顶通知（管理员专用）
func (s *NotificationService) UnpinNotification(ctx context.Context, notificationID uint, userRole models.UserRole) error {
	// 查找通知
	var notification models.Notification
	if err := s.db.WithContext(ctx).First(&notification, notificationID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("通知不存在")
		}
		return err
	}

	// 检查是否已经置顶
	if !notification.IsPinned {
		return errors.New("通知未置顶")
	}

	// 更新置顶状态
	now := time.Now()
	return s.db.WithContext(ctx).Model(&notification).Updates(map[string]interface{}{
		"is_pinned":  false,
		"pinned_at":  nil,
		"updated_at": now,
	}).Error
}
