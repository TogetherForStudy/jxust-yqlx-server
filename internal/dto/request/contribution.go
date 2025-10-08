package request

// CreateContributionRequest 创建投稿请求
type CreateContributionRequest struct {
	Title      string `json:"title" binding:"required,max=200"`         // 投稿标题
	Content    string `json:"content" binding:"required"`               // 投稿内容
	Categories []int  `json:"categories" binding:"required,min=1,dive"` // 建议分类ID数组
}

// GetContributionsRequest 获取投稿列表请求
type GetContributionsRequest struct {
	Page   int    `form:"page" binding:"min=1"`         // 页码
	Size   int    `form:"size" binding:"min=1,max=100"` // 每页数量
	Status *uint8 `form:"status" binding:"omitempty"`   // 状态过滤：1=待审核，2=已采纳，3=已拒绝
	UserID *uint  `form:"user_id" binding:"omitempty"`  // 用户ID过滤（管理员用）
}

// ReviewContributionRequest 审核投稿请求
type ReviewContributionRequest struct {
	Status     uint8  `json:"status" binding:"required,oneof=2 3"`      // 审核结果：2=采纳，3=拒绝
	ReviewNote string `json:"review_note" binding:"max=500"`            // 审核备注
	Points     uint   `json:"points" binding:"omitempty,min=1,max=100"` // 奖励积分（采纳时）
	Title      string `json:"title" binding:"omitempty,max=200"`        // 修改后的标题（采纳时）
	Content    string `json:"content"`                                  // 修改后的内容（采纳时）
	Categories []int  `json:"categories" binding:"omitempty,dive"`      // 修改后的分类（采纳时）
}
