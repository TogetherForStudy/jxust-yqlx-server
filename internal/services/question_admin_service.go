package services

import (
	"context"
	stdjson "encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func (s *QuestionService) ListAdminQuestionProjects(ctx context.Context, req *request.AdminListQuestionProjectsRequest) ([]response.AdminQuestionProjectResponse, int64, error) {
	var projects []models.QuestionProject
	var total int64

	query := s.db.WithContext(ctx).Model(&models.QuestionProject{})
	if req.Keyword != "" {
		keyword := "%" + strings.TrimSpace(req.Keyword) + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询题目项目总数失败: %w", err))
	}

	pagination := utils.GetPagination(req.Page, req.PageSize)
	if err := query.Order("sort ASC, id DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&projects).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询题目项目列表失败: %w", err))
	}

	countMap, err := s.getQuestionCountMap(ctx, collectProjectIDs(projects))
	if err != nil {
		return nil, 0, err
	}

	items := make([]response.AdminQuestionProjectResponse, 0, len(projects))
	for _, item := range projects {
		items = append(items, toAdminQuestionProjectResponse(item, countMap[item.ID]))
	}
	return items, total, nil
}

func (s *QuestionService) GetAdminQuestionProjectByID(ctx context.Context, id uint) (*response.AdminQuestionProjectResponse, error) {
	project, err := s.GetProjectByID(ctx, id)
	if err != nil {
		return nil, err
	}

	countMap, err := s.getQuestionCountMap(ctx, []uint{id})
	if err != nil {
		return nil, err
	}
	resp := toAdminQuestionProjectResponse(*project, countMap[id])
	return &resp, nil
}

func (s *QuestionService) CreateAdminQuestionProject(ctx context.Context, req *request.AdminCreateQuestionProjectRequest) (*response.AdminQuestionProjectResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "项目名称不能为空"
		return nil, err
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}
	version := req.Version
	if version == 0 {
		version = 1
	}

	project := models.QuestionProject{
		Name:        name,
		Description: strings.TrimSpace(req.Description),
		Version:     version,
		Sort:        req.Sort,
		IsActive:    isActive,
	}
	if err := s.db.WithContext(ctx).Create(&project).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建题目项目失败: %w", err))
	}

	resp := toAdminQuestionProjectResponse(project, 0)
	return &resp, nil
}

func (s *QuestionService) UpdateAdminQuestionProject(ctx context.Context, id uint, req *request.AdminUpdateQuestionProjectRequest) (*response.AdminQuestionProjectResponse, error) {
	var project models.QuestionProject
	if err := s.db.WithContext(ctx).First(&project, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询题目项目失败: %w", err))
	}

	updates := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			err := apperr.New(constant.CommonBadRequest)
			err.Message = "项目名称不能为空"
			return nil, err
		}
		updates["name"] = name
	}
	if req.Description != nil {
		updates["description"] = strings.TrimSpace(*req.Description)
	}
	if req.Version != nil {
		updates["version"] = *req.Version
	}
	if req.Sort != nil {
		updates["sort"] = *req.Sort
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}
	if len(updates) == 0 {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "至少提供一个需要更新的字段"
		return nil, err
	}

	if err := s.db.WithContext(ctx).Model(&project).Updates(updates).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新题目项目失败: %w", err))
	}
	return s.GetAdminQuestionProjectByID(ctx, id)
}

func (s *QuestionService) DeleteAdminQuestionProject(ctx context.Context, id uint) error {
	var questionCount int64
	if err := s.db.WithContext(ctx).Model(&models.Question{}).
		Where("project_id = ?", id).
		Count(&questionCount).Error; err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("检查题目项目下题目失败: %w", err))
	}
	if questionCount > 0 {
		err := apperr.New(constant.CommonConflict)
		err.Message = "题目项目下仍有题目，无法删除"
		return err
	}

	result := s.db.WithContext(ctx).Delete(&models.QuestionProject{}, id)
	if result.Error != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("删除题目项目失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return apperr.New(constant.CommonNotFound)
	}
	return nil
}

func (s *QuestionService) ListAdminQuestions(ctx context.Context, req *request.AdminListQuestionsRequest) ([]response.AdminQuestionResponse, int64, error) {
	var questions []models.Question
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Question{})
	if req.ProjectID > 0 {
		query = query.Where("project_id = ?", req.ProjectID)
	}
	if req.Keyword != "" {
		query = query.Where("title LIKE ?", "%"+strings.TrimSpace(req.Keyword)+"%")
	}
	if req.IsActive != nil {
		query = query.Where("is_active = ?", *req.IsActive)
	}
	if req.ParentID != nil {
		if *req.ParentID == 0 {
			query = query.Where("parent_id IS NULL")
		} else {
			query = query.Where("parent_id = ?", *req.ParentID)
		}
	}
	if req.Type != nil {
		query = query.Where("type = ?", *req.Type)
	}
	if req.SortMin != nil {
		query = query.Where("sort >= ?", *req.SortMin)
	}
	if req.SortMax != nil {
		query = query.Where("sort <= ?", *req.SortMax)
	}

	createdFrom, createdTo, err := parseAdminQuestionTimeRange(req.CreatedFrom, req.CreatedTo)
	if err != nil {
		return nil, 0, err
	}
	if createdFrom != nil {
		query = query.Where("created_at >= ?", *createdFrom)
	}
	if createdTo != nil {
		query = query.Where("created_at <= ?", *createdTo)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询题目总数失败: %w", err))
	}

	pagination := utils.GetPagination(req.Page, req.PageSize)
	if err := query.Order("sort ASC, id DESC").
		Offset(pagination.Offset).
		Limit(pagination.Size).
		Find(&questions).Error; err != nil {
		return nil, 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询题目列表失败: %w", err))
	}

	items := make([]response.AdminQuestionResponse, 0, len(questions))
	for _, item := range questions {
		items = append(items, toAdminQuestionResponse(item, false))
	}
	return items, total, nil
}

func (s *QuestionService) GetAdminQuestionByID(ctx context.Context, id uint) (*response.AdminQuestionResponse, error) {
	question, err := s.getAdminQuestion(ctx, id, true)
	if err != nil {
		return nil, err
	}
	resp := toAdminQuestionResponse(*question, true)
	return &resp, nil
}

func (s *QuestionService) CreateAdminQuestion(ctx context.Context, req *request.AdminCreateQuestionRequest) (*response.AdminQuestionResponse, error) {
	if err := s.ensureProjectExists(ctx, req.ProjectID); err != nil {
		return nil, err
	}

	parentID, err := s.normalizeAdminParentID(ctx, req.ProjectID, 0, req.ParentID)
	if err != nil {
		return nil, err
	}
	options, err := normalizeQuestionOptions(req.Type, req.Options, true)
	if err != nil {
		return nil, err
	}
	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	question := models.Question{
		ProjectID: req.ProjectID,
		ParentID:  parentID,
		Type:      models.QuestionType(req.Type),
		Title:     strings.TrimSpace(req.Title),
		Options:   options,
		Answer:    strings.TrimSpace(req.Answer),
		Sort:      req.Sort,
		IsActive:  isActive,
	}
	if err := validateAdminQuestionModel(&question); err != nil {
		return nil, err
	}

	if err := s.db.WithContext(ctx).Create(&question).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建题目失败: %w", err))
	}
	return s.GetAdminQuestionByID(ctx, question.ID)
}

func (s *QuestionService) UpdateAdminQuestion(ctx context.Context, id uint, req *request.AdminUpdateQuestionRequest) (*response.AdminQuestionResponse, error) {
	current, err := s.getAdminQuestion(ctx, id, false)
	if err != nil {
		return nil, err
	}

	childCount, err := s.getChildQuestionCount(ctx, id)
	if err != nil {
		return nil, err
	}

	nextQuestion := *current
	updates := map[string]any{}
	if req.ProjectID != nil {
		nextQuestion.ProjectID = *req.ProjectID
		updates["project_id"] = *req.ProjectID
	}
	if req.Type != nil {
		nextQuestion.Type = models.QuestionType(*req.Type)
		updates["type"] = *req.Type
	}
	if req.Title != nil {
		nextQuestion.Title = strings.TrimSpace(*req.Title)
		updates["title"] = nextQuestion.Title
	}
	if req.Answer != nil {
		nextQuestion.Answer = strings.TrimSpace(*req.Answer)
		updates["answer"] = nextQuestion.Answer
	}
	if req.Sort != nil {
		nextQuestion.Sort = *req.Sort
		updates["sort"] = *req.Sort
	}
	if req.IsActive != nil {
		nextQuestion.IsActive = *req.IsActive
		updates["is_active"] = *req.IsActive
	}
	if req.ProjectID != nil {
		if err := s.ensureProjectExists(ctx, nextQuestion.ProjectID); err != nil {
			return nil, err
		}
		if childCount > 0 && nextQuestion.ProjectID != current.ProjectID {
			err := apperr.New(constant.CommonConflict)
			err.Message = "存在子题的父题不能直接修改所属项目"
			return nil, err
		}
	}

	if req.ParentID != nil {
		nextParentID, err := s.normalizeAdminParentID(ctx, nextQuestion.ProjectID, id, req.ParentID)
		if err != nil {
			return nil, err
		}
		if childCount > 0 && nextParentID != nil {
			err := apperr.New(constant.CommonConflict)
			err.Message = "存在子题的父题不能再设置父题"
			return nil, err
		}
		nextQuestion.ParentID = nextParentID
		if nextParentID == nil {
			updates["parent_id"] = nil
		} else {
			updates["parent_id"] = *nextParentID
		}
	}

	if req.Options != nil || req.Type != nil {
		currentOptions, err := extractQuestionOptions(current.Options)
		if err != nil {
			return nil, err
		}
		nextOptionsInput := currentOptions
		if req.Options != nil {
			nextOptionsInput = *req.Options
		}
		options, err := normalizeQuestionOptions(int8(nextQuestion.Type), nextOptionsInput, true)
		if err != nil {
			return nil, err
		}
		nextQuestion.Options = options
		updates["options"] = options
	}

	if len(updates) == 0 {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "至少提供一个需要更新的字段"
		return nil, err
	}
	if err := validateAdminQuestionModel(&nextQuestion); err != nil {
		return nil, err
	}
	if err := s.db.WithContext(ctx).Model(current).Updates(updates).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新题目失败: %w", err))
	}
	return s.GetAdminQuestionByID(ctx, id)
}

func (s *QuestionService) DeleteAdminQuestion(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var question models.Question
		if err := tx.First(&question, id).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperr.New(constant.CommonNotFound)
			}
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询题目失败: %w", err))
		}

		if err := tx.Where("parent_id = ?", id).Delete(&models.Question{}).Error; err != nil {
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("删除子题失败: %w", err))
		}
		if err := tx.Delete(&question).Error; err != nil {
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("删除题目失败: %w", err))
		}
		return nil
	})
}

func (s *QuestionService) getAdminQuestion(ctx context.Context, id uint, withSubQuestions bool) (*models.Question, error) {
	var question models.Question
	query := s.db.WithContext(ctx).Model(&models.Question{})
	if withSubQuestions {
		query = query.Preload("SubQuestions", func(tx *gorm.DB) *gorm.DB {
			return tx.Order("sort ASC, id ASC")
		})
	}
	if err := query.First(&question, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询题目详情失败: %w", err))
	}
	return &question, nil
}

func (s *QuestionService) ensureProjectExists(ctx context.Context, projectID uint) error {
	if projectID == 0 {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "project_id 不能为空"
		return err
	}

	var count int64
	if err := s.db.WithContext(ctx).Model(&models.QuestionProject{}).
		Where("id = ?", projectID).
		Count(&count).Error; err != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询题目项目失败: %w", err))
	}
	if count == 0 {
		return apperr.New(constant.CommonNotFound)
	}
	return nil
}

func (s *QuestionService) normalizeAdminParentID(ctx context.Context, projectID, currentQuestionID uint, rawParentID *uint) (*uint, error) {
	if rawParentID == nil || *rawParentID == 0 {
		return nil, nil
	}
	if currentQuestionID > 0 && *rawParentID == currentQuestionID {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "题目不能设置自己为父题"
		return nil, err
	}

	var parent models.Question
	if err := s.db.WithContext(ctx).First(&parent, *rawParentID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询父题失败: %w", err))
	}
	if parent.ProjectID != projectID {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "父题必须与当前题目属于同一项目"
		return nil, err
	}
	if parent.ParentID != nil {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "父题不能是子题"
		return nil, err
	}

	parentID := *rawParentID
	return &parentID, nil
}

func (s *QuestionService) getQuestionCountMap(ctx context.Context, projectIDs []uint) (map[uint]int64, error) {
	countMap := make(map[uint]int64, len(projectIDs))
	if len(projectIDs) == 0 {
		return countMap, nil
	}

	var rows []struct {
		ProjectID uint
		Count     int64
	}
	if err := s.db.WithContext(ctx).Model(&models.Question{}).
		Select("project_id, COUNT(*) AS count").
		Where("project_id IN ?", projectIDs).
		Group("project_id").
		Scan(&rows).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询题目统计失败: %w", err))
	}
	for _, row := range rows {
		countMap[row.ProjectID] = row.Count
	}
	return countMap, nil
}

func (s *QuestionService) getChildQuestionCount(ctx context.Context, questionID uint) (int64, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&models.Question{}).
		Where("parent_id = ?", questionID).
		Count(&count).Error; err != nil {
		return 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询子题数量失败: %w", err))
	}
	return count, nil
}

func normalizeQuestionOptions(questionType int8, options []string, strict bool) (datatypes.JSON, error) {
	switch questionType {
	case constant.QuestionTypeChoice:
		trimmed := make([]string, 0, len(options))
		for _, item := range options {
			value := strings.TrimSpace(item)
			if value != "" {
				trimmed = append(trimmed, value)
			}
		}
		if strict && len(trimmed) == 0 {
			err := apperr.New(constant.CommonBadRequest)
			err.Message = "选择题必须提供至少一个选项"
			return nil, err
		}
		data, err := stdjson.Marshal(trimmed)
		if err != nil {
			return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("序列化题目选项失败: %w", err))
		}
		return datatypes.JSON(data), nil
	case constant.QuestionTypeEssay:
		return nil, nil
	default:
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "不支持的题目类型"
		return nil, err
	}
}

func extractQuestionOptions(raw datatypes.JSON) ([]string, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var options []string
	if err := stdjson.Unmarshal(raw, &options); err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("解析题目选项失败: %w", err))
	}
	return options, nil
}

func validateAdminQuestionModel(question *models.Question) error {
	if question.ProjectID == 0 {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "project_id 不能为空"
		return err
	}
	if strings.TrimSpace(question.Title) == "" {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "题目标题不能为空"
		return err
	}
	if strings.TrimSpace(question.Answer) == "" {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "题目答案不能为空"
		return err
	}
	if question.Type != models.QuestionTypeChoice && question.Type != models.QuestionTypeEssay {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "不支持的题目类型"
		return err
	}
	if question.Type == models.QuestionTypeChoice && len(question.Options) == 0 {
		err := apperr.New(constant.CommonBadRequest)
		err.Message = "选择题必须提供至少一个选项"
		return err
	}
	return nil
}

func parseAdminQuestionTimeRange(createdFrom, createdTo string) (*time.Time, *time.Time, error) {
	parseValue := func(label, value string, endOfDay bool) (*time.Time, error) {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, nil
		}

		if parsed, err := utils.ParseDateTime(value); err == nil && parsed != nil {
			return parsed, nil
		}

		loc := utils.GetChinaLocation()
		if parsed, err := time.ParseInLocation("2006-01-02", value, loc); err == nil {
			if endOfDay {
				end := parsed.Add(24*time.Hour - time.Nanosecond)
				return &end, nil
			}
			return &parsed, nil
		}
		err := apperr.New(constant.CommonBadRequest)
		err.Message = label + " 时间格式错误"
		return nil, err
	}

	from, err := parseValue("created_from", createdFrom, false)
	if err != nil {
		return nil, nil, err
	}
	to, err := parseValue("created_to", createdTo, true)
	if err != nil {
		return nil, nil, err
	}
	return from, to, nil
}

func collectProjectIDs(projects []models.QuestionProject) []uint {
	ids := make([]uint, 0, len(projects))
	for _, item := range projects {
		ids = append(ids, item.ID)
	}
	return ids
}

func toAdminQuestionProjectResponse(project models.QuestionProject, questionCount int64) response.AdminQuestionProjectResponse {
	return response.AdminQuestionProjectResponse{
		ID:            project.ID,
		Name:          project.Name,
		Description:   project.Description,
		Version:       project.Version,
		Sort:          project.Sort,
		IsActive:      project.IsActive,
		QuestionCount: questionCount,
		CreatedAt:     project.CreatedAt,
		UpdatedAt:     project.UpdatedAt,
	}
}

func toAdminQuestionResponse(question models.Question, includeSubQuestions bool) response.AdminQuestionResponse {
	resp := response.AdminQuestionResponse{
		ID:        question.ID,
		ProjectID: question.ProjectID,
		ParentID:  question.ParentID,
		Type:      int8(question.Type),
		Title:     question.Title,
		Answer:    question.Answer,
		Sort:      question.Sort,
		IsActive:  question.IsActive,
		CreatedAt: question.CreatedAt,
		UpdatedAt: question.UpdatedAt,
	}

	if options, err := extractQuestionOptions(question.Options); err == nil {
		resp.Options = options
	}

	if includeSubQuestions {
		for _, item := range question.SubQuestions {
			resp.SubQuestions = append(resp.SubQuestions, toAdminQuestionResponse(item, false))
		}
	}

	return resp
}
