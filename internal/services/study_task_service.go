package services

import (
	"errors"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"

	"gorm.io/gorm"
)

type StudyTaskService struct {
	db *gorm.DB
}

func NewStudyTaskService(db *gorm.DB) *StudyTaskService {
	return &StudyTaskService{
		db: db,
	}
}

// CreateStudyTask 创建学习任务
func (s *StudyTaskService) CreateStudyTask(userID uint, req *request.CreateStudyTaskRequest) (*response.StudyTaskResponse, error) {

	// 解析截止日期
	var dueDate *time.Time
	if req.DueDate != "" {
		parsedTime, err := utils.ParseDateTime(req.DueDate)
		if err != nil {
			return nil, errors.New("截止时间格式错误")
		}
		dueDate = parsedTime
	}

	// 创建学习任务
	task := models.StudyTask{
		UserID:      userID,
		Title:       req.Title,
		Description: req.Description,
		DueDate:     dueDate,
		Priority:    models.StudyTaskPriority(req.Priority),
		Status:      models.StudyTaskStatusPending,
	}

	if err := s.db.Create(&task).Error; err != nil {
		return nil, err
	}

	return &response.StudyTaskResponse{
		ID:          task.ID,
		UserID:      task.UserID,
		Title:       task.Title,
		Description: task.Description,
		DueDate:     task.DueDate,
		Priority:    task.Priority,
		Status:      task.Status,
		CompletedAt: task.CompletedAt,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}, nil
}

// GetStudyTasks 获取用户学习任务列表
func (s *StudyTaskService) GetStudyTasks(userID uint, req *request.GetStudyTasksRequest) (*response.PageResponse, error) {
	var tasks []models.StudyTask
	var total int64

	// 构建查询
	query := s.db.Model(&models.StudyTask{}).Where("user_id = ?", userID)

	// 状态过滤
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}

	// 优先级过滤
	if req.Priority != nil {
		query = query.Where("priority = ?", *req.Priority)
	}

	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("title LIKE ? OR description LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (req.Page - 1) * req.Size
	if err := query.Order("due_date ASC, created_at DESC").
		Offset(offset).
		Limit(req.Size).
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	var taskResponses []response.StudyTaskResponse
	for _, task := range tasks {
		taskResponses = append(taskResponses, response.StudyTaskResponse{
			ID:          task.ID,
			UserID:      task.UserID,
			Title:       task.Title,
			Description: task.Description,
			DueDate:     task.DueDate,
			Priority:    task.Priority,
			Status:      task.Status,
			CompletedAt: task.CompletedAt,
			CreatedAt:   task.CreatedAt,
			UpdatedAt:   task.UpdatedAt,
		})
	}

	return &response.PageResponse{
		Data:  taskResponses,
		Total: total,
		Page:  req.Page,
		Size:  req.Size,
	}, nil
}

// GetStudyTaskByID 根据ID获取学习任务详情
func (s *StudyTaskService) GetStudyTaskByID(taskID uint, userID uint) (*response.StudyTaskResponse, error) {
	var task models.StudyTask
	if err := s.db.Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("学习任务不存在或无权限访问")
		}
		return nil, err
	}

	return &response.StudyTaskResponse{
		ID:          task.ID,
		UserID:      task.UserID,
		Title:       task.Title,
		Description: task.Description,
		DueDate:     task.DueDate,
		Priority:    task.Priority,
		Status:      task.Status,
		CompletedAt: task.CompletedAt,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}, nil
}

// UpdateStudyTask 更新学习任务
func (s *StudyTaskService) UpdateStudyTask(taskID uint, userID uint, req *request.UpdateStudyTaskRequest) (*response.StudyTaskResponse, error) {
	// 查找任务
	var task models.StudyTask
	if err := s.db.Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New("学习任务不存在或无权限访问")
		}
		return nil, err
	}

	// 更新字段
	updates := make(map[string]interface{})

	if req.Title != "" {
		updates["title"] = req.Title
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.DueDate != "" {
		dueDate, err := utils.ParseDateTime(req.DueDate)
		if err != nil {
			return nil, errors.New("截止日期格式错误")
		}
		updates["due_date"] = dueDate
	}
	if req.Priority != nil {
		updates["priority"] = models.StudyTaskPriority(*req.Priority)
	}
	if req.Status != nil {
		updates["status"] = models.StudyTaskStatus(*req.Status)
		// 如果标记为已完成，设置完成时间
		if *req.Status == 2 { // 已完成
			now := utils.GetLocalTime()
			updates["completed_at"] = &now
		} else if *req.Status == 1 { // 重新设为待完成
			updates["completed_at"] = nil
		}
	}

	if len(updates) > 0 {
		if err := s.db.Model(&task).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetStudyTaskByID(taskID, userID)
}

// DeleteStudyTask 删除学习任务
func (s *StudyTaskService) DeleteStudyTask(taskID uint, userID uint) error {
	// 查找任务
	var task models.StudyTask
	if err := s.db.Where("id = ? AND user_id = ?", taskID, userID).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return errors.New("学习任务不存在或无权限访问")
		}
		return err
	}

	// 软删除
	return s.db.Delete(&task).Error
}

// GetStudyTaskStats 获取用户学习任务统计
func (s *StudyTaskService) GetStudyTaskStats(userID uint) (*response.StudyTaskStatsResponse, error) {
	stats := &response.StudyTaskStatsResponse{}

	// 总任务数
	var totalCount int64
	if err := s.db.Model(&models.StudyTask{}).Where("user_id = ?", userID).Count(&totalCount).Error; err != nil {
		return nil, err
	}
	stats.TotalCount = int(totalCount)

	// 待完成数量
	var pendingCount int64
	if err := s.db.Model(&models.StudyTask{}).
		Where("user_id = ? AND status = ?", userID, models.StudyTaskStatusPending).
		Count(&pendingCount).Error; err != nil {
		return nil, err
	}
	stats.PendingCount = int(pendingCount)

	// 已完成数量
	var completedCount int64
	if err := s.db.Model(&models.StudyTask{}).
		Where("user_id = ? AND status = ?", userID, models.StudyTaskStatusCompleted).
		Count(&completedCount).Error; err != nil {
		return nil, err
	}
	stats.CompletedCount = int(completedCount)

	return stats, nil
}

// GetCompletedTasks 获取已完成的任务（历史记录）
func (s *StudyTaskService) GetCompletedTasks(userID uint, page, size int) (*response.PageResponse, error) {
	var tasks []models.StudyTask
	var total int64

	// 构建查询
	query := s.db.Model(&models.StudyTask{}).
		Where("user_id = ? AND status = ?", userID, models.StudyTaskStatusCompleted)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (page - 1) * size
	if err := query.Order("completed_at DESC").
		Offset(offset).
		Limit(size).
		Find(&tasks).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	var taskResponses []response.StudyTaskResponse
	for _, task := range tasks {
		taskResponses = append(taskResponses, response.StudyTaskResponse{
			ID:          task.ID,
			UserID:      task.UserID,
			Title:       task.Title,
			Description: task.Description,
			DueDate:     task.DueDate,
			Priority:    task.Priority,
			Status:      task.Status,
			CompletedAt: task.CompletedAt,
			CreatedAt:   task.CreatedAt,
			UpdatedAt:   task.UpdatedAt,
		})
	}

	return &response.PageResponse{
		Data:  taskResponses,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}
