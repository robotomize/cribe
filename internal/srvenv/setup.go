package srvenv

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/robotomize/cribe/internal/db"
	"github.com/robotomize/cribe/internal/hashing"
	"github.com/robotomize/cribe/internal/storage"
	"github.com/robotomize/cribe/pkg/botstate"
	"github.com/sethvargo/go-envconfig"
	"github.com/streadway/amqp"
)

type SessionBackend interface {
	Get(ctx context.Context, k string) ([]byte, error)
	Set(ctx context.Context, k string, v []byte) error
	Delete(ctx context.Context, k string) error
}

type BackendType string

func Setup(ctx context.Context) (*Env, error) {
	var env Env
	var cfg Config

	if err := envconfig.Process(ctx, &cfg); err != nil {
		return nil, fmt.Errorf("env processing: %w", err)
	}
	env.config = cfg

	sessionBackend, err := ProvideSessionBackendFor(cfg)
	if err != nil {
		return nil, fmt.Errorf("setup session backend: %w", err)
	}

	env.sessionBackend = sessionBackend

	telegram, err := SetupTelegram(cfg.Telegram)
	if err != nil {
		return nil, fmt.Errorf("setup telegram client: %w", err)
	}

	rabbitMQConn, err := SetupAMQP(cfg.RabbitMQ)
	if err != nil {
		return nil, fmt.Errorf("setup rabbitmq connection: %w", err)
	}

	blob, err := ProvideStorageFor(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("setup storage: %w", err)
	}

	hashFn, err := ProvideHashingFunc(cfg)
	if err != nil {
		return nil, fmt.Errorf("setup hash func: %w", err)
	}
	database, err := db.New(&cfg.DB)
	if err != nil {
		return nil, fmt.Errorf("setup db: %w", err)
	}

	if err = migrationUp(cfg.DB.ConnectionURL()); err != nil {
		return nil, fmt.Errorf("setup migrations: %w", err)
	}

	env.db = database
	env.hashFunc = hashFn
	env.blob = blob
	env.telegram = telegram
	env.rabbitMQ = rabbitMQConn

	return &env, nil
}

func migrationUp(dsn string) error {
	basePath, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("os.Getwd: %w", err)
	}

	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("sql open: %w", err)
	}

	driver, err := postgres.WithInstance(conn, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("postgres instance: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file:///"+filepath.Join(basePath, "migrations"), "postgres", driver,
	)
	if err != nil {
		return fmt.Errorf("migrate db instance: %w", err)
	}

	if err = m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil
		}
		return fmt.Errorf("migration up: %w", err)
	}

	return nil
}

const (
	HashingFuncMD5 = "md5"
	HashingSha1    = "sha1"
)

func ProvideHashingFunc(cfg Config) (hashing.HashFunc, error) {
	var f hashing.HashFunc
	switch cfg.HashingFunc {
	case HashingSha1:
		f = hashing.SHA1HashFunc()
	case HashingFuncMD5:
		f = hashing.MD5HashFunc()
	default:
		return f, errors.New("unknown hashing function")
	}

	return f, nil
}

const (
	StorageTypeFS = "fs"
	StorageTypeS3 = "s3"
)

func ProvideStorageFor(ctx context.Context, cfg Config) (storage.Blob, error) {
	var blob storage.Blob
	switch cfg.Storage.Type {
	case StorageTypeFS:
		fs, err := storage.NewFilesystemStorage(ctx)
		if err != nil {
			return nil, fmt.Errorf("new fs storage: %w", err)
		}
		blob = fs
	case StorageTypeS3:
		s3, err := storage.NewS3(cfg.Storage.S3)
		if err != nil {
			return nil, fmt.Errorf("new S3 storage: %w", err)
		}
		blob = s3
	}

	return blob, nil
}

const (
	BackendTypeRedis    BackendType = "redis"
	BackendTypeInMemory BackendType = "in_memory"
)

func ProvideSessionBackendFor(cfg Config) (SessionBackend, error) {
	var backend SessionBackend
	switch cfg.SessionBackend {
	case BackendTypeRedis:
		redis, err := botstate.NewRedis(
			botstate.RedisConfig{
				Expiration: cfg.Redis.Expiration,
				Addr:       cfg.Redis.Addr,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("create redis session backend: %w", err)
		}
		backend = redis
	case BackendTypeInMemory:
		backend = botstate.NewInMemoryBackend()
	}

	return backend, nil
}

func SetupAMQP(cfg AMQPConfig) (*amqp.Connection, error) {
	conn, err := amqp.DialConfig(
		cfg.ConnectionURL, amqp.Config{
			Heartbeat: cfg.HeartBeatDuration,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("amqp dialing: %w", err)
	}

	return conn, nil
}

func SetupTelegram(cfg TelegramConfig) (*tgbotapi.BotAPI, error) {
	if cfg.ProxyAddr != "" {
		client, err := tgbotapi.NewBotAPIWithAPIEndpoint(
			cfg.Token, fmt.Sprintf("%s://%s", cfg.ProxySchema, cfg.ProxyAddr)+"/bot%s/%s",
		)
		if err != nil {
			return nil, fmt.Errorf("unable create telegram client with proxy: %w", err)
		}

		return client, nil
	}

	client, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("unable create telegram client: %w", err)
	}

	return client, nil
}
