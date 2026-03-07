package services

import (
	"context"
	"fmt"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/pkg/apperr"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/constant"
	"github.com/TogetherForStudy/jxust-yqlx-server/pkg/utils"

	json "github.com/bytedance/sonic"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type CourseTableService struct {
	db *gorm.DB
}

func NewCourseTableService(db *gorm.DB) *CourseTableService {
	return &CourseTableService{
		db: db,
	}
}

// GetUserCourseTable 获取用户课程表 (保持向后兼容)
func (s *CourseTableService) GetUserCourseTable(ctx context.Context, userID uint, semester string) (*response.CourseTableResponse, error) {
	return s.GetUserCourseTableWithVersion(ctx, userID, semester, nil)
}

// GetUserCourseTableWithVersion 获取用户课程表（带版本检测）
func (s *CourseTableService) GetUserCourseTableWithVersion(ctx context.Context, userID uint, semester string, clientLastModified *int64) (*response.CourseTableResponse, error) {
	// 先获取用户信息，获取其班级ID
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperr.New(constant.CommonUserNotFound)
		}
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询用户信息失败: %w", err))
	}

	// 检查用户是否设置了班级ID
	if user.ClassID == "" {
		return nil, apperr.New(constant.CourseTableClassNotSet)
	}

	// 获取最新的数据修改时间和数据
	latestModified, courseData, classID, err := s.getLatestCourseData(ctx, userID, user.ClassID, semester)
	if err != nil {
		return nil, err
	}

	// 检查客户端数据是否为最新
	if clientLastModified != nil && *clientLastModified >= latestModified {
		return &response.CourseTableResponse{
			ClassID:      classID,
			Semester:     semester,
			LastModified: latestModified,
			HasChanges:   false,
		}, nil
	}

	// 返回完整数据
	return &response.CourseTableResponse{
		ClassID:      classID,
		Semester:     semester,
		CourseData:   courseData,
		LastModified: latestModified,
		HasChanges:   true,
	}, nil
}

// getLatestCourseData 获取最新课程数据和修改时间
func (s *CourseTableService) getLatestCourseData(ctx context.Context, userID uint, classID, semester string) (int64, datatypes.JSON, string, error) {
	// 优先检查用户个性化课表
	var userSchedule models.ScheduleUser
	if err := s.db.WithContext(ctx).Where("user_id = ? AND class_id = ? AND semester = ?", userID, classID, semester).First(&userSchedule).Error; err == nil {
		return userSchedule.UpdatedAt.Unix(), userSchedule.Schedule, userSchedule.ClassID, nil
	} else if err != gorm.ErrRecordNotFound {
		return 0, nil, "", apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询用户个性课表失败: %w", err))
	}

	// 查询班级默认课表
	var courseTable models.CourseTable
	if err := s.db.WithContext(ctx).Where("class_id = ? AND semester = ?", classID, semester).First(&courseTable).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil, "", apperr.New(constant.CourseTableScheduleNotFound)
		}
		return 0, nil, "", apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询课程表失败: %w", err))
	}

	return courseTable.UpdatedAt.Unix(), courseTable.CourseData, courseTable.ClassID, nil
}

// EditUserCourseCell 编辑用户课程表的单个格子（1-35）
func (s *CourseTableService) EditUserCourseCell(ctx context.Context, userID uint, semester string, index string, value datatypes.JSON) error {
	// 获取用户信息
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.New(constant.CommonUserNotFound)
		}
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询用户信息失败: %w", err))
	}
	if user.ClassID == "" {
		return apperr.New(constant.CourseTableClassNotSet)
	}

	// 查询或初始化个人课表
	var userSchedule models.ScheduleUser
	if err := s.db.WithContext(ctx).Where("user_id = ? AND class_id = ? AND semester = ?", userID, user.ClassID, semester).First(&userSchedule).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			// 首次编辑：以默认班级课表为基底
			var courseTable models.CourseTable
			if e := s.db.WithContext(ctx).Where("class_id = ? AND semester = ?", user.ClassID, semester).First(&courseTable).Error; e != nil {
				if e == gorm.ErrRecordNotFound {
					return apperr.New(constant.CourseTableScheduleNotFound)
				}
				return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询课程表失败: %w", e))
			}

			var scheduleMap map[string]any
			if e := json.Unmarshal(courseTable.CourseData, &scheduleMap); e != nil {
				return apperr.Wrap(constant.CommonInternal, fmt.Errorf("解析课程表失败: %w", e))
			}
			var cellValue any
			if e := json.Unmarshal(value, &cellValue); e != nil {
				return apperr.Wrap(constant.CommonInternal, fmt.Errorf("解析提交的格子数据失败: %w", e))
			}
			scheduleMap[index] = cellValue

			bytesData, e := json.Marshal(scheduleMap)
			if e != nil {
				return apperr.Wrap(constant.CommonInternal, fmt.Errorf("序列化课程表失败: %w", e))
			}
			newSchedule := models.ScheduleUser{
				UserID:   userID,
				ClassID:  user.ClassID,
				Semester: semester,
				Schedule: datatypes.JSON(bytesData),
			}
			if e := s.db.WithContext(ctx).Create(&newSchedule).Error; e != nil {
				return apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建用户个性课表失败: %w", e))
			}
			return nil
		}
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询用户个性课表失败: %w", err))
	}

	// 已存在个人课表，更新指定格子
	var scheduleMap map[string]any
	if e := json.Unmarshal(userSchedule.Schedule, &scheduleMap); e != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("解析用户个性课表失败: %w", e))
	}
	var cellValue any
	if e := json.Unmarshal(value, &cellValue); e != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("解析提交的格子数据失败: %w", e))
	}
	scheduleMap[index] = cellValue

	bytesData, e := json.Marshal(scheduleMap)
	if e != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("序列化课程表失败: %w", e))
	}
	if e := s.db.WithContext(ctx).Model(&models.ScheduleUser{}).
		Where("user_id = ? AND class_id = ? AND semester = ?", userID, user.ClassID, semester).
		Updates(map[string]any{"schedule": datatypes.JSON(bytesData), "class_id": user.ClassID}).Error; e != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新用户个性课表失败: %w", e))
	}
	return nil
}

// SearchClasses 模糊搜索班级
func (s *CourseTableService) SearchClasses(ctx context.Context, keyword string, page, size int) (*response.SearchClassResponse, error) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 10
	}

	offset := (page - 1) * size

	// 查询总数
	var total int64
	if err := s.db.WithContext(ctx).Model(&models.CourseTable{}).
		Where("class_id LIKE ?", "%"+keyword+"%").
		Distinct("class_id").
		Count(&total).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询班级总数失败: %w", err))
	}

	// 查询班级列表（去重）
	var courseTables []models.CourseTable
	if err := s.db.WithContext(ctx).Select("DISTINCT class_id").
		Where("class_id LIKE ?", "%"+keyword+"%").
		Order("class_id ASC").
		Offset(offset).
		Limit(size).
		Find(&courseTables).Error; err != nil {
		return nil, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询班级列表失败: %w", err))
	}

	// 转换为响应格式
	var classList []response.ClassInfo
	for _, ct := range courseTables {
		classList = append(classList, response.ClassInfo{
			ClassID:  ct.ClassID,
			Semester: ct.Semester,
		})
	}

	return &response.SearchClassResponse{
		List:  classList,
		Total: total,
		Page:  page,
		Size:  size,
	}, nil
}

// UpdateUserClass 更新用户班级（普通用户2次绑定限制）
func (s *CourseTableService) UpdateUserClass(ctx context.Context, userID uint, classID string) (err error) {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 获取用户信息
		var user models.User
		if e := tx.Where("id = ?", userID).First(&user).Error; e != nil {
			if e == gorm.ErrRecordNotFound {
				return apperr.New(constant.CommonUserNotFound)
			}
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询用户信息失败: %w", e))
		}

		// 查询绑定记录（仅用于读取 bind_count，未找到视为0）
		var br models.BindRecord
		var hasRecord bool
		if e := tx.Where("user_id = ?", userID).First(&br).Error; e != nil {
			if e == gorm.ErrRecordNotFound {
				hasRecord = false
			} else {
				return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询绑定记录失败: %w", e))
			}
		} else {
			hasRecord = true
		}

		// 检查是否有全量班级更新权限或是管理员，若无则执行绑定次数限制
		isPrivileged := utils.IsAdmin(ctx) || utils.HasPermission(ctx, constant.PermissionCourseTableClassUpdateAll)

		// 普通权限用户限制2次绑定（基于 bind_count）
		if !isPrivileged {
			bindCount := 0
			if hasRecord {
				bindCount = br.BindCount
			}
			if bindCount >= 2 {
				return apperr.New(constant.CourseTableBindLimitReached)
			}
		}

		// 检查班级存在
		var exists int64
		if e := tx.Model(&models.CourseTable{}).Where("class_id = ?", classID).Count(&exists).Error; e != nil {
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询班级信息失败: %w", e))
		}
		if exists == 0 {
			return apperr.New(constant.CourseTableClassNotFound)
		}

		// 如果班级未变化，不增加绑定次数
		if user.ClassID == classID {
			return nil
		}

		// 更新用户班级
		if e := tx.Model(&models.User{}).Where("id = ?", userID).Update("class_id", classID).Error; e != nil {
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新用户班级失败: %w", e))
		}

		// 成功绑定：创建或更新绑定记录（仅变更 bind_count）
		if hasRecord {
			if e := tx.Model(&models.BindRecord{}).
				Where("user_id = ?", userID).
				Updates(map[string]any{
					"bind_count": gorm.Expr("bind_count + 1"),
				}).Error; e != nil {
				return apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新绑定次数失败: %w", e))
			}
		} else {
			newRecord := models.BindRecord{
				UserID:    userID,
				BindCount: 1,
			}
			if e := tx.Create(&newRecord).Error; e != nil {
				return apperr.Wrap(constant.CommonInternal, fmt.Errorf("创建绑定记录失败: %w", e))
			}
		}

		return nil
	})
}

// GetUserBindCount 获取当前用户的课表绑定次数
func (s *CourseTableService) GetUserBindCount(ctx context.Context, userID uint) (int, error) {
	var br models.BindRecord
	if err := s.db.WithContext(ctx).Where("user_id = ?", userID).First(&br).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询绑定记录失败: %w", err))
	}
	return br.BindCount, nil
}

// ResetUserSchedule 重置用户个人课表（删除个人编辑的数据条目）
func (s *CourseTableService) ResetUserSchedule(ctx context.Context, userID uint, semester string) error {
	var user models.User
	if err := s.db.WithContext(ctx).Where("id = ?", userID).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return apperr.New(constant.CommonUserNotFound)
		}
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询用户信息失败: %w", err))
	}
	if user.ClassID == "" {
		return apperr.New(constant.CourseTableClassNotSet)
	}

	result := s.db.WithContext(ctx).
		Where("user_id = ? AND class_id = ? AND semester = ?", userID, user.ClassID, semester).
		Delete(&models.ScheduleUser{})
	if result.Error != nil {
		return apperr.Wrap(constant.CommonInternal, fmt.Errorf("重置个人课表失败: %w", result.Error))
	}
	if result.RowsAffected == 0 {
		return apperr.New(constant.CourseTablePersonalScheduleNotFound)
	}
	return nil
}

// ResetUserBindCountToOne 将指定用户的绑定次数置为1（管理员操作）
func (s *CourseTableService) ResetUserBindCountToOne(ctx context.Context, targetUserID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var br models.BindRecord
		if err := tx.Where("user_id = ?", targetUserID).First(&br).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return apperr.Wrap(constant.CommonInternal, fmt.Errorf("查询绑定记录失败: %w", err))
			}
		}
		if err := tx.Model(&models.BindRecord{}).
			Where("user_id = ?", targetUserID).
			Update("bind_count", 1).Error; err != nil {
			return apperr.Wrap(constant.CommonInternal, fmt.Errorf("更新绑定次数失败: %w", err))
		}
		return nil
	})
}
