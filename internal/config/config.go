package config

import (
	"os"
	"sync"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Database `yaml:"database"`
	Redis    `yaml:"redis"`
	MinIO    `yaml:"minio"`

	JWTSecret       string `yaml:"jwt_secret" env:"JWT_SECRET"`
	ServerPort      string `yaml:"server_port" env:"SERVER_PORT" envDefault:"8085"`
	WechatAppID     string `yaml:"wechat_app_id" env:"WECHAT_APP_ID"`
	WechatAppSecret string `yaml:"wechat_app_secret" env:"WECHAT_APP_SECRET" envDefault:""`

	Domain string `yaml:"domain" env:"DOMAIN" envDefault:"http://localhost:8085"`
}

// GlobalConfig is a singleton instance of Config that can be accessed globally.
var GlobalConfig *Config

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
}

var _once sync.Once

// NewConfig initializes and return the configuration by reading environment variables.
//
//	If the configuration has already been initialized, it returns the existing instance.
func NewConfig() *Config {
	_once.Do(func() {
		var cfg Config
		if err := env.Parse(&cfg); err != nil {
			println("Failed to parse environment variables: ", err)
			os.Exit(1)
		}
		GlobalConfig = &cfg
	})

	return GlobalConfig
}
