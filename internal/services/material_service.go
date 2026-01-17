package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/request"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/dto/response"
	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"

	"gorm.io/gorm"
)

type MaterialService struct {
	db *gorm.DB
}

func NewMaterialService(db *gorm.DB) *MaterialService {
	return &MaterialService{db: db}
}

// ==================== 资料管理 ====================

// GetMaterialList 获取资料列表（按热度排序）
func (s *MaterialService) GetMaterialList(ctx context.Context, req *request.MaterialListRequest) ([]response.MaterialListResponse, int64, error) {
	var materials []models.Material
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Material{}).
		Joins("LEFT JOIN material_descs ON materials.md5 = material_descs.md5").
		Joins("LEFT JOIN material_categories ON materials.category_id = material_categories.id")

	// 按分类ID筛选
	if req.CategoryID != nil {
		query = query.Where("materials.category_id = ?", *req.CategoryID)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("获取资料总数失败: %w", err)
	}

	// 排序
	sortBy := "material_descs.total_hotness DESC"
	if req.SortBy == "time" {
		sortBy = "materials.created_at DESC"
	}
	query = query.Order(sortBy)

	// 分页
	offset := (req.Page - 1) * req.PageSize
	query = query.Offset(offset).Limit(req.PageSize)

	// 预加载关联数据
	if err := query.Preload("Category").Preload("Desc").Find(&materials).Error; err != nil {
		return nil, 0, fmt.Errorf("获取资料列表失败: %w", err)
	}

	// 转换为响应格式
	var result []response.MaterialListResponse
	for _, material := range materials {
		// 使用预加载的资料描述数据
		var desc models.MaterialDesc
		if material.Desc != nil {
			desc = *material.Desc
		}

		item := response.MaterialListResponse{
			ID:            material.ID,
			MD5:           material.MD5,
			FileName:      material.FileName,
			FileSize:      material.FileSize,
			CategoryID:    material.CategoryID,
			Tags:          desc.Tags,
			TotalHotness:  desc.TotalHotness,
			PeriodHotness: desc.PeriodHotness,
			DownloadCount: desc.DownloadCount,
			CreatedAt:     material.CreatedAt,
		}
		result = append(result, item)
	}

	return result, total, nil
}

// GetMaterialByMD5 根据MD5获取资料详情
func (s *MaterialService) GetMaterialByMD5(ctx context.Context, md5 string, userID *uint) (*response.MaterialDetailResponse, error) {
	var material models.Material
	if err := s.db.WithContext(ctx).Where("md5 = ?", md5).Preload("Category").Preload("Desc").First(&material).Error; err != nil {
		return nil, fmt.Errorf("资料不存在: %w", err)
	}

	// 使用预加载的资料描述
	var desc models.MaterialDesc
	if material.Desc != nil {
		desc = *material.Desc
	}

	// 获取用户评分（如果用户已登录）
	var userRating *int
	if userID != nil {
		var log models.MaterialLog
		err := s.db.WithContext(ctx).Model(&models.MaterialLog{}).Where("user_id = ? AND material_md5 = ? AND type = ?",
			*userID, md5, models.MaterialLogTypeRating).First(&log).Error
		if err == nil && log.Rating != nil {
			userRating = log.Rating
		}
	}

	// 构建完整响应
	return &response.MaterialDetailResponse{
		ID:            material.ID,
		MD5:           material.MD5,
		FileName:      material.FileName,
		FileSize:      material.FileSize,
		CategoryID:    material.CategoryID,
		CategoryName:  material.Category.Name,
		CategoryPath:  s.buildCategoryPath(ctx, material.CategoryID),
		Tags:          desc.Tags,
		Description:   desc.Description,
		ExternalLink:  desc.ExternalLink,
		TotalHotness:  desc.TotalHotness,
		ViewCount:     desc.ViewCount,
		DownloadCount: desc.DownloadCount,
		Rating:        desc.Rating,
		RatingCount:   desc.RatingCount,
		UserRating:    userRating,
		IsRecommended: desc.IsRecommended,
		CreatedAt:     material.CreatedAt,
	}, nil
}

// DeleteMaterial 删除资料（管理员）
func (s *MaterialService) DeleteMaterial(ctx context.Context, md5 string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 删除资料
		if err := tx.Where("md5 = ?", md5).Delete(&models.Material{}).Error; err != nil {
			return fmt.Errorf("删除资料失败: %w", err)
		}

		// 删除资料描述
		if err := tx.Where("md5 = ?", md5).Delete(&models.MaterialDesc{}).Error; err != nil {
			return fmt.Errorf("删除资料描述失败: %w", err)
		}

		// 删除资料日志
		if err := tx.Where("material_md5 = ?", md5).Delete(&models.MaterialLog{}).Error; err != nil {
			return fmt.Errorf("删除资料日志失败: %w", err)
		}
		return nil
	})
}

// SearchMaterials 搜索资料
func (s *MaterialService) SearchMaterials(ctx context.Context, req *request.MaterialSearchRequest) (*response.MaterialSearchResponse, error) {
	var materials []models.Material
	var total int64

	// 构建搜索查询
	searchPattern := "%" + req.Keywords + "%"
	query := s.db.WithContext(ctx).Model(&models.Material{}).
		Joins("LEFT JOIN material_descs ON materials.md5 = material_descs.md5").
		Joins("LEFT JOIN material_categories ON materials.category_id = material_categories.id").
		Where("materials.file_name LIKE ? OR material_descs.tags LIKE ? OR material_descs.description LIKE ? OR material_categories.name LIKE ?",
			searchPattern, searchPattern, searchPattern, searchPattern)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("获取搜索结果总数失败: %w", err)
	}

	// 分页和排序
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("material_descs.total_hotness DESC").
		Offset(offset).Limit(req.PageSize).
		Preload("Category").Preload("Desc").Find(&materials).Error; err != nil {
		return nil, fmt.Errorf("搜索资料失败: %w", err)
	}

	// 转换为响应格式
	var materialList []response.MaterialListResponse
	for _, material := range materials {
		// 使用预加载的资料描述数据
		var desc models.MaterialDesc
		if material.Desc != nil {
			desc = *material.Desc
		}

		item := response.MaterialListResponse{
			ID:            material.ID,
			MD5:           material.MD5,
			FileName:      material.FileName,
			FileSize:      material.FileSize,
			CategoryID:    material.CategoryID,
			Tags:          desc.Tags,
			TotalHotness:  desc.TotalHotness,
			PeriodHotness: desc.PeriodHotness,
			DownloadCount: desc.DownloadCount,
			CreatedAt:     material.CreatedAt,
		}
		materialList = append(materialList, item)
	}

	return &response.MaterialSearchResponse{
		Materials: materialList,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
		Keywords:  req.Keywords,
	}, nil
}

// GetTopMaterials 获取TOP热门资料
func (s *MaterialService) GetTopMaterials(ctx context.Context, req *request.TopMaterialsRequest) ([]response.MaterialListResponse, error) {
	var materials []models.Material

	// 根据type参数选择排序字段和过滤条件
	orderBy := "material_descs.total_hotness DESC"
	minHotness := 1 // 最小热度，过滤零热度的资料
	if req.Type == 7 {
		// 期间热度（7天）
		orderBy = "material_descs.period_hotness DESC"
	}

	query := s.db.WithContext(ctx).Joins("LEFT JOIN material_descs ON materials.md5 = material_descs.md5")

	// 过滤零热度的资料
	if req.Type == 7 {
		query = query.Where("material_descs.period_hotness >= ?", minHotness)
	} else {
		query = query.Where("material_descs.total_hotness >= ?", minHotness)
	}

	if err := query.Order(orderBy).
		Limit(req.Limit).
		Preload("Category").Preload("Desc").Find(&materials).Error; err != nil {
		return nil, fmt.Errorf("获取热门资料失败: %w", err)
	}

	// 转换为响应格式
	var result []response.MaterialListResponse
	for _, material := range materials {
		// 使用预加载的资料描述数据
		var desc models.MaterialDesc
		if material.Desc != nil {
			desc = *material.Desc
		}

		item := response.MaterialListResponse{
			ID:            material.ID,
			MD5:           material.MD5,
			FileName:      material.FileName,
			FileSize:      material.FileSize,
			CategoryID:    material.CategoryID,
			Tags:          desc.Tags,
			TotalHotness:  desc.TotalHotness,
			PeriodHotness: desc.PeriodHotness,
			DownloadCount: desc.DownloadCount,
			CreatedAt:     material.CreatedAt,
		}
		result = append(result, item)
	}

	return result, nil
}

// ==================== 资料描述管理 ====================

// UpdateMaterialDesc 更新资料描述（管理员，不存在时自动创建）
func (s *MaterialService) UpdateMaterialDesc(ctx context.Context, md5 string, req *request.MaterialDescUpdateRequest) error {
	// 检查记录是否存在
	var existing models.MaterialDesc
	err := s.db.WithContext(ctx).Model(&models.MaterialDesc{}).Where("md5 = ?", md5).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 记录不存在，创建新记录
		desc := models.MaterialDesc{
			MD5:       md5,
			UpdatedAt: time.Now(),
		}

		// 设置字段值
		if req.Tags != nil {
			desc.Tags = *req.Tags
		}
		if req.Description != nil {
			desc.Description = *req.Description
		}
		if req.ExternalLink != nil {
			desc.ExternalLink = *req.ExternalLink
		}
		if req.IsRecommended != nil {
			desc.IsRecommended = *req.IsRecommended
		}

		if err := s.db.WithContext(ctx).Create(&desc).Error; err != nil {
			return fmt.Errorf("创建资料描述失败: %w", err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("查询资料描述失败: %w", err)
	}

	// 记录存在，更新记录
	updates := make(map[string]interface{})

	if req.Tags != nil {
		updates["tags"] = *req.Tags
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.ExternalLink != nil {
		updates["external_link"] = *req.ExternalLink
	}
	if req.IsRecommended != nil {
		updates["is_recommended"] = *req.IsRecommended
	}
	updates["updated_at"] = time.Now()

	if err := s.db.WithContext(ctx).Model(&models.MaterialDesc{}).Where("md5 = ?", md5).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新资料描述失败: %w", err)
	}

	return nil
}

// GetMaterialDesc 获取资料描述
func (s *MaterialService) GetMaterialDesc(ctx context.Context, md5 string) (*response.MaterialDescResponse, error) {
	var desc models.MaterialDesc
	if err := s.db.WithContext(ctx).Model(&models.MaterialDesc{}).Where("md5 = ?", md5).First(&desc).Error; err != nil {
		return nil, fmt.Errorf("资料描述不存在: %w", err)
	}

	return &response.MaterialDescResponse{
		MD5:           desc.MD5,
		Tags:          desc.Tags,
		Description:   desc.Description,
		ExternalLink:  desc.ExternalLink,
		TotalHotness:  desc.TotalHotness,
		PeriodHotness: desc.PeriodHotness,
		ViewCount:     desc.ViewCount,
		DownloadCount: desc.DownloadCount,
		Rating:        desc.Rating,
		RatingCount:   desc.RatingCount,
		IsRecommended: desc.IsRecommended,
		UpdatedAt:     desc.UpdatedAt,
	}, nil
}

// ==================== 分类管理 ====================

// GetCategoriesByParent 通过上级分类ID获取分类（只获取第一层子分类）
func (s *MaterialService) GetCategoriesByParent(ctx context.Context, parentID *uint) ([]response.MaterialCategoryResponse, error) {
	var categories []models.MaterialCategory

	query := s.db.WithContext(ctx).Order("sort ASC, name ASC")
	if parentID != nil {
		query = query.Where("parent_id = ?", *parentID)
	} else {
		query = query.Where("parent_id = 0")
	}

	if err := query.Find(&categories).Error; err != nil {
		return nil, fmt.Errorf("获取分类失败: %w", err)
	}

	// 批量查询所有分类的资料数量
	var categoryIDs []uint
	for _, category := range categories {
		categoryIDs = append(categoryIDs, category.ID)
	}

	// 批量统计资料数量
	var countResults []struct {
		CategoryID    uint  `gorm:"column:category_id"`
		MaterialCount int64 `gorm:"column:material_count"`
	}
	if len(categoryIDs) > 0 {
		s.db.WithContext(ctx).Model(&models.Material{}).
			Select("category_id, COUNT(*) as material_count").
			Where("category_id IN ?", categoryIDs).
			Group("category_id").
			Scan(&countResults)
	}

	// 创建映射
	countMap := make(map[uint]int64)
	for _, result := range countResults {
		countMap[result.CategoryID] = result.MaterialCount
	}

	// 转换为响应格式
	var result []response.MaterialCategoryResponse
	for _, category := range categories {
		materialCount := countMap[category.ID] // 默认为0

		item := response.MaterialCategoryResponse{
			ID:            category.ID,
			Name:          category.Name,
			ParentID:      category.ParentID,
			Level:         category.Level,
			Sort:          category.Sort,
			CreatedAt:     category.CreatedAt,
			MaterialCount: int(materialCount),
		}

		result = append(result, item)
	}

	return result, nil
}

// ==================== 资料日志管理 ====================

// CreateMaterialLog 创建资料日志
func (s *MaterialService) CreateMaterialLog(ctx context.Context, userID uint, req *request.MaterialLogCreateRequest) error {
	// 对于查看、下载和搜索行为，如果同一用户的相同记录已存在，则增加count
	if req.Type == int(models.MaterialLogTypeView) || req.Type == int(models.MaterialLogTypeDownload) {
		// 查看和下载：按用户ID + 资料MD5 + 类型查找
		var existingLog models.MaterialLog
		err := s.db.WithContext(ctx).Where("user_id = ? AND material_md5 = ? AND type = ?",
			userID, req.MaterialMD5, req.Type).First(&existingLog).Error

		if err == nil {
			// 记录已存在，增加count
			if err := s.db.WithContext(ctx).Model(&existingLog).UpdateColumn("count", gorm.Expr("count + 1")).Error; err != nil {
				return fmt.Errorf("更新日志计数失败: %w", err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 记录不存在，创建新记录，初始count为1
			log := models.MaterialLog{
				UserID:      userID,
				Type:        models.MaterialLogType(req.Type),
				MaterialMD5: req.MaterialMD5,
				Count:       1,
				CreatedAt:   time.Now(),
			}
			if err := s.db.WithContext(ctx).Create(&log).Error; err != nil {
				return fmt.Errorf("创建日志失败: %w", err)
			}
		} else {
			return fmt.Errorf("查询日志失败: %w", err)
		}
	} else if req.Type == int(models.MaterialLogTypeSearch) {
		// 搜索：按用户ID + 关键词 + 类型查找
		var existingLog models.MaterialLog
		err := s.db.WithContext(ctx).Where("user_id = ? AND keywords = ? AND type = ?",
			userID, req.Keywords, req.Type).First(&existingLog).Error

		if err == nil {
			// 记录已存在，增加count
			if err := s.db.WithContext(ctx).Model(&existingLog).UpdateColumn("count", gorm.Expr("count + 1")).Error; err != nil {
				return fmt.Errorf("更新日志计数失败: %w", err)
			}
		} else if err == gorm.ErrRecordNotFound {
			// 记录不存在，创建新记录，初始count为1
			log := models.MaterialLog{
				UserID:    userID,
				Type:      models.MaterialLogType(req.Type),
				Keywords:  req.Keywords,
				Count:     1,
				CreatedAt: time.Now(),
			}
			if err := s.db.WithContext(ctx).Create(&log).Error; err != nil {
				return fmt.Errorf("创建日志失败: %w", err)
			}
		} else {
			return fmt.Errorf("查询日志失败: %w", err)
		}
	} else {
		// 评分行为，每次都创建新记录（或在RateMaterial中处理更新）
		log := models.MaterialLog{
			UserID:      userID,
			Type:        models.MaterialLogType(req.Type),
			MaterialMD5: req.MaterialMD5,
			Rating:      req.Rating,
			Count:       1,
			CreatedAt:   time.Now(),
		}

		if err := s.db.WithContext(ctx).Create(&log).Error; err != nil {
			return fmt.Errorf("创建日志失败: %w", err)
		}
	}

	// 如果是评分，更新资料描述的评分统计
	if req.Type == int(models.MaterialLogTypeRating) && req.Rating != nil && req.MaterialMD5 != "" {
		if err := s.updateMaterialRating(ctx, req.MaterialMD5, *req.Rating); err != nil {
			// 评分统计更新失败不影响日志创建
			fmt.Printf("更新评分统计失败: %v", err)
		}
	}

	// 如果是查看或下载，增加相应计数（不存在则创建）
	if req.MaterialMD5 != "" {
		if req.Type == int(models.MaterialLogTypeView) {
			s.ensureDescExists(ctx, req.MaterialMD5)
			s.db.WithContext(ctx).Model(&models.MaterialDesc{}).Where("md5 = ?", req.MaterialMD5).
				UpdateColumn("view_count", gorm.Expr("view_count + 1"))
		} else if req.Type == int(models.MaterialLogTypeDownload) {
			s.ensureDescExists(ctx, req.MaterialMD5)
			s.db.WithContext(ctx).Model(&models.MaterialDesc{}).Where("md5 = ?", req.MaterialMD5).
				UpdateColumn("download_count", gorm.Expr("download_count + 1"))
		}
	}

	return nil
}

// RateMaterial 为资料评分
func (s *MaterialService) RateMaterial(ctx context.Context, userID uint, md5 string, rating int) error {
	// 检查用户是否已经对该资料评过分
	var existingLog models.MaterialLog
	err := s.db.WithContext(ctx).Model(&models.MaterialLog{}).Where("user_id = ? AND material_md5 = ? AND type = ?",
		userID, md5, models.MaterialLogTypeRating).First(&existingLog).Error

	if err == nil {
		// 已评分，更新评分
		existingLog.Rating = &rating
		if err := s.db.WithContext(ctx).Save(&existingLog).Error; err != nil {
			return fmt.Errorf("更新评分失败: %w", err)
		}
	} else if err == gorm.ErrRecordNotFound {
		// 未评分，创建新评分
		req := &request.MaterialLogCreateRequest{
			Type:        int(models.MaterialLogTypeRating),
			MaterialMD5: md5,
			Rating:      &rating,
		}
		return s.CreateMaterialLog(ctx, userID, req)
	} else {
		return fmt.Errorf("查询评分记录失败: %w", err)
	}

	// 重新计算平均评分
	return s.updateMaterialRating(ctx, md5, rating)
}

// GetHotWords 获取搜索热词
func (s *MaterialService) GetHotWords(ctx context.Context, limit int) ([]response.HotWordsResponse, error) {
	var results []struct {
		Keywords  string
		UserCount int
		Count     int
	}

	// 统计搜索关键词，只返回超过50人搜索的词
	minUserCount := 50
	if err := s.db.WithContext(ctx).Model(&models.MaterialLog{}).
		Select("keywords, COUNT(DISTINCT user_id) as user_count, SUM(count) as count").
		Where("type = ? AND keywords != ''", models.MaterialLogTypeSearch).
		Group("keywords").
		Having("COUNT(DISTINCT user_id) >= ?", minUserCount).
		Order("count DESC").
		Limit(limit).
		Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("获取热词失败: %w", err)
	}

	var hotWords []response.HotWordsResponse
	for _, result := range results {
		hotWords = append(hotWords, response.HotWordsResponse{
			Keywords: result.Keywords,
			Count:    result.Count,
		})
	}

	return hotWords, nil
}

// ==================== 定时任务相关 ====================

// CalculateHotness 计算热度（定时任务调用）
func (s *MaterialService) CalculateHotness(ctx context.Context) error {
	// 计算最近7天的热度
	sevenDaysAgo := time.Now().AddDate(0, 0, -7)

	var periodResults []struct {
		MaterialMD5   string
		ViewCount     int
		DownloadCount int
		RatingCount   int
	}

	// 统计最近7天的活动，使用count字段累加行为次数
	if err := s.db.WithContext(ctx).Model(&models.MaterialLog{}).
		Select("material_md5, "+
			"SUM(CASE WHEN type = ? THEN count ELSE 0 END) as view_count, "+
			"SUM(CASE WHEN type = ? THEN count ELSE 0 END) as download_count, "+
			"SUM(CASE WHEN type = ? THEN count ELSE 0 END) as rating_count",
			models.MaterialLogTypeView, models.MaterialLogTypeDownload, models.MaterialLogTypeRating).
		Where("created_at >= ? AND material_md5 != ''", sevenDaysAgo).
		Group("material_md5").
		Scan(&periodResults).Error; err != nil {
		return fmt.Errorf("统计期间热度数据失败: %w", err)
	}

	// 统计所有时间的活动（总热度）
	var totalResults []struct {
		MaterialMD5   string
		ViewCount     int
		DownloadCount int
		RatingCount   int
	}

	if err := s.db.WithContext(ctx).Model(&models.MaterialLog{}).
		Select("material_md5, "+
			"SUM(CASE WHEN type = ? THEN count ELSE 0 END) as view_count, "+
			"SUM(CASE WHEN type = ? THEN count ELSE 0 END) as download_count, "+
			"SUM(CASE WHEN type = ? THEN count ELSE 0 END) as rating_count",
			models.MaterialLogTypeView, models.MaterialLogTypeDownload, models.MaterialLogTypeRating).
		Where("material_md5 != ''").
		Group("material_md5").
		Scan(&totalResults).Error; err != nil {
		return fmt.Errorf("统计总热度数据失败: %w", err)
	}

	// 构建期间热度映射
	periodHotnessMap := make(map[string]int)
	for _, result := range periodResults {
		// 热度计算公式：查看次数 + 下载次数*3 + 评分次数*2
		periodHotness := result.ViewCount + result.DownloadCount*3 + result.RatingCount*2
		periodHotnessMap[result.MaterialMD5] = periodHotness
	}

	// 更新每个资料的总热度和期间热度
	for _, result := range totalResults {
		// 计算总热度（基于所有历史数据）
		totalHotness := result.ViewCount + result.DownloadCount*3 + result.RatingCount*2

		// 获取期间热度（如果有的话）
		periodHotness := periodHotnessMap[result.MaterialMD5]

		// 更新热度数据
		if err := s.db.WithContext(ctx).Model(&models.MaterialDesc{}).Where("md5 = ?", result.MaterialMD5).
			Updates(map[string]interface{}{
				"period_hotness": periodHotness,
				"total_hotness":  totalHotness,
				"updated_at":     time.Now(),
			}).Error; err != nil {
			fmt.Printf("更新资料热度失败 MD5=%s: %v\n", result.MaterialMD5, err)
		}
	}

	return nil
}

// ==================== 私有辅助方法 ====================

// buildCategoryPath 构建分类路径字符串
func (s *MaterialService) buildCategoryPath(ctx context.Context, categoryID uint) string {
	path := s.getCategoryPath(ctx, categoryID)
	var names []string
	for _, category := range path {
		names = append(names, category.Name)
	}
	return strings.Join(names, " > ")
}

// getCategoryPath 获取分类路径数组
func (s *MaterialService) getCategoryPath(ctx context.Context, categoryID uint) []response.MaterialCategoryResponse {
	var path []response.MaterialCategoryResponse
	var currentID uint = categoryID

	for currentID > 0 {
		var category models.MaterialCategory
		if err := s.db.WithContext(ctx).First(&category, currentID).Error; err != nil {
			break
		}

		pathItem := response.MaterialCategoryResponse{
			ID:        category.ID,
			Name:      category.Name,
			ParentID:  category.ParentID,
			Level:     category.Level,
			Sort:      category.Sort,
			CreatedAt: category.CreatedAt,
		}

		path = append([]response.MaterialCategoryResponse{pathItem}, path...)
		currentID = category.ParentID
	}

	return path
}

// updateMaterialRating 更新资料评分统计
func (s *MaterialService) updateMaterialRating(ctx context.Context, md5 string, newRating int) error {
	// 确保描述记录存在
	s.ensureDescExists(ctx, md5)

	// 计算平均评分
	var result struct {
		AvgRating float64
		Count     int64
	}

	if err := s.db.WithContext(ctx).Model(&models.MaterialLog{}).
		Select("AVG(CAST(rating AS DECIMAL(3,2))) as avg_rating, COUNT(*) as count").
		Where("material_md5 = ? AND type = ? AND rating IS NOT NULL", md5, models.MaterialLogTypeRating).
		Scan(&result).Error; err != nil {
		return fmt.Errorf("计算平均评分失败: %w", err)
	}

	// 更新资料描述的评分信息
	return s.db.WithContext(ctx).Model(&models.MaterialDesc{}).Where("md5 = ?", md5).Updates(map[string]interface{}{
		"rating":       result.AvgRating,
		"rating_count": result.Count,
		"updated_at":   time.Now(),
	}).Error
}

// ensureDescExists 确保资料描述记录存在，不存在则创建
func (s *MaterialService) ensureDescExists(ctx context.Context, md5 string) {
	var existing models.MaterialDesc
	err := s.db.WithContext(ctx).Model(&models.MaterialDesc{}).Where("md5 = ?", md5).First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 记录不存在，创建默认记录
		desc := models.MaterialDesc{
			MD5:       md5,
			UpdatedAt: time.Now(),
		}
		s.db.WithContext(ctx).Create(&desc)
	}
}
