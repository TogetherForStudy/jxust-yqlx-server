package services

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"strings"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *CourseTableService) ListAdminCourseTables(ctx context.Context, req *request.AdminListCourseTablesRequest) ([]response.AdminCourseTableResponse, int64, error) {
	var courseTables []models.CourseTable
	var total int64

	query := s.db.WithContext(ctx).Model(&models.CourseTable{})
	if req.ClassID != "" {
		query = query.Where("class_id = ?", strings.TrimSpace(req.ClassID))
	}
	if req.Semester != "" {
		query = query.Where("semester = ?", strings.TrimSpace(req.Semester))
	}
	if req.Keyword != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		query = query.Where("class_id LIKE ? OR semester LIKE ?", keyword, keyword)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询课表总数失败: %w", err))
	}

	pagination := utils.GetPagination(req.Page, req.PageSize)
	if err := query.Order("updated_at DESC, id DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&courseTables).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询课表列表失败: %w", err))
	}

	items := make([]response.AdminCourseTableResponse, 0, len(courseTables))
	for _, item := range courseTables {
		items = append(items, toAdminCourseTableResponse(item))
	}
	return items, total, nil
}

func (s *CourseTableService) GetAdminCourseTableByID(ctx context.Context, id uint) (*response.AdminCourseTableResponse, error) {
	var courseTable models.CourseTable
	if err := s.db.WithContext(ctx).First(&courseTable, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询课表详情失败: %w", err))
	}

	resp := toAdminCourseTableResponse(courseTable)
	return &resp, nil
}

func (s *CourseTableService) CreateAdminCourseTable(ctx context.Context, req *request.AdminCreateCourseTableRequest) (*response.AdminCourseTableResponse, error) {
	classID := strings.TrimSpace(req.ClassID)
	semester := strings.TrimSpace(req.Semester)
	if classID == "" || semester == "" {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "班级和学期不能为空"
		return nil, err
	}

	courseData, err := normalizeCourseTableJSON(req.CourseData)
	if err != nil {
		return nil, err
	}
	if err := s.ensureCourseTableUnique(ctx, 0, classID, semester); err != nil {
		return nil, err
	}

	courseTable := models.CourseTable{
		ClassID:    classID,
		Semester:   semester,
		CourseData: courseData,
	}
	if err := s.db.WithContext(ctx).Create(&courseTable).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建课表失败: %w", err))
	}

	resp := toAdminCourseTableResponse(courseTable)
	return &resp, nil
}

func (s *CourseTableService) UpdateAdminCourseTable(ctx context.Context, id uint, req *request.AdminUpdateCourseTableRequest) (*response.AdminCourseTableResponse, error) {
	var courseTable models.CourseTable
	if err := s.db.WithContext(ctx).First(&courseTable, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询课表失败: %w", err))
	}

	classID := courseTable.ClassID
	if req.ClassID != nil {
		classID = strings.TrimSpace(*req.ClassID)
	}
	semester := courseTable.Semester
	if req.Semester != nil {
		semester = strings.TrimSpace(*req.Semester)
	}
	if classID == "" || semester == "" {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "班级和学期不能为空"
		return nil, err
	}

	updates := map[string]any{}
	if req.ClassID != nil {
		updates["class_id"] = classID
	}
	if req.Semester != nil {
		updates["semester"] = semester
	}
	if req.CourseData != nil {
		courseData, err := normalizeCourseTableJSON(req.CourseData)
		if err != nil {
			return nil, err
		}
		updates["course_data"] = courseData
	}
	if len(updates) == 0 {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "至少提供一个需要更新的字段"
		return nil, err
	}

	if err := s.ensureCourseTableUnique(ctx, id, classID, semester); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Model(&courseTable).Updates(updates).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新课表失败: %w", err))
	}
	return s.GetAdminCourseTableByID(ctx, id)
}

func (s *CourseTableService) DeleteAdminCourseTable(ctx context.Context, id uint) error {
	result := s.db.WithContext(ctx).Delete(&models.CourseTable{}, id)
	if result.Error != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("删除课表失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return apperr.New(constant.CommonNotFound)
	}
	return nil
}

func (s *CourseTableService) ResetUserBindCount(ctx context.Context, targetUserID uint) error {
	var user models.User
	if err := s.db.WithContext(ctx).Select("id").First(&user, targetUserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.New(constant.CommonUserNotFound)
		}
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询用户失败: %w", err))
	}

	if err := s.db.WithContext(ctx).Model(&models.BindRecord{}).
		Where("user_id = ?", targetUserID).
		Update("bind_count", 0).Error; err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("重置绑定次数失败: %w", err))
	}
	return nil
}

func (s *CourseTableService) ensureCourseTableUnique(ctx context.Context, id uint, classID, semester string) error {
	var count int64
	query := s.db.WithContext(ctx).Model(&models.CourseTable{}).
		Where("class_id = ? AND semester = ?", classID, semester)
	if id > 0 {
		query = query.Where("id <> ?", id)
	}
	if err := query.Count(&count).Error; err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("检查课表唯一性失败: %w", err))
	}
	if count > 0 {
		err := apperr.New(constant.CommonConflict)
		err.Message = "该班级在当前学期的课表已存在"
		return err
	}
	return nil
}

func normalizeCourseTableJSON(raw stdjson.RawMessage) (datatypes.JSON, error) {
	if len(raw) == 0 || !stdjson.Valid(raw) {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "course_data 必须是有效 JSON"
		return nil, err
	}
	return datatypes.JSON(raw), nil
}

func toAdminCourseTableResponse(courseTable models.CourseTable) response.AdminCourseTableResponse {
	return response.AdminCourseTableResponse{
		ID:         courseTable.ID,
		ClassID:    courseTable.ClassID,
		Semester:   courseTable.Semester,
		CourseData: courseTable.CourseData,
		CreatedAt:  courseTable.CreatedAt,
		UpdatedAt:  courseTable.UpdatedAt,
	}
}
