package services

import (
	"context"
	"fmt"
	"strings"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"gorm.io/gorm"
)

type OrganizationService struct {
	db *gorm.DB
}

func NewOrganizationService(db *gorm.DB) *OrganizationService {
	return &OrganizationService{db: db}
}

func (s *OrganizationService) ListOrganizations(ctx context.Context, req *request.ListOrganizationsRequest) ([]models.Organization, int64, error) {
	var items []models.Organization
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Organization{})
	if keyword := strings.TrimSpace(req.Query); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"name LIKE ? OR organization_type LIKE ? OR affiliation LIKE ? OR campus LIKE ? OR introduction LIKE ? OR contact LIKE ?",
			like, like, like, like, like, like,
		)
	}
	if organizationType := strings.TrimSpace(req.OrganizationType); organizationType != "" {
		query = query.Where("organization_type = ?", organizationType)
	}
	if affiliation := strings.TrimSpace(req.Affiliation); affiliation != "" {
		query = query.Where("affiliation = ?", affiliation)
	}
	if campus := strings.TrimSpace(req.Campus); campus != "" {
		query = query.Where("campus = ?", campus)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询组织列表失败: %w", err))
	}

	pagination := utils.GetPagination(req.Page, req.PageSize)
	if err := query.Order("created_at DESC, id DESC").Offset(pagination.Offset).Limit(pagination.Size).Find(&items).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询组织列表失败: %w", err))
	}

	return items, total, nil
}

func (s *OrganizationService) GetOrganizationByID(ctx context.Context, id uint) (*models.Organization, error) {
	var item models.Organization
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.OrganizationNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询组织详情失败: %w", err))
	}
	return &item, nil
}

func (s *OrganizationService) CreateOrganization(ctx context.Context, req *request.CreateOrganizationRequest) (*models.Organization, error) {
	item := models.Organization{
		Name:             strings.TrimSpace(req.Name),
		OrganizationType: strings.TrimSpace(req.OrganizationType),
		Affiliation:      strings.TrimSpace(req.Affiliation),
		Campus:           strings.TrimSpace(req.Campus),
		Introduction:     strings.TrimSpace(req.Introduction),
		Contact:          strings.TrimSpace(req.Contact),
	}
	if err := validateOrganizationForCreate(&item); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Create(&item).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建组织失败: %w", err))
	}
	return &item, nil
}

func (s *OrganizationService) UpdateOrganization(ctx context.Context, id uint, req *request.UpdateOrganizationRequest) (*models.Organization, error) {
	var item models.Organization
	if err := s.db.WithContext(ctx).First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.OrganizationNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询组织失败: %w", err))
	}

	updates := map[string]any{}
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" {
			return nil, newOrganizationBadRequest("组织名称不能为空")
		}
		updates["name"] = value
	}
	if req.OrganizationType != nil {
		value := strings.TrimSpace(*req.OrganizationType)
		if value == "" {
			return nil, newOrganizationBadRequest("组织类型不能为空")
		}
		updates["organization_type"] = value
	}
	if req.Affiliation != nil {
		value := strings.TrimSpace(*req.Affiliation)
		if value == "" {
			return nil, newOrganizationBadRequest("组织所属不能为空")
		}
		updates["affiliation"] = value
	}
	if req.Campus != nil {
		value := strings.TrimSpace(*req.Campus)
		if value == "" {
			return nil, newOrganizationBadRequest("组织校区不能为空")
		}
		updates["campus"] = value
	}
	if req.Introduction != nil {
		value := strings.TrimSpace(*req.Introduction)
		if value == "" {
			return nil, newOrganizationBadRequest("组织介绍不能为空")
		}
		updates["introduction"] = value
	}
	if req.Contact != nil {
		value := strings.TrimSpace(*req.Contact)
		if value == "" {
			return nil, newOrganizationBadRequest("联系方式不能为空")
		}
		updates["contact"] = value
	}
	if len(updates) == 0 {
		return nil, newOrganizationBadRequest("至少提供一个需要更新的字段")
	}

	if err := s.db.WithContext(ctx).Model(&item).Updates(updates).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新组织失败: %w", err))
	}
	return s.GetOrganizationByID(ctx, id)
}

func (s *OrganizationService) DeleteOrganization(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&models.Organization{}, id)
	if result.Error != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("删除组织失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return apperr.New(constant.OrganizationNotFound)
	}
	return nil
}

func validateOrganizationForCreate(item *models.Organization) error {
	if item.Name == "" {
		return newOrganizationBadRequest("组织名称不能为空")
	}
	if item.OrganizationType == "" {
		return newOrganizationBadRequest("组织类型不能为空")
	}
	if item.Affiliation == "" {
		return newOrganizationBadRequest("组织所属不能为空")
	}
	if item.Campus == "" {
		return newOrganizationBadRequest("组织校区不能为空")
	}
	if item.Introduction == "" {
		return newOrganizationBadRequest("组织介绍不能为空")
	}
	if item.Contact == "" {
		return newOrganizationBadRequest("联系方式不能为空")
	}
	return nil
}

func newOrganizationBadRequest(message string) error {
	err := apperr.New(constant.CommonBadRequest)
	err.Message = message
	return err
}
