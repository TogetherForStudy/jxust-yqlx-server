package response

type OSSGetTokenResponse struct {
	// TokenParam 即 _upt 参数值，如: abcdefgh1700000600
	TokenParam string `json:"token_param"`
	// ExpireAt 过期时间 Unix 秒
	ExpireAt int64 `json:"expire_at"`
	// SignedURL 如果配置了 CDN 基础域名则给出完整签名 URL，否则为空
	SignedURL string `json:"signed_url"`
}
