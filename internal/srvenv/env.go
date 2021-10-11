package srvenv

import (
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robotomize/cribe/internal/db"
	"github.com/robotomize/cribe/internal/hashing"
	"github.com/robotomize/cribe/internal/storage"
	"github.com/streadway/amqp"
)

type Env struct {
	config         Config
	db             *db.DB
	hashFunc       hashing.HashFunc
	sessionBackend SessionBackend
	telegram       *tgbotapi.BotAPI
	rabbitMQ       *amqp.Connection
	blob           storage.Blob
}

func (e Env) Config() Config {
	return e.config
}

func (e Env) DB() *db.DB {
	return e.db
}

func (e Env) HashFunc() hashing.HashFunc {
	return e.hashFunc
}

func (e Env) Blob() storage.Blob {
	return e.blob
}

func (e Env) SessionBackend() SessionBackend {
	return e.sessionBackend
}

func (e Env) AMQP() *amqp.Connection {
	return e.rabbitMQ
}

func (e Env) Telegram() *tgbotapi.BotAPI {
	return e.telegram
}
