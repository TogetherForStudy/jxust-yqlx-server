package processors

import (
	"context"
	"fmt"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/worker"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/logger"

	json "github.com/bytedance/sonic"
	"gorm.io/gorm"
)

// QuestionTask represents a question-related async task.
type QuestionTask struct {
	Type       string    `json:"type"`
	UserID     uint      `json:"user_id"`
	QuestionID uint      `json:"question_id,omitempty"`
	ProjectID  uint      `json:"project_id,omitempty"`
	Time       time.Time `json:"time"`
	RetryCount int       `json:"retry_count,omitempty"`
}

// Ensure QuestionTask implements worker.Task interface
var _ worker.Task = (*QuestionTask)(nil)

// GetType returns the task type identifier.
func (t *QuestionTask) GetType() string {
	return t.Type
}

// Marshal serializes the task to JSON.
func (t *QuestionTask) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

// GetRetryCount returns the current retry count.
func (t *QuestionTask) GetRetryCount() int {
	return t.RetryCount
}

// IncrementRetry increments the retry counter.
func (t *QuestionTask) IncrementRetry() {
	t.RetryCount++
}

// GetTimestamp returns when the task was created.
func (t *QuestionTask) GetTimestamp() time.Time {
	return t.Time
}

// QuestionTaskProcessor processes question-related tasks.
type QuestionTaskProcessor struct {
	db *gorm.DB
}

// Ensure QuestionTaskProcessor implements worker.TaskProcessor interface
var _ worker.TaskProcessor = (*QuestionTaskProcessor)(nil)

// NewQuestionTaskProcessor creates a new question task processor.
func NewQuestionTaskProcessor(db *gorm.DB) *QuestionTaskProcessor {
	return &QuestionTaskProcessor{
		db: db,
	}
}

// ProcessTask executes the task's business logic.
func (p *QuestionTaskProcessor) ProcessTask(ctx context.Context, task worker.Task) error {
	qt, ok := task.(*QuestionTask)
	if !ok {
		return fmt.Errorf("invalid task type: expected *QuestionTask")
	}

	switch qt.Type {
	case constant.TaskTypeStudy:
		return p.syncStudyToDB(ctx, qt.UserID, qt.QuestionID, qt.Time)
	case constant.TaskTypePractice:
		return p.syncPracticeToDB(ctx, qt.UserID, qt.QuestionID, qt.Time)
	case constant.TaskTypeUsage:
		return p.syncUsageToDB(ctx, qt.UserID, qt.ProjectID, qt.Time)
	default:
		return fmt.Errorf("unknown task type: %s", qt.Type)
	}
}

// Unmarshal deserializes task data from JSON.
func (p *QuestionTaskProcessor) Unmarshal(data []byte) (worker.Task, error) {
	var task QuestionTask
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, err
	}
	return &task, nil
}

// GetSupportedTypes returns the list of task types this processor handles.
func (p *QuestionTaskProcessor) GetSupportedTypes() []string {
	return []string{constant.TaskTypeStudy, constant.TaskTypePractice, constant.TaskTypeUsage}
}

// syncStudyToDB syncs study count to database.
func (p *QuestionTaskProcessor) syncStudyToDB(ctx context.Context, userID, questionID uint, t time.Time) error {
	var existingUsage models.UserQuestionUsage
	err := p.db.WithContext(ctx).Where("user_id = ? AND question_id = ?", userID, questionID).First(&existingUsage).Error

	if err == gorm.ErrRecordNotFound {
		usage := models.UserQuestionUsage{
			UserID:        userID,
			QuestionID:    questionID,
			StudyCount:    1,
			LastStudiedAt: &t,
			CreatedAt:     t,
			UpdatedAt:     t,
		}
		if err := p.db.WithContext(ctx).Create(&usage).Error; err != nil {
			return err
		}
		logger.InfoCtx(ctx, map[string]any{
			"action":      "synced_study_to_db",
			"user_id":     userID,
			"question_id": questionID,
		})
		return nil
	} else if err != nil {
		return err
	}

	if err := p.db.WithContext(ctx).Model(&existingUsage).Updates(map[string]interface{}{
		"study_count":     gorm.Expr("study_count + ?", 1),
		"last_studied_at": t,
		"updated_at":      t,
	}).Error; err != nil {
		return err
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":      "synced_study_to_db",
		"user_id":     userID,
		"question_id": questionID,
	})
	return nil
}

// syncPracticeToDB syncs practice count to database.
func (p *QuestionTaskProcessor) syncPracticeToDB(ctx context.Context, userID, questionID uint, t time.Time) error {
	var existingUsage models.UserQuestionUsage
	err := p.db.WithContext(ctx).Where("user_id = ? AND question_id = ?", userID, questionID).First(&existingUsage).Error

	if err == gorm.ErrRecordNotFound {
		usage := models.UserQuestionUsage{
			UserID:          userID,
			QuestionID:      questionID,
			PracticeCount:   1,
			LastPracticedAt: &t,
			CreatedAt:       t,
			UpdatedAt:       t,
		}
		if err := p.db.WithContext(ctx).Create(&usage).Error; err != nil {
			return err
		}
		logger.InfoCtx(ctx, map[string]any{
			"action":      "synced_practice_to_db",
			"user_id":     userID,
			"question_id": questionID,
		})
		return nil
	} else if err != nil {
		return err
	}

	if err := p.db.WithContext(ctx).Model(&existingUsage).Updates(map[string]interface{}{
		"practice_count":    gorm.Expr("practice_count + ?", 1),
		"last_practiced_at": t,
		"updated_at":        t,
	}).Error; err != nil {
		return err
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":      "synced_practice_to_db",
		"user_id":     userID,
		"question_id": questionID,
	})
	return nil
}

// syncUsageToDB syncs project usage count to database.
func (p *QuestionTaskProcessor) syncUsageToDB(ctx context.Context, userID, projectID uint, t time.Time) error {
	var usage models.UserProjectUsage
	err := p.db.WithContext(ctx).Where("user_id = ? AND project_id = ?", userID, projectID).First(&usage).Error

	if err == gorm.ErrRecordNotFound {
		usage = models.UserProjectUsage{
			UserID:     userID,
			ProjectID:  projectID,
			UsageCount: 1,
			LastUsedAt: t,
			CreatedAt:  t,
			UpdatedAt:  t,
		}
		if err := p.db.WithContext(ctx).Create(&usage).Error; err != nil {
			return err
		}
		logger.InfoCtx(ctx, map[string]any{
			"action":     "synced_usage_to_db",
			"user_id":    userID,
			"project_id": projectID,
		})
		return nil
	} else if err != nil {
		return err
	}

	if err := p.db.WithContext(ctx).Model(&usage).Updates(map[string]interface{}{
		"usage_count":  gorm.Expr("usage_count + ?", 1),
		"last_used_at": t,
		"updated_at":   t,
	}).Error; err != nil {
		return err
	}

	logger.InfoCtx(ctx, map[string]any{
		"action":     "synced_usage_to_db",
		"user_id":    userID,
		"project_id": projectID,
	})
	return nil
}
