package services

import (
	"context"
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"

	"gorm.io/gorm"
)

type ReviewService struct {
	db            *gorm.DB
	pointsService *PointsService
}

func NewReviewService(db *gorm.DB, pointsService *PointsService) *ReviewService {
	return &ReviewService{
		db:            db,
		pointsService: pointsService,
	}
}

// CreateReview 创建教师评价
func (s *ReviewService) CreateReview(ctx context.Context, userID uint, req *request.CreateReviewRequest) error {
	// 检查是否已经评价过该教师的这门课程
	var existingReview models.TeacherReview
	err := s.db.WithContext(ctx).Where("user_id = ? AND teacher_name = ? AND course_name = ?", userID, req.TeacherName, req.CourseName).First(&existingReview).Error
	if err == nil {
		return fmt.Errorf("您已经评价过该教师的这门课程")
	} else if err != gorm.ErrRecordNotFound {
		return fmt.Errorf("检查评价记录失败: %w", err)
	}

	// 创建评价
	review := &models.TeacherReview{
		UserID:      userID,
		TeacherName: req.TeacherName,
		Campus:      req.Campus,
		CourseName:  req.CourseName,
		Content:     req.Content,
		Attitude:    req.Attitude,
		Status:      models.TeacherReviewStatusPending,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	return s.db.WithContext(ctx).Create(review).Error
}

// GetReviews 获取评价列表
func (s *ReviewService) GetReviews(ctx context.Context, page, size int, teacherName string, status models.TeacherReviewStatus) ([]models.TeacherReview, int64, error) {
	var reviews []models.TeacherReview
	var total int64

	query := s.db.WithContext(ctx).Model(&models.TeacherReview{})

	// 按教师名称筛选
	if teacherName != "" {
		query = query.Where("teacher_name LIKE ?", "%"+teacherName+"%")
	}

	// 按状态筛选
	if status > 0 {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	pagination := utils.GetPagination(page, size)
	err := query.Order("created_at DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&reviews).Error

	return reviews, total, err
}

// GetReviewsByTeacher 获取指定教师的评价
func (s *ReviewService) GetReviewsByTeacher(ctx context.Context, teacherName string, page, size int) ([]models.TeacherReview, int64, error) {
	var reviews []models.TeacherReview
	var total int64

	query := s.db.WithContext(ctx).Model(&models.TeacherReview{}).
		Where("teacher_name = ? AND status = ?", teacherName, models.TeacherReviewStatusApproved)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	pagination := utils.GetPagination(page, size)
	err := query.Order("created_at DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&reviews).Error

	return reviews, total, err
}

// GetUserReviews 获取用户的评价记录
func (s *ReviewService) GetUserReviews(ctx context.Context, userID uint, page, size int) ([]models.TeacherReview, int64, error) {
	var reviews []models.TeacherReview
	var total int64

	query := s.db.WithContext(ctx).Model(&models.TeacherReview{}).
		Where("user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	pagination := utils.GetPagination(page, size)
	err := query.Order("created_at DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&reviews).Error

	return reviews, total, err
}

// ApproveReview 审核通过评价
func (s *ReviewService) ApproveReview(ctx context.Context, reviewID uint, adminNote string) error {
	return s.db.WithContext(ctx).Model(&models.TeacherReview{}).
		Where("id = ?", reviewID).
		Updates(map[string]any{
			"status":     models.TeacherReviewStatusApproved,
			"admin_note": adminNote,
			"updated_at": time.Now(),
		}).Error
}

// RejectReview 审核拒绝评价
func (s *ReviewService) RejectReview(ctx context.Context, reviewID uint, adminNote string) error {
	return s.db.WithContext(ctx).Model(&models.TeacherReview{}).
		Where("id = ?", reviewID).
		Updates(map[string]any{
			"status":     models.TeacherReviewStatusRejected,
			"admin_note": adminNote,
			"updated_at": time.Now(),
		}).Error
}

// DeleteReview 删除评价
func (s *ReviewService) DeleteReview(ctx context.Context, reviewID uint) error {
	return s.db.WithContext(ctx).Delete(&models.TeacherReview{}, reviewID).Error
}

// GetReviewByID 根据ID获取评价
func (s *ReviewService) GetReviewByID(ctx context.Context, reviewID uint) (*models.TeacherReview, error) {
	var review models.TeacherReview
	err := s.db.WithContext(ctx).First(&review, reviewID).Error
	if err != nil {
		return nil, err
	}
	return &review, nil
}
