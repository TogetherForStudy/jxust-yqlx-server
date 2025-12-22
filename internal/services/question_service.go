package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"

	"gorm.io/gorm"
)

type QuestionService struct {
	db *gorm.DB
}

func NewQuestionService(db *gorm.DB) *QuestionService {
	return &QuestionService{
		db: db,
	}
}

// ===================== 项目查询 =====================

// GetProjectByID 根据ID获取项目
func (s *QuestionService) GetProjectByID(projectID uint) (*models.QuestionProject, error) {
	var project models.QuestionProject
	if err := s.db.First(&project, projectID).Error; err != nil {
		return nil, err
	}
	return &project, nil
}

// GetProjects 获取项目列表
func (s *QuestionService) GetProjects(userID uint) ([]response.QuestionProjectResponse, error) {
	var projects []models.QuestionProject

	if err := s.db.Where("is_active = ?", true).
		Order("sort ASC, created_at DESC").
		Find(&projects).Error; err != nil {
		return nil, err
	}

	var result []response.QuestionProjectResponse
	for _, project := range projects {
		// 获取项目下所有启用题目的 ID（用于统计数量和刷题次数）
		var questionIDs []uint
		if err := s.db.Model(&models.Question{}).
			Where("project_id = ? AND is_active = ? AND type != 0", project.ID, true).
			Pluck("id", &questionIDs).Error; err != nil {
			return nil, err
		}

		// 题目数量
		questionCount := int64(len(questionIDs))

		// 获取使用过该项目的用户数量
		var userCount int64
		s.db.Model(&models.UserProjectUsage{}).
			Where("project_id = ?", project.ID).
			Count(&userCount)

		// 统计项目内题目总刷题次数（学习+练习）
		var usageCount int64
		if len(questionIDs) > 0 {
			if err := s.db.Model(&models.UserQuestionUsage{}).
				Where("question_id IN ?", questionIDs).
				Select("COALESCE(SUM(study_count + practice_count), 0)").
				Scan(&usageCount).Error; err != nil {
				return nil, err
			}
		}

		result = append(result, response.ToQuestionProjectResponse(&project, questionCount, userCount, usageCount))
	}

	return result, nil
}

// ===================== 获取题目 =====================

// GetQuestions 获取题目列表（只返回题目ID数组，支持顺序/乱序）
func (s *QuestionService) GetQuestions(userID uint, req *request.GetQuestionRequest) (*response.QuestionListResponse, error) {
	// 更新项目使用次数
	if err := s.updateProjectUsage(userID, req.ProjectID); err != nil {
		return nil, fmt.Errorf("更新项目使用记录失败: %w", err)
	}

	// 获取项目下所有启用的主题目/独立题（parent_id 为 null）的ID
	var questionIDs []uint
	query := s.db.Model(&models.Question{}).
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
		return nil, fmt.Errorf("获取题目列表失败: %w", err)
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
func (s *QuestionService) getQuestionByID(questionID uint) (*models.Question, error) {
	var question models.Question
	if err := s.db.Preload("SubQuestions", "is_active = ?", true).First(&question, questionID).Error; err != nil {
		return nil, err
	}
	return &question, nil
}

// GetQuestionByID 获取题目详情（公开方法）
func (s *QuestionService) GetQuestionByID(userID, questionID uint) (*response.QuestionResponse, error) {
	// 获取题目
	question, err := s.getQuestionByID(questionID)
	if err != nil {
		return nil, fmt.Errorf("题目不存在")
	}

	// 检查题目是否启用
	if !question.IsActive {
		return nil, fmt.Errorf("题目已禁用")
	}

	// 获取用户使用记录
	var usage models.UserQuestionUsage
	s.db.Where("user_id = ? AND question_id = ?", userID, questionID).First(&usage)

	// 批量获取子题的使用记录
	var subQuestionIDs []uint
	for _, subQ := range question.SubQuestions {
		subQuestionIDs = append(subQuestionIDs, subQ.ID)
	}

	var usages []models.UserQuestionUsage
	if len(subQuestionIDs) > 0 {
		s.db.Where("user_id = ? AND question_id IN ?", userID, subQuestionIDs).Find(&usages)
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
func (s *QuestionService) RecordStudy(userID uint, req *request.RecordStudyRequest) error {
	// 验证题目存在
	_, err := s.getQuestionByID(req.QuestionID)
	if err != nil {
		return fmt.Errorf("题目不存在")
	}

	// 更新学习次数
	now := time.Now()
	var existingUsage models.UserQuestionUsage
	err = s.db.Where("user_id = ? AND question_id = ?", userID, req.QuestionID).
		First(&existingUsage).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		usage := models.UserQuestionUsage{
			UserID:        userID,
			QuestionID:    req.QuestionID,
			StudyCount:    1,
			LastStudiedAt: &now,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		return s.db.Create(&usage).Error
	}

	// 更新记录
	return s.db.Model(&existingUsage).Updates(map[string]interface{}{
		"study_count":     gorm.Expr("study_count + ?", 1),
		"last_studied_at": now,
		"updated_at":      now,
	}).Error
}

// SubmitPractice 提交做题（仅记录做题次数）
func (s *QuestionService) SubmitPractice(userID uint, req *request.SubmitPracticeRequest) error {
	// 验证题目存在
	_, err := s.getQuestionByID(req.QuestionID)
	if err != nil {
		return fmt.Errorf("题目不存在")
	}

	// 更新做题次数
	now := time.Now()
	var existingUsage models.UserQuestionUsage
	err = s.db.Where("user_id = ? AND question_id = ?", userID, req.QuestionID).
		First(&existingUsage).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		usage := models.UserQuestionUsage{
			UserID:          userID,
			QuestionID:      req.QuestionID,
			PracticeCount:   1,
			LastPracticedAt: &now,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		return s.db.Create(&usage).Error
	}

	// 更新记录
	return s.db.Model(&existingUsage).Updates(map[string]interface{}{
		"practice_count":    gorm.Expr("practice_count + ?", 1),
		"last_practiced_at": now,
		"updated_at":        now,
	}).Error
}

// ===================== 辅助函数 =====================

// updateProjectUsage 更新项目使用次数
func (s *QuestionService) updateProjectUsage(userID, projectID uint) error {
	now := time.Now()

	var usage models.UserProjectUsage
	err := s.db.Where("user_id = ? AND project_id = ?", userID, projectID).
		First(&usage).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		usage = models.UserProjectUsage{
			UserID:     userID,
			ProjectID:  projectID,
			UsageCount: 1,
			LastUsedAt: now,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
		return s.db.Create(&usage).Error
	} else if err != nil {
		return err
	}

	// 更新记录
	return s.db.Model(&usage).Updates(map[string]interface{}{
		"usage_count":  gorm.Expr("usage_count + ?", 1),
		"last_used_at": now,
		"updated_at":   now,
	}).Error
}
