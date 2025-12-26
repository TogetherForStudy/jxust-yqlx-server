package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/cache"

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
			Where("project_id = ? AND is_active = ?", project.ID, true).
			Pluck("id", &questionIDs).Error; err != nil {
			return nil, err
		}

		// 题目数量
		questionCount := int64(len(questionIDs))

		// 从 Redis 获取使用过该项目的用户数量
		ctx := context.Background()
		var userCount int64
		userSetKey := fmt.Sprintf("project:users:%d", project.ID)
		if cache.GlobalCache != nil {
			var err error
			userCount, err = cache.GlobalCache.SCard(ctx, userSetKey)
			if err != nil {
				// 如果 Redis 查询失败，回退到数据库查询
				s.db.Model(&models.UserProjectUsage{}).
					Where("project_id = ?", project.ID).
					Count(&userCount)

				// 从数据库初始化Redis：查询所有使用过该项目的用户ID并添加到Redis集合
				var userIDs []uint
				if err := s.db.Model(&models.UserProjectUsage{}).
					Where("project_id = ?", project.ID).
					Pluck("user_id", &userIDs).Error; err == nil && len(userIDs) > 0 {
					// 将用户ID转换为字符串并添加到Redis集合
					members := make([]interface{}, len(userIDs))
					for i, id := range userIDs {
						members[i] = strconv.FormatUint(uint64(id), 10)
					}
					_, _ = cache.GlobalCache.SAdd(ctx, userSetKey, members...)
				}
			}
		} else {
			// 如果 Redis 未初始化，使用数据库查询
			s.db.Model(&models.UserProjectUsage{}).
				Where("project_id = ?", project.ID).
				Count(&userCount)
		}

		// 从 Redis 获取项目内题目总刷题次数（学习+练习）
		var usageCount int64
		usageKey := fmt.Sprintf("project:usage:%d", project.ID)
		if cache.GlobalCache != nil {
			var err error
			usageCount, err = cache.GlobalCache.GetInt(ctx, usageKey)
			if err != nil {
				// 如果 Redis 查询失败，回退到数据库查询
				if len(questionIDs) > 0 {
					if err := s.db.Model(&models.UserQuestionUsage{}).
						Where("question_id IN ?", questionIDs).
						Select("COALESCE(SUM(study_count + practice_count), 0)").
						Scan(&usageCount).Error; err != nil {
						return nil, err
					}
				}
				// 从数据库初始化Redis：将查询到的值写入Redis（包括0值，避免每次都查询数据库）
				// 即使questionIDs为空，usageCount为0，也写入Redis以保持一致性
				// 使用 time.Duration(0) 表示永不过期
				noExpiration := time.Duration(0)
				_ = cache.GlobalCache.Set(ctx, usageKey, strconv.FormatInt(usageCount, 10), &noExpiration)
			}
		} else {
			// 如果 Redis 未初始化，使用数据库查询
			if len(questionIDs) > 0 {
				if err := s.db.Model(&models.UserQuestionUsage{}).
					Where("question_id IN ?", questionIDs).
					Select("COALESCE(SUM(study_count + practice_count), 0)").
					Scan(&usageCount).Error; err != nil {
					return nil, err
				}
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
	question, err := s.getQuestionByID(req.QuestionID)
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
		if err := s.db.Create(&usage).Error; err != nil {
			return err
		}
	} else {
		// 更新记录
		if err := s.db.Model(&existingUsage).Updates(map[string]interface{}{
			"study_count":     gorm.Expr("study_count + ?", 1),
			"last_studied_at": now,
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}
	}

	// 更新 Redis 中的项目刷题次数
	ctx := context.Background()
	usageKey := fmt.Sprintf("project:usage:%d", question.ProjectID)
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.Incr(ctx, usageKey)
	}

	return nil
}

// SubmitPractice 提交做题（仅记录做题次数）
func (s *QuestionService) SubmitPractice(userID uint, req *request.SubmitPracticeRequest) error {
	// 验证题目存在
	question, err := s.getQuestionByID(req.QuestionID)
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
		if err := s.db.Create(&usage).Error; err != nil {
			return err
		}
	} else {
		// 更新记录
		if err := s.db.Model(&existingUsage).Updates(map[string]interface{}{
			"practice_count":    gorm.Expr("practice_count + ?", 1),
			"last_practiced_at": now,
			"updated_at":        now,
		}).Error; err != nil {
			return err
		}
	}

	// 更新 Redis 中的项目刷题次数
	ctx := context.Background()
	usageKey := fmt.Sprintf("project:usage:%d", question.ProjectID)
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.Incr(ctx, usageKey)
	}

	return nil
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
		if err := s.db.Create(&usage).Error; err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else {
		// 更新记录
		if err := s.db.Model(&usage).Updates(map[string]interface{}{
			"usage_count":  gorm.Expr("usage_count + ?", 1),
			"last_used_at": now,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}
	}

	// 更新 Redis 中的用户集合
	ctx := context.Background()
	userSetKey := fmt.Sprintf("project:users:%d", projectID)
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.SAdd(ctx, userSetKey, strconv.FormatUint(uint64(userID), 10))
	}

	return nil
}
