package srvenv

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robotomize/cribe/internal/storage"
	"github.com/streadway/amqp"
)

type Env struct {
	config         Config
	sessionBackend SessionBackend
	telegram       *tgbotapi.BotAPI
	rabbitMQ       *amqp.Connection
	blob           storage.Blob
}

func (e Env) Config() Config {
	return e.config
}

func (e Env) Blob() storage.Blob {
	return e.blob
}

func (e Env) SessionBackend() SessionBackend {
	return e.sessionBackend
}

func (e Env) RabbitMQ() *amqp.Connection {
	return e.rabbitMQ
}

func (e Env) Telegram() *tgbotapi.BotAPI {
	return e.telegram
}
