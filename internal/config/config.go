package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	App     AppConfig
	MongoDB MongoDBConfig
	Redis   RedisConfig
	JWT     JWTConfig
	R2      R2Config
	Asynq   AsynqConfig
	Cipher  CipherConfig
	Email   ResendConfig
}

type AppConfig struct {
	Env  string
	Port int
}

type MongoDBConfig struct {
	URI string
	DB  string
}

type RedisConfig struct {
	URL string
}

type JWTConfig struct {
	AccessSecret  string
	RefreshSecret string
	AccessTTL     time.Duration
	RefreshTTL    time.Duration
}

type R2Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
}

type AsynqConfig struct {
	RedisURL    string
	Concurrency int
}

type CipherConfig struct {
	EncryptionKey string
}

type ResendConfig struct {
	ApiKey string
}

func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()

	// Set defaults
	viper.SetDefault("APP_ENV", "development")
	viper.SetDefault("APP_PORT", 8080)
	viper.SetDefault("MONGO_URI", "mongodb://localhost:27017")
	viper.SetDefault("MONGO_DB", "saas_stock")
	viper.SetDefault("REDIS_URL", "redis://localhost:6379")
	viper.SetDefault("JWT_ACCESS_TTL", "15m")
	viper.SetDefault("JWT_REFRESH_TTL", "7d")
	viper.SetDefault("ASYNQ_CONCURRENCY", 10)

	_ = viper.ReadInConfig() // Ignore error if .env doesn't exist

	accessTTL, err := time.ParseDuration(viper.GetString("JWT_ACCESS_TTL"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid JWT_ACCESS_TTL: %w", err)
	}

	refreshTTL, err := time.ParseDuration(viper.GetString("JWT_REFRESH_TTL"))
	if err != nil {
		return nil, fmt.Errorf("config: invalid JWT_REFRESH_TTL: %w", err)
	}

	cfg := &Config{
		App: AppConfig{
			Env:  viper.GetString("APP_ENV"),
			Port: viper.GetInt("APP_PORT"),
		},
		MongoDB: MongoDBConfig{
			URI: viper.GetString("MONGO_URI"),
			DB:  viper.GetString("MONGO_DB"),
		},
		Redis: RedisConfig{
			URL: viper.GetString("REDIS_URL"),
		},
		JWT: JWTConfig{
			AccessSecret:  viper.GetString("JWT_ACCESS_SECRET"),
			RefreshSecret: viper.GetString("JWT_REFRESH_SECRET"),
			AccessTTL:     accessTTL,
			RefreshTTL:    refreshTTL,
		},
		R2: R2Config{
			Endpoint:  viper.GetString("R2_ENDPOINT"),
			Bucket:    viper.GetString("R2_BUCKET"),
			AccessKey: viper.GetString("R2_ACCESS_KEY"),
			SecretKey: viper.GetString("R2_SECRET_KEY"),
		},
		Asynq: AsynqConfig{
			RedisURL:    viper.GetString("ASYNQ_REDIS_URL"),
			Concurrency: viper.GetInt("ASYNQ_CONCURRENCY"),
		},
		Cipher: CipherConfig{
			EncryptionKey: viper.GetString("TOKEN_ENCRYPTION_KEY"),
		},
		Email: ResendConfig{
			ApiKey: viper.GetString("RESEND_API_KEY"),
		},
	}

	return cfg, nil
}
