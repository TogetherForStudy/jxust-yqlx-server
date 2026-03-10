package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"gorm.io/gorm"
)

func (s *FailRateService) ListAdminFailRates(ctx context.Context, req *request.AdminListFailRatesRequest) ([]response.AdminFailRateResponse, int64, error) {
	var failRates []models.FailRate
	var total int64

	query := s.db.WithContext(ctx).Model(&models.FailRate{})
	if req.Keyword != "" {
		query = query.Where("course_name LIKE ?", "%"+strings.TrimSpace(req.Keyword)+"%")
	}
	if req.Department != "" {
		query = query.Where("department = ?", strings.TrimSpace(req.Department))
	}
	if req.Semester != "" {
		query = query.Where("semester = ?", strings.TrimSpace(req.Semester))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.FailRateQueryFailed, err)
	}

	pagination := utils.GetPagination(req.Page, req.PageSize)
	if err := query.Order("fail_rate DESC, id DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&failRates).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.FailRateQueryFailed, err)
	}

	items := make([]response.AdminFailRateResponse, 0, len(failRates))
	for _, item := range failRates {
		items = append(items, toAdminFailRateResponse(item))
	}
	return items, total, nil
}

func (s *FailRateService) GetAdminFailRateByID(ctx context.Context, id uint) (*response.AdminFailRateResponse, error) {
	var failRate models.FailRate
	if err := s.db.WithContext(ctx).First(&failRate, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonNotFound)
		}
		return nil, apperr.Wrap(constant.FailRateQueryFailed, err)
	}
	resp := toAdminFailRateResponse(failRate)
	return &resp, nil
}

func (s *FailRateService) CreateAdminFailRate(ctx context.Context, req *request.AdminCreateFailRateRequest) (*response.AdminFailRateResponse, error) {
	failRate := models.FailRate{
		CourseName: strings.TrimSpace(req.CourseName),
		Department: strings.TrimSpace(req.Department),
		Semester:   strings.TrimSpace(req.Semester),
		FailRate:   req.FailRate,
	}
	if err := validateFailRateModel(&failRate); err != nil {
		return nil, err
	}
	if err := s.ensureFailRateUnique(ctx, 0, failRate.CourseName, failRate.Department, failRate.Semester); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(&failRate).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建挂科率记录失败: %w", err))
	}
	resp := toAdminFailRateResponse(failRate)
	return &resp, nil
}

func (s *FailRateService) UpdateAdminFailRate(ctx context.Context, id uint, req *request.AdminUpdateFailRateRequest) (*response.AdminFailRateResponse, error) {
	var failRate models.FailRate
	if err := s.db.WithContext(ctx).First(&failRate, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonNotFound)
		}
		return nil, apperr.Wrap(constant.FailRateQueryFailed, err)
	}

	updated := failRate
	updates := map[string]any{}
	if req.CourseName != nil {
		updated.CourseName = strings.TrimSpace(*req.CourseName)
		updates["course_name"] = updated.CourseName
	}
	if req.Department != nil {
		updated.Department = strings.TrimSpace(*req.Department)
		updates["department"] = updated.Department
	}
	if req.Semester != nil {
		updated.Semester = strings.TrimSpace(*req.Semester)
		updates["semester"] = updated.Semester
	}
	if req.FailRate != nil {
		updated.FailRate = *req.FailRate
		updates["fail_rate"] = updated.FailRate
	}
	if len(updates) == 0 {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "至少提供一个需要更新的字段"
		return nil, err
	}

	if err := validateFailRateModel(&updated); err != nil {
		return nil, err
	}
	if err := s.ensureFailRateUnique(ctx, id, updated.CourseName, updated.Department, updated.Semester); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(&failRate).Updates(updates).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新挂科率记录失败: %w", err))
	}
	return s.GetAdminFailRateByID(ctx, id)
}

func (s *FailRateService) DeleteAdminFailRate(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&models.FailRate{}, id)
	if result.Error != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("删除挂科率记录失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return apperr.New(constant.CommonNotFound)
	}
	return nil
}

func (s *FailRateService) ensureFailRateUnique(ctx context.Context, id uint, courseName, department, semester string) error {
	var count int64
	query := s.db.WithContext(ctx).Model(&models.FailRate{}).
		Where("course_name = ? AND department = ? AND semester = ?", courseName, department, semester)
	if id > 0 {
		query = query.Where("id <> ?", id)
	}
	if err := query.Count(&count).Error; err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("检查挂科率唯一性失败: %w", err))
	}
	if count > 0 {
		err := apperr.New(constant.CommonConflict)
		err.Message = "该课程在当前开课单位和学期下的挂科率记录已存在"
		return err
	}
	return nil
}

func validateFailRateModel(item *models.FailRate) error {
	if item.CourseName == "" || item.Department == "" || item.Semester == "" {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "课程名、开课单位和学期不能为空"
		return err
	}
	if item.FailRate < 0 || item.FailRate > 100 {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "failrate 必须在 0 到 100 之间"
		return err
	}
	return nil
}

func toAdminFailRateResponse(item models.FailRate) response.AdminFailRateResponse {
	return response.AdminFailRateResponse{
		ID:         item.ID,
		CourseName: item.CourseName,
		Department: item.Department,
		Semester:   item.Semester,
		FailRate:   item.FailRate,
		CreatedAt:  item.CreatedAt,
		UpdatedAt:  item.UpdatedAt,
	}
}
