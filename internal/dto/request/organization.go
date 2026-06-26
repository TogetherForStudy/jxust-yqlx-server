package request

// ListOrganizationsRequest 组织列表查询请求
type ListOrganizationsRequest struct {
	Query            string `form:"query" json:"query"`
	OrganizationType string `form:"organization_type" json:"organization_type"`
	Affiliation      string `form:"affiliation" json:"affiliation"`
	Campus           string `form:"campus" json:"campus"`
	Page             int    `form:"page" json:"page" binding:"min=1"`
	PageSize         int    `form:"page_size" json:"page_size" binding:"min=1,max=100"`
}

// CreateOrganizationRequest 创建组织请求
type CreateOrganizationRequest struct {
	Name             string `json:"name" binding:"required,max=255"`
	OrganizationType string `json:"organization_type" binding:"required,max=100"`
	Affiliation      string `json:"affiliation" binding:"required,max=255"`
	Campus           string `json:"campus" binding:"required,max=100"`
	Introduction     string `json:"introduction" binding:"required,max=1000"`
	Contact          string `json:"contact" binding:"required,max=255"`
}

// UpdateOrganizationRequest 更新组织请求
type UpdateOrganizationRequest struct {
	Name             *string `json:"name" binding:"omitempty,max=255"`
	OrganizationType *string `json:"organization_type" binding:"omitempty,max=100"`
	Affiliation      *string `json:"affiliation" binding:"omitempty,max=255"`
	Campus           *string `json:"campus" binding:"omitempty,max=100"`
	Introduction     *string `json:"introduction" binding:"omitempty,max=1000"`
	Contact          *string `json:"contact" binding:"omitempty,max=255"`
}
