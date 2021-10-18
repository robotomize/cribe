package srvenv

import (
	"time"

	"github.com/robotomize/cribe/internal/db"
	"github.com/robotomize/cribe/internal/storage"
)

type RedisConfig struct {
	Expiration time.Duration `env:"REDIS_EXPIRATION,default=86400s"`
	Addr       string        `env:"REDIS_ADDR,default=localhost:6380"`
}

type TelegramConfig struct {
	ProxySchema    string `env:"TELEGRAM_PROXY_SCHEMA,default=http"`
	ProxyAddr      string `env:"TELEGRAM_PROXY_ADDR,default=127.0.0.1:8081"`
	WebHookURL     string `env:"TELEGRAM_WEBHOOK_URL"`
	WebHookAddr    string `env:"TELEGRAM_WEBHOOK_ADDR"`
	Token          string `env:"TELEGRAM_TOKEN"`
	PollingTimeout int    `env:"TELEGRAM_POLLING_TIMEOUT,default=10"`
}

type AMQPConfig struct {
	ConnectionURL     string        `env:"AMQP_SERVER_URL,default=amqp://guest:guest@localhost:5672/"`
	HeartBeatDuration time.Duration `env:"AMQP_HEARTBEAT_DURATION,default=12h"`
}

type StorageConfig struct {
	Type   string `env:"STORAGE_TYPE,default=fs"`
	Bucket string `env:"UPLOAD_BUCKET_NAME,default=/tmp"`
	S3     storage.S3Config
}

type Config struct {
	Addr                      string      `env:"ADDR,default=localhost:8080"`
	LogLevel                  string      `env:"LOG_LEVEL,default=error"`
	TelegramUpdatesMaxWorkers int         `env:"TELEGRAM_UPDATES_MAX_WORKERS,default=10"`
	FetchingMaxWorkers        int         `env:"FETCHING_MAX_WORKERS,default=10"`
	UploadingMaxWorkers       int         `env:"UPLOADING_MAX_WORKERS,default=5"`
	HashingFunc               string      `env:"FILE_HASHING_FUNC,default=md5"`
	SessionBackend            BackendType `env:"SESSION_BACKEND_TYPE,default=redis"`
	DB                        db.Config
	Redis                     RedisConfig
	Telegram                  TelegramConfig
	RabbitMQ                  AMQPConfig
	Storage                   StorageConfig
}
