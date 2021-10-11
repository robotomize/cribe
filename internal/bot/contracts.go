package bot

import (
	"context"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kkdai/youtube/v2"
	"github.com/robotomize/cribe/internal/db"
	"github.com/streadway/amqp"
	"io"
)

var _ io.ReadCloser

type (
	Yotuber interface {
		GetVideo(url string) (*youtube.Video, error)
		GetStream(video *youtube.Video, format *youtube.Format) (io.ReadCloser, int64, error)
	}

	MetadataDB interface {
		FetchByMetadata(ctx context.Context, videoID string, mime string, quality string) (db.Metadata, error)
		Save(ctx context.Context, model db.Metadata) error
	}

	Telegram interface {
		Send(c tgbotapi.Chattable) (tgbotapi.Message, error)
		UploadFileWithContext(ctx context.Context, endpoint string, params map[string]string, fieldname string, file interface{}) (tgbotapi.APIResponse, error)
		GetUpdatesChan(config tgbotapi.UpdateConfig) (tgbotapi.UpdatesChannel, error)
		ListenForWebhook(pattern string) tgbotapi.UpdatesChannel
		GetWebhookInfo() (tgbotapi.WebhookInfo, error)
		SetWebhook(config tgbotapi.WebhookConfig) (tgbotapi.APIResponse, error)
		RemoveWebhook() (tgbotapi.APIResponse, error)
		StopReceivingUpdates()
	}

	AMQPChannel interface {
		Publish(exchange, key string, mandatory, immediate bool, msg amqp.Publishing) error
		QueueDeclare(name string, durable, autoDelete, exclusive, noWait bool, args amqp.Table) (amqp.Queue, error)
		Consume(queue, consumer string, autoAck, exclusive, noLocal, noWait bool, args amqp.Table) (<-chan amqp.Delivery, error)
		Close() error
	}

	AMQPConnection interface {
		Chan() (AMQPChannel, error)
		Close() error
	}

	Blob interface {
		CreateObject(ctx context.Context, bucket, key string, contents []byte) error
		DeleteObject(ctx context.Context, bucket, key string) error
		GetObject(ctx context.Context, bucket, key string) ([]byte, error)
	}
)
