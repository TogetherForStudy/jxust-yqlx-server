package request

// MaterialListRequest 资料列表查询请求
type MaterialListRequest struct {
	CategoryID *uint  `form:"category_id" json:"category_id"`                                // 分类ID
	Page       int    `form:"page" json:"page" binding:"min=1"`                              // 页码
	PageSize   int    `form:"page_size" json:"page_size" binding:"min=1,max=100"`            // 每页数量
	SortBy     string `form:"sort_by" json:"sort_by" binding:"omitempty,oneof=hotness time"` // 排序方式: hotness(热度), time(时间)
}

// MaterialSearchRequest 资料搜索请求
type MaterialSearchRequest struct {
	Keywords string `form:"keywords" json:"keywords" binding:"required,min=1,max=100"` // 搜索关键词
	Page     int    `form:"page" json:"page" binding:"min=1"`                          // 页码
	PageSize int    `form:"page_size" json:"page_size" binding:"min=1,max=100"`        // 每页数量
}

// MaterialDescCreateRequest 创建资料描述请求
type MaterialDescCreateRequest struct {
	MD5           string `json:"md5" binding:"required,len=32"`    // 文件MD5
	Tags          string `json:"tags" binding:"max=500"`           // 标签
	Description   string `json:"description" binding:"max=5000"`   // 描述
	ExternalLink  string `json:"external_link" binding:"max=1000"` // 外部链接
	IsRecommended bool   `json:"is_recommended"`                   // 人工推荐
}

// MaterialDescUpdateRequest 更新资料描述请求
type MaterialDescUpdateRequest struct {
	Tags          *string `json:"tags" binding:"omitempty,max=500"`           // 标签
	Description   *string `json:"description" binding:"omitempty,max=5000"`   // 描述
	ExternalLink  *string `json:"external_link" binding:"omitempty,max=1000"` // 外部链接
	IsRecommended *bool   `json:"is_recommended"`                             // 人工推荐
}

// MaterialLogCreateRequest 创建资料日志请求
type MaterialLogCreateRequest struct {
	Type        int    `json:"type" binding:"required,min=1,max=4"`    // 记录类型：1=搜索，2=查看，3=评分，4=下载
	Keywords    string `json:"keywords" binding:"max=200"`             // 搜索关键词
	MaterialMD5 string `json:"material_md5" binding:"max=32"`          // 资料MD5
	Rating      *int   `json:"rating" binding:"omitempty,min=1,max=5"` // 评分(1-5)
}

// MaterialCategoryCreateRequest 创建分类请求
type MaterialCategoryCreateRequest struct {
	Name     string `json:"name" binding:"required,min=1,max=50"` // 分类名称
	ParentID uint   `json:"parent_id"`                            // 上级分类ID
	Sort     int    `json:"sort"`                                 // 排序
}

// MaterialCategoryUpdateRequest 更新分类请求
type MaterialCategoryUpdateRequest struct {
	Name *string `json:"name" binding:"omitempty,min=1,max=50"` // 分类名称
	Sort *int    `json:"sort"`                                  // 排序
}

// MaterialCategoryListRequest 分类列表查询请求
type MaterialCategoryListRequest struct {
	ParentID *uint `form:"parent_id" json:"parent_id"` // 上级分类ID
}

// MaterialRatingRequest 资料评分请求
type MaterialRatingRequest struct {
	Rating int `json:"rating" binding:"required,min=1,max=5"` // 评分(1-5)
}

// HotWordsRequest 热词查询请求
type HotWordsRequest struct {
	Limit int `form:"limit" json:"limit" binding:"min=1,max=50"` // 数量限制
}

// TopMaterialsRequest TOP热门资料请求
type TopMaterialsRequest struct {
	Limit int `form:"limit" json:"limit" binding:"min=1,max=50"`      // 数量限制，默认10
	Type  int `form:"type" json:"type" binding:"omitempty,oneof=0 7"` // 热度类型：0=全部热度，7=期间热度(7天)，默认0
}
