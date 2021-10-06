package srvenv

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
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

	env.sessionBackend = ProvideSessionBackendFor(cfg)
	telegram, err := SetupTelegram(cfg.Telegram)
	if err != nil {
		return nil, fmt.Errorf("configure telegram client: %w", err)
	}

	rabbitMQConn, err := SetupAMQP(cfg.RabbitMQ)
	if err != nil {
		return nil, fmt.Errorf("configure rabbitmq connection: %w", err)
	}

	blob, err := ProvideStorageFor(cfg)
	if err != nil {
		return nil, fmt.Errorf("configure storage: %w", err)
	}

	env.blob = blob
	env.telegram = telegram
	env.rabbitMQ = rabbitMQConn

	return &env, nil
}

const (
	StorageTypeTelegram = "telegram"
	StorageTypeS3       = "s3"
)

func ProvideStorageFor(cfg Config) (storage.Blob, error) {
	var blob storage.Blob

	switch cfg.Storage.Type {
	case StorageTypeTelegram:
	case StorageTypeS3:
		s3, err := storage.NewS3(cfg.Storage.S3)
		if err != nil {
			return nil, fmt.Errorf("configure blob: %w", err)
		}
		blob = s3
	}

	return blob, nil
}

const (
	BackendTypeRedis    BackendType = "redis"
	BackendTypeInMemory BackendType = "in_memory"
)

func ProvideSessionBackendFor(cfg Config) SessionBackend {
	var backend SessionBackend
	switch cfg.SessionBackend {
	case BackendTypeRedis:
		backend = botstate.NewRedis(
			botstate.RedisConfig{
				Expiration: cfg.Redis.Expiration,
				Addr:       cfg.Redis.Addr,
			},
		)
	case BackendTypeInMemory:
		backend = botstate.NewInMemoryBackend()
	}

	return backend
}

func SetupAMQP(cfg RabbitMQConfig) (*amqp.Connection, error) {
	conn, err := amqp.Dial(cfg.ConnectionURL)
	if err != nil {
		return nil, fmt.Errorf("amqp dialing: %w", err)
	}

	return conn, nil
}

func SetupTelegram(cfg TelegramConfig) (*tgbotapi.BotAPI, error) {
	client, err := tgbotapi.NewBotAPI(cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("unable create telegram client: %w", err)
	}

	return client, nil
}
