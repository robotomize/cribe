package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kkdai/youtube/v2"
	"github.com/robotomize/cribe/internal/db"
	"github.com/robotomize/cribe/internal/hashing"
	"github.com/robotomize/cribe/internal/logging"
	"github.com/robotomize/cribe/internal/srvenv"
	"github.com/robotomize/cribe/pkg/botstate"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

const SendingMessageError = "Oops, something went wrong, try sending the link again"

type Options struct {
	Bucket                    string
	TelegramPollingTimeout    int
	TelegramUpdatesMaxWorkers int
	FetchingMaxWorker         int
	UploadingMaxWorker        int
}

type Option func(*Dispatcher)

func NewDispatcher(env *srvenv.Env, opts ...Option) *Dispatcher {
	cfg := env.Config()
	d := Dispatcher{
		opts: Options{
			Bucket:                    cfg.Storage.Bucket,
			TelegramPollingTimeout:    cfg.Telegram.PollingTimeout,
			TelegramUpdatesMaxWorkers: cfg.TelegramUpdatesMaxWorkers,
			FetchingMaxWorker:         cfg.FetchingMaxWorkers,
			UploadingMaxWorker:        cfg.UploadingMaxWorkers,
		},
		env:           env,
		metadataDB:    db.NewMetadataRepository(env.DB()),
		hashFunc:      env.HashFunc(),
		youtubeClient: &youtube.Client{},
		broker:        NewAMQPBroker(env.AMQP()),
		storage:       env.Blob(),
	}

	for _, o := range opts {
		o(&d)
	}

	return &d
}

type Dispatcher struct {
	opts          Options
	metadataDB    MetadataDB
	env           *srvenv.Env
	hashFunc      hashing.HashFunc
	youtubeClient Yotuber
	storage       Blob
	broker        AMQPConnection
}

func (s *Dispatcher) Run(ctx context.Context, telegram *tgbotapi.BotAPI, cfg srvenv.Config) error {
	logger := logging.FromContext(ctx)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	updates, err := s.setupTelegramMode(ctx, telegram, cfg.Telegram)
	if err != nil {
		return fmt.Errorf("configuring telegram updates: %w", err)
	}

	var wg sync.WaitGroup
	for i := 0; i < s.opts.FetchingMaxWorker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err = s.consumingVideoFetching(ctx, telegram); err != nil {
				logger.Errorf("consume fetching: %v", err)
				cancel()
			}
		}()
	}

	for i := 0; i < s.opts.UploadingMaxWorker; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err = s.consumingVideoUploading(ctx, telegram); err != nil {
				logger.Errorf("consume uploading: %v", err)
				cancel()
			}
		}()
	}

	go func() {
		<-ctx.Done()
		telegram.StopReceivingUpdates()
	}()

	for i := 0; i < s.opts.TelegramUpdatesMaxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.dispatchingMessages(ctx, telegram, updates)
		}()
	}

	wg.Wait()

	return nil
}

func (s *Dispatcher) setupTelegramMode(ctx context.Context, telegram *tgbotapi.BotAPI, cfg srvenv.TelegramConfig) (tgbotapi.UpdatesChannel, error) {
	logger := logging.FromContext(ctx).Named("Dispatcher.setupTelegramMode")
	if cfg.WebHookURL != "" {
		if _, err := telegram.SetWebhook(tgbotapi.NewWebhook(cfg.WebHookURL + cfg.Token)); err != nil {
			return nil, fmt.Errorf("telegram set webhook: %w", err)
		}
		info, err := telegram.GetWebhookInfo()
		if err != nil {
			return nil, fmt.Errorf("telegram get webhook info: %w", err)
		}

		if info.LastErrorDate != 0 {
			logger.Errorf("Telegram callback failed: %s", info.LastErrorMessage)
		}

		updates := telegram.ListenForWebhook("/" + cfg.Token)
		go func() {
			if err = http.ListenAndServe(cfg.WebHookURL, nil); err != nil {
				logger.Fatalf("Listen and serve http stopped: %v", err)
			}
		}()

		return updates, nil
	}

	resp, err := telegram.RemoveWebhook()
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
	updatesChanConfig.Timeout = s.opts.TelegramPollingTimeout
	updates, err := telegram.GetUpdatesChan(updatesChanConfig)
	if err != nil {
		return nil, fmt.Errorf("telegram get updates chan: %w", err)
	}

	return updates, nil
}

func (s *Dispatcher) handleMessage(ctx context.Context, sender TelegramSender, message *tgbotapi.Message) error {
	logger := logging.FromContext(ctx)
	userID := message.From.ID
	sessionBackend := s.env.SessionBackend()
	session := botstate.NewSession(strconv.FormatInt(int64(userID), 10), sessionBackend, provideFSM())
	if err := session.Load(ctx); err != nil {
		return fmt.Errorf("unable load session: %w", err)
	}

	if session.Current() == botstate.Default {
		if err := session.SendEvent(
			ParseVideoEvent, ParsingCtx{
				hashFunc:      s.env.HashFunc(),
				message:       message.Text,
				chatID:        message.Chat.ID,
				logger:        logger,
				youtubeClient: s.youtubeClient,
				tg:            sender,
			},
		); err != nil {
			logger.Errorf("send session event: %v", err)
		}
	}

	return nil
}

func (s *Dispatcher) dispatchingMessages(ctx context.Context, sender TelegramSender, updates tgbotapi.UpdatesChannel) {
	logger := logging.FromContext(ctx).Named("Dispatcher.dispatchingMessages")

	for update := range updates {
		if update.Message != nil {
			if err := s.handleMessage(ctx, sender, update.Message); err != nil {
				logger.Errorf("handle telegram message: %v", err)
			}
		}
	}
}

func (s *Dispatcher) consumingVideoFetching(ctx context.Context, sender TelegramSender) error {
	channel, err := s.broker.Chan()
	if err != nil {
		return fmt.Errorf("can not create broker channel: %w", err)
	}

	defer channel.Close()

	if _, err = channel.QueueDeclare("fetching", true, false, false, false, nil); err != nil {
		return fmt.Errorf("can not declare broker queue: %w", err)
	}
	logger := logging.FromContext(ctx).Named("Dispatcher.consumingVideoFetching")
	messages, err := channel.Consume("fetching", "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("amqp consume fetching: %w", err)
	}

	go func() {
		<-ctx.Done()
		if err = channel.Close(); err != nil {
			logger.Errorf("broker channel close: %v", err)
		}
	}()

	for message := range messages {
		var payload Payload
		if err = json.Unmarshal(message.Body, &payload); err != nil {
			logger.Errorf("json unmarshal: %v", err)

			if _, err = sender.Send(tgbotapi.NewMessage(payload.ChatID, SendingMessageError)); err != nil {
				logger.Errorf("send message: %v", err)
				continue
			}
			continue
		}

		if err = s.fetch(ctx, channel, payload); err != nil {
			logger.Errorf("fetching video: %v", err)

			if _, err = sender.Send(tgbotapi.NewMessage(payload.ChatID, SendingMessageError)); err != nil {
				logger.Errorf("send message: %v", err)
				continue
			}
			continue
		}
	}

	return nil
}

func (s *Dispatcher) consumingVideoUploading(ctx context.Context, sender TelegramSender) error {
	channel, err := s.broker.Chan()
	if err != nil {
		return fmt.Errorf("can not create broker channel: %w", err)
	}

	defer channel.Close()

	if _, err = channel.QueueDeclare("uploading", true, false, false, false, nil); err != nil {
		return fmt.Errorf("can not declare broker queue: %w", err)
	}

	logger := logging.FromContext(ctx).Named("Dispatcher.consumingVideoUploading")
	messages, err := channel.Consume("uploading", "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("amqp consume uploading: %w", err)
	}

	go func() {
		<-ctx.Done()
		if err = channel.Close(); err != nil {
			logger.Errorf("broker channel close: %v", err)
		}
	}()

	for message := range messages {
		var payload Payload
		if err = json.Unmarshal(message.Body, &payload); err != nil {
			logger.Errorf("json unmarshal: %v", err)
			if _, err = sender.Send(tgbotapi.NewMessage(payload.ChatID, SendingMessageError)); err != nil {
				logger.Errorf("send message: %v", err)
				continue
			}
			continue
		}

		if err = s.upload(ctx, sender, payload); err != nil {
			logger.Errorf("uploading video: %v", err)

			if _, err = sender.Send(tgbotapi.NewMessage(payload.ChatID, SendingMessageError)); err != nil {
				logger.Errorf("send message: %v", err)
				continue
			}
			continue
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

type Payload struct {
	Mime    string `json:"mime"`
	Quality string `json:"quality"`
	VideoID string `json:"video_id"`
	ChatID  int64  `json:"chat_id"`
}

type ParsingCtx struct {
	hashFunc      func([]byte) ([]byte, error)
	broker        *amqp.Connection
	tg            TelegramSender
	youtubeClient Yotuber
	logger        *zap.SugaredLogger
	message       string
	chatID        int64
}

type ParsingAction struct{}

func (p *ParsingAction) Execute(eventCtx botstate.EventContext) botstate.EventType {
	ctx := eventCtx.(ParsingCtx)
	logger := ctx.logger.Named("ParsingAction.Execute")
	nextState := botstate.Noop

	video, err := ctx.youtubeClient.GetVideo(ctx.message)
	if err != nil {
		logger.Warnf("parsing video metadata: %v", err)
		if _, err = ctx.tg.Send(tgbotapi.NewMessage(ctx.chatID, SendingMessageError)); err != nil {
			logger.Errorf("send message: %v", err)

			return nextState
		}

		return nextState
	}

	encoded, err := json.Marshal(Payload{
		VideoID: video.ID,
		ChatID:  ctx.chatID,
	})
	if err != nil {
		logger.Errorf("json marshal: %v", err)
		if _, err = ctx.tg.Send(tgbotapi.NewMessage(ctx.chatID, SendingMessageError)); err != nil {
			logger.Errorf("sending message: %v", err)

			return nextState
		}

		return nextState
	}

	channel, err := ctx.broker.Channel()
	if err != nil {
		logger.Errorf("parsing action, asquire amqp chan: %v", err)
		if _, err = ctx.tg.Send(tgbotapi.NewMessage(ctx.chatID, SendingMessageError)); err != nil {
			logger.Errorf("sending message: %v", err)

			return nextState
		}

		return nextState
	}

	defer channel.Close()

	if err = channel.Publish(
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
