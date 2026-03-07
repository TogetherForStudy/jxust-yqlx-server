package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
	"gopkg.in/yaml.v3"
)

const configFileName = "yqlx-config.yaml"

type Runtime struct {
	RunningLevel string `yaml:"running_level" env:"ENV_RUNNING_LEVEL" envDefault:"info"`
	LogFormat    string `yaml:"log_format" env:"ENV_LOG_FORMAT" envDefault:""`
	GinMode      string `yaml:"gin_mode" env:"GIN_MODE" envDefault:"debug"`
}

type Config struct {
	Runtime `yaml:",inline"`

	Database `yaml:"database"`
	Redis    `yaml:"redis"`
	MinIO    `yaml:"minio"`
	LLM      `yaml:"llm"`

	JWTSecret          string        `yaml:"jwt_secret" env:"JWT_SECRET"`
	RefreshTokenSecret string        `yaml:"refresh_token_secret" env:"REFRESH_TOKEN_SECRET" envDefault:""`
	AccessTokenTTL     time.Duration `yaml:"access_token_ttl" env:"ACCESS_TOKEN_TTL" envDefault:"2h"`
	RefreshTokenTTL    time.Duration `yaml:"refresh_token_ttl" env:"REFRESH_TOKEN_TTL" envDefault:"720h"`
	ServerPort         string        `yaml:"server_port" env:"SERVER_PORT" envDefault:"8085"`
	WechatAppID        string        `yaml:"wechat_app_id" env:"WECHAT_APP_ID"`
	WechatAppSecret    string        `yaml:"wechat_app_secret" env:"WECHAT_APP_SECRET" envDefault:""`
	InitRbac           bool          `yaml:"init_rbac" env:"INIT_RBAC" envDefault:"false"`

	// Upyun/CDN Token 防盗链配置
	UpyunTokenSecret string `yaml:"upyun_token_secret" env:"UPYUN_TOKEN_SECRET" envDefault:""`
	CdnBaseURL       string `yaml:"cdn_base_url" env:"CDN_BASE_URL" envDefault:""`

	// Tencent CLS 配置
	ClsEnable    bool   `yaml:"cls_enable" env:"CLS_ENABLE" envDefault:"false"`
	ClsSecretID  string `yaml:"cls_secret_id" env:"CLS_SECRET_ID"`
	ClsSecretKey string `yaml:"cls_secret_key" env:"CLS_SECRET_KEY"`
	ClsEndpoint  string `yaml:"cls_endpoint" env:"CLS_ENDPOINT" envDefault:"ap-guangzhou.cls.tencentcs.com"`
	ClsTopicID   string `yaml:"cls_topic_id" env:"CLS_TOPIC_ID"`

	// For minio signature and correct reverse proxy configuration
	Host   string `yaml:"host" env:"HOST" envDefault:"localhost:8085"` // The port is usually the same as the ServerPort.
	Scheme string `yaml:"scheme" env:"SCHEME" envDefault:"http"`
}

// GlobalConfig is a singleton instance of Config that can be accessed globally.
var GlobalConfig = sync.OnceValue[*Config](func() *Config {
	cfg := &Config{}
	if err := load(cfg); err != nil {
		panic(err.Error())
	}
	return cfg
})

// GlobalRuntime is a singleton instance of Runtime used by low-level packages that
// need process settings without requiring the full application config.
var GlobalRuntime = sync.OnceValue[*Runtime](func() *Runtime {
	cfg := &Runtime{}
	if err := load(cfg); err != nil {
		panic(err.Error())
	}
	return cfg
})

type Database struct {
	DBHost     string `yaml:"db_host" env:"DB_HOST" envDefault:"localhost"`
	DBPort     string `yaml:"db_port" env:"DB_PORT" envDefault:"3306"`
	DBUsername string `yaml:"db_username" env:"DB_USERNAME" envDefault:"root"`
	DBPassword string `yaml:"db_password" env:"DB_PASSWORD" envDefault:""`
	DBName     string `yaml:"db_name" env:"DB_NAME" envDefault:"gojxust"`
}
type Redis struct {
	RedisHost     string `yaml:"redis_host" env:"REDIS_HOST" envDefault:"localhost"`
	RedisPort     int    `yaml:"redis_port" env:"REDIS_PORT" envDefault:"6379"`
	RedisPassword string `yaml:"redis_password" env:"REDIS_PASSWORD" envDefault:""`
	RedisDB       int    `yaml:"redis_db" env:"REDIS_DB" envDefault:"0"`
}
type MinIO struct {
	MinIOEndpoint  string `yaml:"minio_endpoint" env:"MINIO_ENDPOINT" envDefault:"localhost:9000"`
	MinIOAccessKey string `yaml:"minio_access_key" env:"MINIO_ACCESS_KEY" envDefault:"minioadmin"`
	MinIOSecretKey string `yaml:"minio_secret_key" env:"MINIO_SECRET_KEY" envDefault:"minioadmin"`
	MinIOUseSSL    bool   `yaml:"minio_use_ssl" env:"MINIO_USE_SSL" envDefault:"false"`
	BucketName     string `yaml:"bucket_name" env:"BUCKET_NAME" envDefault:"yqlx"`
}

type LLM struct {
	RAGFlowMCPURL string `yaml:"ragflow_mcp_url" env:"RAGFLOW_MCP_URL" envDefault:""` // e.g., "http://localhost:8080/mcp/sse"
	RAGFlowAPIKey string `yaml:"ragflow_api_key" env:"RAGFLOW_API_KEY" envDefault:""`
	Model         string `yaml:"llm_model" env:"LLM_MODEL" envDefault:"gpt-4"`
	APIKey        string `yaml:"llm_api_key" env:"LLM_API_KEY" envDefault:""`
	BaseURL       string `yaml:"llm_base_url" env:"LLM_BASE_URL" envDefault:""`
}

// NewConfig initializes and return the configuration by reading environment variables.
//
//	If the configuration has already been initialized, it returns the existing instance.
var NewConfig = GlobalConfig

// NewRuntime initializes and returns runtime-only configuration.
var NewRuntime = GlobalRuntime

var loadDotEnvOnce sync.Once

func load[T any](cfg *T) error {
	loadDotEnvOnce.Do(func() {
		_ = godotenv.Load()
	})

	info, err := os.Stat(configFileName)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("config path %q is a directory", configFileName)
		}
		file, err := os.Open(configFileName)
		if err != nil {
			return fmt.Errorf("failed to open config file: %w", err)
		}
		defer file.Close()
		if err := yaml.NewDecoder(file).Decode(cfg); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("failed to stat config file: %w", err)
	}
	if err := env.Parse(cfg); err != nil {
		return fmt.Errorf("failed to parse environment variables: %w", err)
	}
	return nil
}
