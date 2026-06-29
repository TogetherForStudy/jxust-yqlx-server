package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/worker/processors"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"

	json "github.com/bytedance/sonic"
	"gorm.io/gorm"
)

type QuestionService struct {
	db             *gorm.DB
	featureService *FeatureService
}

func NewQuestionService(db *gorm.DB, featureService *FeatureService) *QuestionService {
	return &QuestionService{
		db:             db,
		featureService: featureService,
	}
}

// ===================== 项目查询 =====================

// GetProjectByID 根据ID获取项目
func (s *QuestionService) GetProjectByID(ctx context.Context, projectID uint) (*models.QuestionProject, error) {
	var project models.QuestionProject
	if err := s.db.WithContext(ctx).First(&project, projectID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, err)
	}
	return &project, nil
}

// GetProjects 获取项目列表
// 灰度规则：若 features 表中存在 review.project.{id} 且 is_enabled=true，则该项目灰度
// feature 不存在或 is_enabled=false → 公开可见
func (s *QuestionService) GetProjects(ctx context.Context, userID uint) ([]response.QuestionProjectResponse, error) {
	var projects []models.QuestionProject

	if err := s.db.WithContext(ctx).Where("is_active = ?", true).
		Order("sort ASC, created_at DESC").
		Find(&projects).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("获取项目列表失败: %w", err))
	}

	// 灰度预加载
	var enabledGraySet map[string]bool
	userFeatureSet := make(map[string]bool)

	if s.featureService != nil {
		enabledGraySet, _ = s.featureService.GetEnabledProjectFeatures(ctx)
		userFeatures, _ := s.featureService.GetUserFeatures(ctx, userID)
		for _, f := range userFeatures {
			userFeatureSet[f] = true
		}
	}

	// 过滤灰度项目 + 收集可见项目ID
	type visibleProject struct {
		project models.QuestionProject
	}
	var visible []visibleProject
	for _, project := range projects {
		featureKey := fmt.Sprintf("review.project.%d", project.ID)
		if enabledGraySet[featureKey] && !userFeatureSet[featureKey] {
			continue
		}
		visible = append(visible, visibleProject{project: project})
	}
	if len(visible) == 0 {
		return []response.QuestionProjectResponse{}, nil
	}

	// 收集可见项目 ID
	projectIDs := make([]uint, len(visible))
	for i, v := range visible {
		projectIDs[i] = v.project.ID
	}

	// 批量查询题目数量（一次 GROUP BY 替代 N 次 COUNT）
	type countRow struct {
		ProjectID uint
		Count     int64
	}
	var questionCounts []countRow
	s.db.WithContext(ctx).Model(&models.Question{}).
		Select("project_id, COUNT(*) as count").
		Where("project_id IN ? AND is_active = ? AND parent_id IS NULL", projectIDs, true).
		Group("project_id").Find(&questionCounts)
	qCountMap := make(map[uint]int64, len(questionCounts))
	for _, r := range questionCounts {
		qCountMap[r.ProjectID] = r.Count
	}

	// 构建结果
	var result []response.QuestionProjectResponse
	for _, v := range visible {
		project := v.project
		questionCount := qCountMap[project.ID]

		// 用户数 & 刷题次数（Redis 为主，每个项目 ≈0.2ms，总量可控）
		var userCount int64
		userSetKey := fmt.Sprintf("project:users:%d", project.ID)
		if cache.GlobalCache != nil {
			var err error
			userCount, err = cache.GlobalCache.SCard(ctx, userSetKey)
			if err != nil {
				s.db.WithContext(ctx).Model(&models.UserProjectUsage{}).
					Where("project_id = ?", project.ID).Count(&userCount)
			}
		} else {
			s.db.WithContext(ctx).Model(&models.UserProjectUsage{}).
				Where("project_id = ?", project.ID).Count(&userCount)
		}

		var usageCount int64
		usageKey := fmt.Sprintf("project:usage:%d", project.ID)
		if cache.GlobalCache != nil {
			var err error
			usageCount, err = cache.GlobalCache.GetInt(ctx, usageKey)
			if err != nil {
				s.db.WithContext(ctx).
					Table("user_question_usages uq").
					Joins("JOIN questions q ON q.id = uq.question_id").
					Where("q.project_id = ? AND q.is_active = ?", project.ID, true).
					Select("COALESCE(SUM(uq.study_count + uq.practice_count), 0)").
					Scan(&usageCount)
			}
		} else {
			s.db.WithContext(ctx).
				Table("user_question_usages uq").
				Joins("JOIN questions q ON q.id = uq.question_id").
				Where("q.project_id = ? AND q.is_active = ?", project.ID, true).
				Select("COALESCE(SUM(uq.study_count + uq.practice_count), 0)").
				Scan(&usageCount)
		}

		result = append(result, response.ToQuestionProjectResponse(&project, questionCount, userCount, usageCount))
	}

	return result, nil
}

// ===================== 获取题目 =====================

// GetQuestions 获取题目列表（只返回题目ID数组，支持顺序/乱序）
func (s *QuestionService) GetQuestions(ctx context.Context, userID uint, req *request.GetQuestionRequest) (*response.QuestionListResponse, error) {
	// 更新项目使用次数
	if err := s.updateProjectUsage(ctx, userID, req.ProjectID); err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新项目使用记录失败: %w", err))
	}
	s.markProjectOnline(ctx, req.ProjectID, userID)

	// 获取项目下所有启用的主题目/独立题（parent_id 为 null）的ID
	var questionIDs []uint
	query := s.db.WithContext(ctx).Model(&models.Question{}).
		Where("project_id = ? AND is_active = ? AND parent_id IS NULL", req.ProjectID, true).
		Select("id")

	if req.Random {
		// 乱序
		query = query.Order("RAND()")
	} else {
		// 顺序
		query = query.Order("sort ASC, id ASC")
	}

	if err := query.Pluck("id", &questionIDs).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("获取题目列表失败: %w", err))
	}

	return &response.QuestionListResponse{
		QuestionIDs: questionIDs,
	}, nil
}

// toQuestionResponseWithUsage 转换题目为usage map
func (s *QuestionService) toQuestionResponseWithUsage(question *models.Question, usageMap map[uint]*models.UserQuestionUsage) response.QuestionResponse {
	resp := response.QuestionResponse{
		ID:        question.ID,
		ProjectID: question.ProjectID,
		ParentID:  question.ParentID,
		Type:      int8(question.Type),
		Title:     question.Title,
		Answer:    question.Answer,
		Sort:      question.Sort,
	}

	// 解析选项
	if question.Type == models.QuestionTypeChoice && question.Options != nil {
		var options []string
		err := json.Unmarshal(question.Options, &options)
		if err != nil {
			return resp
		}
		resp.Options = options
	}

	// 填充用户使用统计
	if usage, ok := usageMap[question.ID]; ok {
		resp.StudyCount = usage.StudyCount
		resp.PracticeCount = usage.PracticeCount
	}

	// 转换子题
	if len(question.SubQuestions) > 0 {
		for _, subQ := range question.SubQuestions {
			resp.SubQuestions = append(resp.SubQuestions, s.toQuestionResponseWithUsage(&subQ, usageMap))
		}
	}

	return resp
}

// getQuestionByID 获取题目（内部使用）
func (s *QuestionService) getQuestionByID(ctx context.Context, questionID uint) (*models.Question, error) {
	var question models.Question
	if err := s.db.WithContext(ctx).Preload("SubQuestions", "is_active = ?", true).First(&question, questionID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.QuestionNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, err)
	}
	return &question, nil
}

// GetQuestionByID 获取题目详情（公开方法）
func (s *QuestionService) GetQuestionByID(ctx context.Context, userID, questionID uint) (*response.QuestionResponse, error) {
	// 获取题目
	question, err := s.getQuestionByID(ctx, questionID)
	if err != nil {
		return nil, err
	}

	// 检查题目是否启用
	if !question.IsActive {
		return nil, apperr.New(constant.QuestionDisabled)
	}
	s.markProjectOnline(ctx, question.ProjectID, userID)

	// 获取用户使用记录
	var usage models.UserQuestionUsage
	s.db.WithContext(ctx).Where("user_id = ? AND question_id = ?", userID, questionID).First(&usage)

	// 批量获取子题的使用记录
	var subQuestionIDs []uint
	for _, subQ := range question.SubQuestions {
		subQuestionIDs = append(subQuestionIDs, subQ.ID)
	}

	var usages []models.UserQuestionUsage
	if len(subQuestionIDs) > 0 {
		s.db.WithContext(ctx).Where("user_id = ? AND question_id IN ?", userID, subQuestionIDs).Find(&usages)
	}

	// 构建 usage map
	usageMap := make(map[uint]*models.UserQuestionUsage)
	if usage.ID > 0 {
		usageMap[questionID] = &usage
	}
	for i := range usages {
		usageMap[usages[i].QuestionID] = &usages[i]
	}

	// 转换为响应格式
	result := s.toQuestionResponseWithUsage(question, usageMap)
	return &result, nil
}

// ===================== 记录操作 =====================

// RecordStudy 记录学习（仅记录学习次数）
func (s *QuestionService) RecordStudy(ctx context.Context, userID uint, req *request.RecordStudyRequest) error {
	// 验证题目存在
	question, err := s.getQuestionByID(ctx, req.QuestionID)
	if err != nil {
		return err
	}

	now := time.Now()
	// 将任务推送到 Redis 队列
	task := processors.QuestionTask{
		Type:       constant.TaskTypeStudy,
		UserID:     userID,
		QuestionID: req.QuestionID,
		Time:       now,
	}
	taskData, _ := json.Marshal(task)

	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.LPush(ctx, "sync:question:usage", string(taskData))
	}

	// 更新 Redis 中的项目刷题次数
	usageKey := fmt.Sprintf("project:usage:%d", question.ProjectID)
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.Incr(ctx, usageKey)
	}

	s.markProjectOnline(ctx, question.ProjectID, userID)

	return nil
}

// SubmitPractice 提交做题（仅记录做题次数）
func (s *QuestionService) SubmitPractice(ctx context.Context, userID uint, req *request.SubmitPracticeRequest) error {
	// 验证题目存在
	question, err := s.getQuestionByID(ctx, req.QuestionID)
	if err != nil {
		return err
	}

	now := time.Now()
	// 将任务推送到 Redis 队列
	task := processors.QuestionTask{
		Type:       constant.TaskTypePractice,
		UserID:     userID,
		QuestionID: req.QuestionID,
		Time:       now,
	}
	taskData, _ := json.Marshal(task)

	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.LPush(ctx, "sync:question:usage", string(taskData))
	}

	// 更新 Redis 中的项目刷题次数
	usageKey := fmt.Sprintf("project:usage:%d", question.ProjectID)
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.Incr(ctx, usageKey)
	}

	s.markProjectOnline(ctx, question.ProjectID, userID)

	return nil
}

// ===================== 辅助函数 =====================

// updateProjectUsage 更新项目使用次数
func (s *QuestionService) updateProjectUsage(ctx context.Context, userID, projectID uint) error {
	now := time.Now()

	// 将任务推送到 Redis 队列
	task := processors.QuestionTask{
		Type:      constant.TaskTypeUsage,
		UserID:    userID,
		ProjectID: projectID,
		Time:      now,
	}
	taskData, _ := json.Marshal(task)

	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.LPush(ctx, "sync:question:usage", string(taskData))
	}

	// 更新 Redis 中的用户集合
	userSetKey := fmt.Sprintf("project:users:%d", projectID)
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.SAdd(ctx, userSetKey, strconv.FormatUint(uint64(userID), 10))
	}

	return nil
}

func (s *QuestionService) markProjectOnline(ctx context.Context, projectID, userID uint) {
	if cache.GlobalCache == nil || projectID == 0 || userID == 0 {
		return
	}

	userIDStr := strconv.FormatUint(uint64(userID), 10)
	projectOnlineKey := fmt.Sprintf("online:project:%d", projectID)
	onlineNow := float64(time.Now().Unix())
	_ = cache.GlobalCache.ZAdd(ctx, projectOnlineKey, onlineNow, userIDStr)
}
