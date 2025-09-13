package response

type UploadFileResponse struct {
	ResourceID string `json:"resource_id"`
}

type GetFileURLResponse struct {
	URL string `json:"url"`
}
