package dto

type Response struct {
	StatusCode    int    `json:"StatusCode"`       // 状态码
	StatusMessage string `json:"StatusMessage"`    // 状态信息
	RequestId     string `json:"RequestId"`        // 请求ID
	Result        any    `json:"Result,omitempty"` // 结果数据
}
