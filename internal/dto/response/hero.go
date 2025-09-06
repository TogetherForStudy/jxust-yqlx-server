package response

type HeroItem struct {
	ID        uint   `json:"id"`
	Name      string `json:"name"`
	Sort      int    `json:"sort"`
	IsShow    bool   `json:"isshow"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type HeroListResponse struct {
	List []HeroItem `json:"list"`
}
