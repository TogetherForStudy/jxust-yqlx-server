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
	Sort   int    `json:"sort"`
	IsShow bool   `json:"is_show"`
}
