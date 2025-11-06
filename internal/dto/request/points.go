package request

// GetPointsTransactionsRequest 获取积分交易记录请求
type GetPointsTransactionsRequest struct {
	Page   int    `form:"page" binding:"min=1"`         // 页码
	Size   int    `form:"size" binding:"min=1,max=100"` // 每页数量
	Type   *uint8 `form:"type" binding:"omitempty"`     // 交易类型：1=获得，2=消耗
	Source *uint8 `form:"source" binding:"omitempty"`   // 交易来源：1=投稿采纳，2=兑换奖品
	UserID *uint  `form:"user_id" binding:"omitempty"`  // 用户ID（管理员查看）
}

// SpendPointsRequest 积分消费请求
type SpendPointsRequest struct {
	Points      uint   `json:"points" binding:"required,min=1"`        // 消费积分数量
	Description string `json:"description" binding:"required,max=200"` // 消费描述
}
