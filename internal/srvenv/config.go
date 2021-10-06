package srvenv

import (
	"time"

	"github.com/robotomize/cribe/internal/storage"
)

type RedisConfig struct {
	Expiration time.Duration `env:"REDIS_EXPIRATION,default=86400s"`
	Addr       string        `env:"REDIS_ADDR,default=6379"`
}

type TelegramConfig struct {
	WebHookURL     string `env:"TELEGRAM_WEBHOOK_URL"`
	WebHookAddr    string `env:"TELEGRAM_WEBHOOK_ADDR"`
	Token          string `env:"TELEGRAM_TOKEN"`
	PollingTimeout int    `env:"TELEGRAM_POLLING_TIMEOUT,default=30"`
}

type RabbitMQConfig struct {
	ConnectionURL string `env:"AMQP_SERVER_URL,default=amqp://guest:guest@localhost:5672/"`
}

type StorageConfig struct {
	Type   string `env:"STORAGE_TYPE,default=telegram"`
	Bucket string `env:"UPLOAD_BUCKET_NAME"`
	S3     storage.S3Config
}

type Config struct {
	SessionBackend BackendType `env:"SESSION_BACKEND_TYPE,default=redis"`
	Redis          RedisConfig
	Telegram       TelegramConfig
	RabbitMQ       RabbitMQConfig
	Storage        StorageConfig
}
