package request

// OSSGetTokenRequest 请求生成 CDN/OSS 访问的签名
// 参考又拍云 Token 防盗链: https://help.upyun.com/knowledge-base/cdn-token-limite/
type OSSGetTokenRequest struct {
	// URI 必须是资源路径部分，以 "/" 开头，不包含查询参数
	URI string `json:"uri" binding:"required"`
	// ExpireSeconds 自定义有效期(秒)，可选。为空则由服务端设默认
	ExpireSeconds int64 `json:"expire_seconds"`
}
