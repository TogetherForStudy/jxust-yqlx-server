package config

import "os"

type Config struct {
	DBHost          string
	DBPort          string
	DBUsername      string
	DBPassword      string
	DBName          string
	JWTSecret       string
	ServerPort      string
	WechatAppID     string
	WechatAppSecret string
}

func NewConfig() *Config {
	return &Config{
		DBHost:          getEnv("DB_HOST", "localhost"),
		DBPort:          getEnv("DB_PORT", "3306"),
		DBUsername:      getEnv("DB_USERNAME", "root"),
		DBPassword:      getEnv("DB_PASSWORD", ""),
		DBName:          getEnv("DB_NAME", "gojxust"),
		JWTSecret:       getEnv("JWT_SECRET", "default_secret"),
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		WechatAppID:     getEnv("WECHAT_APP_ID", ""),
		WechatAppSecret: getEnv("WECHAT_APP_SECRET", ""),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
