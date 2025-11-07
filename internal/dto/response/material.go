package response

import (
	"time"

	"github.com/TogetherForStudy/jxust-yqlx-server/internal/models"
)

// MaterialListResponse 资料列表响应（仅包含Material表字段 + tags）
type MaterialListResponse struct {
	ID            uint      `json:"id"`
	MD5           string    `json:"md5"`
	FileName      string    `json:"file_name"`
	FileSize      int64     `json:"file_size"`
	CategoryID    uint      `json:"category_id"`
	Tags          string    `json:"tags"`           // 来自MaterialDesc表
	TotalHotness  int       `json:"total_hotness"`  // 总热度
	PeriodHotness int       `json:"period_hotness"` // 期间热度
	DownloadCount int       `json:"download_count"` // 下载次数
	CreatedAt     time.Time `json:"created_at"`
}

// MaterialDetailResponse 资料详情响应（包含Material表所有字段 + MaterialDesc所有字段）
type MaterialDetailResponse struct {
	ID            uint      `json:"id"`
	MD5           string    `json:"md5"`
	FileName      string    `json:"file_name"`
	FileSize      int64     `json:"file_size"`
	CategoryID    uint      `json:"category_id"`
	CategoryName  string    `json:"category_name"`
	CategoryPath  string    `json:"category_path"` // 完整分类路径
	Tags          string    `json:"tags"`
	Description   string    `json:"description"`
	ExternalLink  string    `json:"external_link"`
	TotalHotness  int       `json:"total_hotness"`
	ViewCount     int       `json:"view_count"`
	DownloadCount int       `json:"download_count"`
	Rating        float64   `json:"rating"`       // 平均评分
	RatingCount   int       `json:"rating_count"` // 评分总数
	UserRating    *int      `json:"user_rating"`  // 当前用户的评分（未评分则为null）
	IsRecommended bool      `json:"is_recommended"`
	CreatedAt     time.Time `json:"created_at"`
}

// MaterialDescResponse 资料描述响应
type MaterialDescResponse struct {
	MD5           string    `json:"md5"`
	Tags          string    `json:"tags"`
	Description   string    `json:"description"`
	ExternalLink  string    `json:"external_link"`
	TotalHotness  int       `json:"total_hotness"`
	PeriodHotness int       `json:"period_hotness"`
	ViewCount     int       `json:"view_count"`
	DownloadCount int       `json:"download_count"`
	Rating        float64   `json:"rating"`
	RatingCount   int       `json:"rating_count"`
	IsRecommended bool      `json:"is_recommended"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// MaterialCategoryResponse 分类响应
type MaterialCategoryResponse struct {
	ID            uint                       `json:"id"`
	Name          string                     `json:"name"`
	ParentID      uint                       `json:"parent_id"`
	Level         int                        `json:"level"`
	Sort          int                        `json:"sort"`
	CreatedAt     time.Time                  `json:"created_at"`
	Children      []MaterialCategoryResponse `json:"children,omitempty"`
	MaterialCount int                        `json:"material_count,omitempty"` // 资料数量
}

// MaterialLogResponse 资料日志响应
type MaterialLogResponse struct {
	ID          uint                   `json:"id"`
	UserID      uint                   `json:"user_id"`
	Type        models.MaterialLogType `json:"type"`
	TypeName    string                 `json:"type_name"` // 类型名称
	Keywords    string                 `json:"keywords"`
	MaterialMD5 string                 `json:"material_md5"`
	Rating      *int                   `json:"rating"`
	CreatedAt   time.Time              `json:"created_at"`
}

// MaterialSearchResponse 资料搜索响应
type MaterialSearchResponse struct {
	Materials []MaterialListResponse `json:"materials"`
	Total     int64                  `json:"total"`
	Page      int                    `json:"page"`
	PageSize  int                    `json:"page_size"`
	Keywords  string                 `json:"keywords"`
}

// HotWordsResponse 热词响应
type HotWordsResponse struct {
	Keywords string `json:"keywords"`
	Count    int    `json:"count"`
}

// TopMaterialsResponse TOP热门资料响应
type TopMaterialsResponse struct {
	Materials []MaterialListResponse `json:"materials"`
}

// MaterialStatsResponse 资料统计响应
type MaterialStatsResponse struct {
	TotalMaterials  int64 `json:"total_materials"`  // 总资料数
	TotalCategories int64 `json:"total_categories"` // 总分类数
	TotalViewCount  int64 `json:"total_view_count"` // 总查看次数
	TotalDownloads  int64 `json:"total_downloads"`  // 总下载次数
	TodayUploads    int64 `json:"today_uploads"`    // 今日上传数
	WeeklyUploads   int64 `json:"weekly_uploads"`   // 本周上传数
	MonthlyUploads  int64 `json:"monthly_uploads"`  // 本月上传数
}
