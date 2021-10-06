package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kkdai/youtube/v2"
	"github.com/robotomize/cribe/internal/hashing"
	"github.com/robotomize/cribe/internal/logging"
	"github.com/robotomize/cribe/internal/srvenv"
	"github.com/robotomize/cribe/internal/storage"
	"github.com/robotomize/cribe/pkg/botstate"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

const SendingMessageError = "Oops, something went wrong, try sending the link again"

var workerNum = runtime.NumCPU()

func NewDispatcher(env *srvenv.Env) *Dispatcher {
	return &Dispatcher{
		env:      env,
		client:   env.Telegram(),
		rabbitMQ: env.RabbitMQ(),
		storage:  env.Blob(),
	}
}

type Dispatcher struct {
	env      *srvenv.Env
	client   *tgbotapi.BotAPI
	storage  storage.Blob
	rabbitMQ *amqp.Connection
}

func (s *Dispatcher) Run(ctx context.Context, cfg srvenv.Config) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	updates, err := s.setupTelegramMode(ctx, cfg.Telegram)
	if err != nil {
		return fmt.Errorf("configuring telegram updates: %w", err)
	}

	amqpChannel, err := s.rabbitMQ.Channel()
	if err != nil {
		return fmt.Errorf("can not create rabbitMQ channel: %w", err)
	}

	defer amqpChannel.Close()

	if _, err = amqpChannel.QueueDeclare("fetching", true, false, false, false, nil); err != nil {
		return fmt.Errorf("can not declare rabbitMQ queue: %w", err)
	}

	if _, err = amqpChannel.QueueDeclare("uploading", true, false, false, false, nil); err != nil {
		return fmt.Errorf("can not declare rabbitMQ queue: %w", err)
	}

	if err = s.fetchingPool(ctx, amqpChannel); err != nil {
		return fmt.Errorf("consume fetching: %w", err)
	}

	if err = s.uploadingPool(ctx, amqpChannel); err != nil {
		return fmt.Errorf("consume uploading: %w", err)
	}

	if err = s.dispatchingMessages(ctx, updates, amqpChannel); err != nil {
		return fmt.Errorf("telegram dispatchingMessages: %w", err)
	}

	return nil
}

func (s *Dispatcher) setupTelegramMode(ctx context.Context, cfg srvenv.TelegramConfig) (tgbotapi.UpdatesChannel, error) {
	var updates tgbotapi.UpdatesChannel
	logger := logging.FromContext(ctx).Named("Dispatcher.setupTelegramMode")
	if cfg.WebHookURL != "" {
		if _, err := s.client.SetWebhook(tgbotapi.NewWebhook(cfg.WebHookURL + cfg.Token)); err != nil {
			return nil, fmt.Errorf("telegram set webhook: %w", err)
		}

		info, err := s.client.GetWebhookInfo()
		if err != nil {
			return nil, fmt.Errorf("telegram get webhook info: %w", err)
		}

		if info.LastErrorDate != 0 {
			logger.Errorf("Telegram callback failed: %s", info.LastErrorMessage)
		}

		updates = s.client.ListenForWebhook("/" + s.client.Token)
		go func() {
			if err = http.ListenAndServe(cfg.WebHookURL, nil); err != nil {
				logger.Fatalf("Listen and serve http stopped: %v", err)
			}
		}()
	} else {
		resp, err := s.client.RemoveWebhook()
		if err != nil {
			return nil, fmt.Errorf("telegram client remove webhook: %w", err)
		}

		if !resp.Ok {
			if resp.ErrorCode > 0 {
				return nil, fmt.Errorf(
					"telegram client remove webhook with error code %d and description %s",
					resp.ErrorCode, resp.Description,
				)
			}

			return nil, fmt.Errorf("telegram client remove webhook response not ok")
		}

		updatesChanConfig := tgbotapi.NewUpdate(0)
		updatesChanConfig.Timeout = cfg.PollingTimeout
		ch, err := s.client.GetUpdatesChan(updatesChanConfig)
		if err != nil {
			return nil, fmt.Errorf("telegram get updates chan: %w", err)
		}

		updates = ch
	}

	return updates, nil
}

func (s *Dispatcher) dispatchingMessages(
	ctx context.Context, updates tgbotapi.UpdatesChannel, rabbitMQChan *amqp.Channel,
) error {
	logger := logging.FromContext(ctx).Named("Dispatcher.dispatchingMessages")
	backend := s.env.SessionBackend()
	hashFunc := hashing.MD5HashFunc()

	go func() {
		<-ctx.Done()
		s.client.StopReceivingUpdates()
	}()

	for update := range updates {
		if update.Message != nil {
			userID := update.Message.From.ID
			session := botstate.NewSession(
				strconv.FormatInt(int64(userID), 10), backend, provideFSM(),
			)
			if err := session.Load(ctx); err != nil {
				logger.Errorf("unable load session: %v", err)
			}

			if session.Current() == botstate.Default {
				if err := session.SendEvent(
					ParseVideoEvent, ParsingCtx{
						hashFunc:        hashFunc,
						rabbitMQChannel: rabbitMQChan,
						message:         update.Message.Text,
						chatID:          update.Message.Chat.ID,
						logger:          logger,
						tg:              s.client,
					},
				); err != nil {
					logger.Errorf("send session event: %v", err)
				}
			}
		}
	}

	return nil
}

func provideFSM() *botstate.StateMachine {
	return botstate.NewStateMachine(
		botstate.States{
			botstate.Default: botstate.State{
				Events: botstate.Events{
					ParseVideoEvent: FetchingState,
				},
			},
			FetchingState: botstate.State{
				Action: &ParsingAction{},
				Events: botstate.Events{
					ParseVideoEvent: botstate.Default,
				},
			},
		},
	)
}

const (
	ParseVideoEvent botstate.EventType = "parse_url"
	FetchingState   botstate.StateType = "parsing_url"
)

type FetchingPayload struct {
	VideoID string `json:"video_id"`
	ChatID  int64  `json:"chat_id"`
}

type UploadPayload struct {
	ChatID         int64  `json:"chat_id"`
	Title          string `json:"title"`
	UploadFileName string `json:"upload_file_name"`
	LocalFileName  string `json:"local_file_name"`
	OriginFileName string `json:"origin_file_name"`
	Caption        string `json:"caption"`
}

type ParsingCtx struct {
	hashFunc        func([]byte) ([]byte, error)
	tg              *tgbotapi.BotAPI
	rabbitMQChannel *amqp.Channel
	logger          *zap.SugaredLogger
	message         string
	chatID          int64
}

type ParsingAction struct{}

func (p *ParsingAction) Execute(eventCtx botstate.EventContext) botstate.EventType {
	ctx := eventCtx.(ParsingCtx)
	logger := ctx.logger.Named("ParsingAction.Execute")
	client := youtube.Client{}
	nextState := botstate.Noop

	if _, err := client.GetVideo(ctx.message); err != nil {
		logger.Warnf("parsing video metadata: %v", err)
		if _, err = ctx.tg.Send(tgbotapi.NewMessage(ctx.chatID, SendingMessageError)); err != nil {
			logger.Errorf("send message: %v", err)

			return nextState
		}

		return nextState
	}

	encoded, err := json.Marshal(&FetchingPayload{VideoID: ctx.message, ChatID: ctx.chatID})
	if err != nil {
		logger.Errorf("json marshal: %v", err)
		if _, err = ctx.tg.Send(tgbotapi.NewMessage(ctx.chatID, SendingMessageError)); err != nil {
			logger.Errorf("sending message: %v", err)

			return nextState
		}

		return nextState
	}

	if err = ctx.rabbitMQChannel.Publish(
		"", "fetching", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        encoded,
		},
	); err != nil {
		logger.Errorf("publish message to fetching queue: %v", err)
		if _, err = ctx.tg.Send(tgbotapi.NewMessage(ctx.chatID, SendingMessageError)); err != nil {
			logger.Errorf("sending message: %v", err)
			return nextState
		}

		return nextState
	}

	if _, err = ctx.tg.Send(
		tgbotapi.NewMessage(ctx.chatID, "Starting to download the video"),
	); err != nil {
		logger.Errorf("send message: %v", err)

		return nextState
	}

	return nextState
}
