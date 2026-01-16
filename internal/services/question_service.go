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
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

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

// SyncTaskType 同步任务类型
type SyncTaskType string

const (
	SyncTaskStudy    SyncTaskType = "study"
	SyncTaskPractice SyncTaskType = "practice"
	SyncTaskUsage    SyncTaskType = "usage"

	SyncQueueKey   = "sync:question:usage"
	MaxSyncRetries = 3
)

// SyncTask 同步任务结构
type SyncTask struct {
	Type       SyncTaskType `json:"type"`
	UserID     uint         `json:"user_id"`
	QuestionID uint         `json:"question_id,omitempty"`
	ProjectID  uint         `json:"project_id,omitempty"`
	Time       time.Time    `json:"time"`
	RetryCount int          `json:"retry_count,omitempty"`
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
			Where("project_id = ? AND is_active = ? AND parent_id IS NULL", project.ID, true).
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

	now := time.Now()
	// 将任务推送到 Redis 队列
	task := SyncTask{
		Type:       SyncTaskStudy,
		UserID:     userID,
		QuestionID: req.QuestionID,
		Time:       now,
	}
	taskData, _ := json.Marshal(task)

	ctx := context.Background()
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.LPush(ctx, SyncQueueKey, string(taskData))
	}

	// 更新 Redis 中的项目刷题次数
	usageKey := fmt.Sprintf("project:usage:%d", question.ProjectID)
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.Incr(ctx, usageKey)
	}

	// 更新项目在线人数统计（每个用户独立TTL 1分钟）
	if cache.GlobalCache != nil {
		userIDStr := strconv.FormatUint(uint64(userID), 10)
		projectOnlineKey := fmt.Sprintf("online:project:%d", question.ProjectID)
		onlineNow := float64(time.Now().Unix())
		_ = cache.GlobalCache.ZAdd(ctx, projectOnlineKey, onlineNow, userIDStr)
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

	now := time.Now()
	// 将任务推送到 Redis 队列
	task := SyncTask{
		Type:       SyncTaskPractice,
		UserID:     userID,
		QuestionID: req.QuestionID,
		Time:       now,
	}
	taskData, _ := json.Marshal(task)

	ctx := context.Background()
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.LPush(ctx, SyncQueueKey, string(taskData))
	}

	// 更新 Redis 中的项目刷题次数
	usageKey := fmt.Sprintf("project:usage:%d", question.ProjectID)
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.Incr(ctx, usageKey)
	}

	// 更新项目在线人数统计（每个用户独立TTL 1分钟）
	if cache.GlobalCache != nil {
		userIDStr := strconv.FormatUint(uint64(userID), 10)
		projectOnlineKey := fmt.Sprintf("online:project:%d", question.ProjectID)
		onlineNow := float64(time.Now().Unix())
		_ = cache.GlobalCache.ZAdd(ctx, projectOnlineKey, onlineNow, userIDStr)
	}

	return nil
}

// ===================== 辅助函数 =====================

// updateProjectUsage 更新项目使用次数
func (s *QuestionService) updateProjectUsage(userID, projectID uint) error {
	now := time.Now()

	// 将任务推送到 Redis 队列
	task := SyncTask{
		Type:      SyncTaskUsage,
		UserID:    userID,
		ProjectID: projectID,
		Time:      now,
	}
	taskData, _ := json.Marshal(task)

	ctx := context.Background()
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.LPush(ctx, SyncQueueKey, string(taskData))
	}

	// 更新 Redis 中的用户集合
	userSetKey := fmt.Sprintf("project:users:%d", projectID)
	if cache.GlobalCache != nil {
		_, _ = cache.GlobalCache.SAdd(ctx, userSetKey, strconv.FormatUint(uint64(userID), 10))
	}

	return nil
}

// StartSyncWorker 启动同步 Worker，从 Redis 队列读取任务并同步到数据库
func (s *QuestionService) StartSyncWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.processSyncQueue(ctx)
		}
	}
}

// processSyncQueue 处理同步队列
func (s *QuestionService) processSyncQueue(ctx context.Context) {
	if cache.GlobalCache == nil {
		return
	}

	for {
		// 从队列中弹出一个任务
		taskData, err := cache.GlobalCache.RPop(ctx, SyncQueueKey)
		if err != nil || taskData == "" {
			break
		}

		var task SyncTask
		if err := json.Unmarshal([]byte(taskData), &task); err != nil {
			continue
		}

		// 执行数据库同步
		if err := s.syncTaskToDB(task); err != nil {
			// 如果未达到重试上限，推回队列重试
			if task.RetryCount < MaxSyncRetries {
				task.RetryCount++
				retryData, _ := json.Marshal(task)
				_, _ = cache.GlobalCache.LPush(ctx, SyncQueueKey, string(retryData))

				logger.WarnCtx(ctx, map[string]any{
					"action":      "sync_task_retry_pushed",
					"task_type":   task.Type,
					"user_id":     task.UserID,
					"retry_count": task.RetryCount,
					"error":       err.Error(),
				})
			} else {
				// 达到重试上限，记录错误并放弃
				logger.ErrorCtx(ctx, map[string]any{
					"action":    "sync_task_to_db_failed_final",
					"task_type": task.Type,
					"user_id":   task.UserID,
					"error":     err.Error(),
					"task_data": taskData,
				})
			}
		}
	}
}

// syncTaskToDB 将单个任务同步到数据库
func (s *QuestionService) syncTaskToDB(task SyncTask) error {
	switch task.Type {
	case SyncTaskStudy:
		return s.syncStudyToDB(task.UserID, task.QuestionID, task.Time)
	case SyncTaskPractice:
		return s.syncPracticeToDB(task.UserID, task.QuestionID, task.Time)
	case SyncTaskUsage:
		return s.syncUsageToDB(task.UserID, task.ProjectID, task.Time)
	default:
		return fmt.Errorf("unknown task type: %s", task.Type)
	}
}

func (s *QuestionService) syncStudyToDB(userID, questionID uint, t time.Time) error {
	var existingUsage models.UserQuestionUsage
	err := s.db.Where("user_id = ? AND question_id = ?", userID, questionID).First(&existingUsage).Error

	if err == gorm.ErrRecordNotFound {
		usage := models.UserQuestionUsage{
			UserID:        userID,
			QuestionID:    questionID,
			StudyCount:    1,
			LastStudiedAt: &t,
			CreatedAt:     t,
			UpdatedAt:     t,
		}
		return s.db.Create(&usage).Error
	} else if err != nil {
		return err
	}

	return s.db.Model(&existingUsage).Updates(map[string]interface{}{
		"study_count":     gorm.Expr("study_count + ?", 1),
		"last_studied_at": t,
		"updated_at":      t,
	}).Error
}

func (s *QuestionService) syncPracticeToDB(userID, questionID uint, t time.Time) error {
	var existingUsage models.UserQuestionUsage
	err := s.db.Where("user_id = ? AND question_id = ?", userID, questionID).First(&existingUsage).Error

	if err == gorm.ErrRecordNotFound {
		usage := models.UserQuestionUsage{
			UserID:          userID,
			QuestionID:      questionID,
			PracticeCount:   1,
			LastPracticedAt: &t,
			CreatedAt:       t,
			UpdatedAt:       t,
		}
		return s.db.Create(&usage).Error
	} else if err != nil {
		return err
	}

	return s.db.Model(&existingUsage).Updates(map[string]interface{}{
		"practice_count":    gorm.Expr("practice_count + ?", 1),
		"last_practiced_at": t,
		"updated_at":        t,
	}).Error
}

func (s *QuestionService) syncUsageToDB(userID, projectID uint, t time.Time) error {
	var usage models.UserProjectUsage
	err := s.db.Where("user_id = ? AND project_id = ?", userID, projectID).First(&usage).Error

	if err == gorm.ErrRecordNotFound {
		usage = models.UserProjectUsage{
			UserID:     userID,
			ProjectID:  projectID,
			UsageCount: 1,
			LastUsedAt: t,
			CreatedAt:  t,
			UpdatedAt:  t,
		}
		return s.db.Create(&usage).Error
	} else if err != nil {
		return err
	}

	return s.db.Model(&usage).Updates(map[string]interface{}{
		"usage_count":  gorm.Expr("usage_count + ?", 1),
		"last_used_at": t,
		"updated_at":   t,
	}).Error
}
