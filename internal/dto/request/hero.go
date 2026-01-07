package request

// CreateHeroRequest 创建 Hero 请求
type CreateHeroRequest struct {
	Name   string `json:"name" binding:"required"`
	Sort   int    `json:"sort"`
	IsShow bool   `json:"is_show"`
}

// UpdateHeroRequest 更新 Hero 请求
type UpdateHeroRequest struct {
	Name   string `json:"name" binding:"required"`
	Sort   *int   `json:"sort"`
	IsShow *bool  `json:"is_show"`
}

// SearchHeroRequest 搜索英雄请求
type SearchHeroRequest struct {
	Query  string `form:"q" json:"q"`                  // 搜索关键词
	IsShow *bool  `form:"is_show" json:"is_show"`      // 是否显示过滤（可选）
	Page   int    `form:"page,default=1" json:"page"`  // 页码
	Size   int    `form:"size,default=10" json:"size"` // 每页数量
}
