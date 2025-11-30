package constant

// Envs

// common environment variables used in the application
const (
	ENV_RUNNING_LEVEL = "ENV_RUNNING_LEVEL" // 环境变量：运行环境级别
	ENV_LogFormatStr  = "ENV_LOG_FORMAT"

	ENV_GIN_MODE = "GIN_MODE" // Gin 模式

	ENV_MINIO_ENDPOINT   = "MINIO_ENDPOINT"
	ENV_MINIO_ACCESS_KEY = "MINIO_ACCESS_KEY"
	ENV_MINIO_SECRET_KEY = "MINIO_SECRET_KEY"
	ENV_MINIO_USE_SSL    = "MINIO_USE_SSL" // "true" or "false"
)
