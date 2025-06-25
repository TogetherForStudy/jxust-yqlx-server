package config

import (
	"os"
	"sync"

	"github.com/caarlos0/env/v11"
)

type Config struct {
	Database `yaml:"database"`

	JWTSecret       string `yaml:"jwt_secret" env:"JWT_SECRET"`
	ServerPort      string `yaml:"server_port" env:"SERVER_PORT" envDefault:"8085"`
	WechatAppID     string `yaml:"wechat_app_id" env:"WECHAT_APP_ID"`
	WechatAppSecret string `yaml:"wechat_app_secret" env:"WECHAT_APP_SECRET" envDefault:""`
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
