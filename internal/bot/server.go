package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"

	"github.com/enescakir/emoji"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/hashicorp/go-multierror"
	"github.com/kkdai/youtube/v2"
	"github.com/robotomize/cribe/internal/db"
	"github.com/robotomize/cribe/internal/logging"
	"github.com/robotomize/cribe/internal/srvenv"
	"github.com/robotomize/cribe/pkg/botstate"
	"github.com/streadway/amqp"
	"go.uber.org/zap"
)

const SendingMessageError = "Oops, something went wrong, try sending the link again"

const (
	QueueFetching  = "fetching"
	QueueUploading = "uploading"
)

type JobKind uint8

const (
	JobKindFetching JobKind = iota
	JobKindUploading
)

type Job struct {
	Kind JobKind
	Payload
}

type Options struct {
	Bucket                    string
	TelegramPollingTimeout    int
	TelegramUpdatesMaxWorkers int
	FetchingMaxWorker         int
	UploadingMaxWorker        int
}

type Option func(*Dispatcher)

func NewDispatcher(env *srvenv.Env, opts ...Option) (*Dispatcher, error) {
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
		youtubeClient: &youtube.Client{},
		broker:        NewAMQPBroker(env.AMQP()),
		storage:       env.Blob(),
		Jobs:          make([]Job, 0),
	}

	for _, o := range opts {
		o(&d)
	}

	return &d, nil
}

type Dispatcher struct {
	env  *srvenv.Env
	opts Options

	metadataDB    MetadataDB
	youtubeClient YoutubeClient
	storage       Blob
	broker        AMQPConnection

	mtx  sync.RWMutex
	Jobs []Job
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

	if err := s.finalization(); err != nil {
		return fmt.Errorf("finalization: %w", err)
	}

	return nil
}

func (s *Dispatcher) finalization() error {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	channel, err := s.broker.Chan()
	if err != nil {
		return fmt.Errorf("can not create broker channel: %w", err)
	}

	defer channel.Close()

	if _, err = channel.QueueDeclare(QueueFetching, true, false, false, false, nil); err != nil {
		return fmt.Errorf("can not declare broker queue: %w", err)
	}

	if _, err = channel.QueueDeclare(QueueUploading, true, false, false, false, nil); err != nil {
		return fmt.Errorf("can not declare broker queue: %w", err)
	}

	var merr *multierror.Error
	for _, job := range s.Jobs {
		encoded, err := json.Marshal(Payload{
			VideoID: job.VideoID,
			ChatID:  job.ChatID,
		})
		if err != nil {
			merr = multierror.Append(err, merr)
			continue
		}

		var queue string
		if job.Kind == JobKindFetching {
			queue = QueueFetching
		} else {
			queue = QueueUploading
		}

		if err = channel.Publish(
			"", queue, false, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        encoded,
			},
		); err != nil {
			merr = multierror.Append(err, merr)
		}
	}

	return merr.ErrorOrNil()
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
				ctx:           ctx,
				broker:        s.broker,
				message:       message.Text,
				chatID:        message.Chat.ID,
				logger:        logger,
				youtubeClient: s.youtubeClient,
				tg:            sender,
			},
		); err != nil {
			return fmt.Errorf("send session event: %w", err)
		}
	}

	if err := session.Flush(ctx); err != nil {
		return fmt.Errorf("session flush: %w", err)
	}

	return nil
}

const (
	StartCommandText = "start"
)

var StartCommandMessage = "Hi, this is a bot" + emoji.Robot.String() + " for downloading videos from youtube\n\n" +
	"Just send a link to the youtube video and follow the further instructions\n" +
	"\n*source code:* [github](https://github.com/robotomize/cribe)"

func (s *Dispatcher) dispatchingMessages(ctx context.Context, sender TelegramSender, updates tgbotapi.UpdatesChannel) {
	logger := logging.FromContext(ctx).Named("Dispatcher.dispatchingMessages")
	for update := range updates {
		if update.Message != nil {
			if update.Message.IsCommand() {
				cmd := update.Message.Command()
				if cmd == StartCommandText {
					config := tgbotapi.NewMessage(update.Message.Chat.ID, StartCommandMessage)
					config.ParseMode = tgbotapi.ModeMarkdown
					if _, err := sender.Send(config); err != nil {
						logger.Errorf("send message: %v", err)
					}
				}
				return
			}

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

	if _, err = channel.QueueDeclare(QueueFetching, true, false, false, false, nil); err != nil {
		return fmt.Errorf("can not declare broker queue: %w", err)
	}
	logger := logging.FromContext(ctx).Named("Dispatcher.consumingVideoFetching")
	messages, err := channel.Consume(QueueFetching, "", true, false, false, false, nil)
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

		s.mtx.Lock()
		s.Jobs = append(s.Jobs, Job{
			Kind:    JobKindFetching,
			Payload: Payload{VideoID: payload.VideoID, ChatID: payload.ChatID},
		})
		s.mtx.Unlock()

		if err = s.fetch(ctx, channel, payload); err != nil {
			if !errors.Is(err, context.Canceled) {
				logger.Errorf("fetching video: %v", err)

				if _, err = sender.Send(tgbotapi.NewMessage(payload.ChatID, SendingMessageError)); err != nil {
					logger.Errorf("send message: %v", err)
					continue
				}
				continue
			}
			continue
		}
		s.deleteJob(payload.VideoID, JobKindFetching)
	}

	return nil
}

func (s *Dispatcher) consumingVideoUploading(ctx context.Context, sender TelegramSender) error {
	channel, err := s.broker.Chan()
	if err != nil {
		return fmt.Errorf("can not create broker channel: %w", err)
	}

	defer channel.Close()

	if _, err = channel.QueueDeclare(QueueUploading, true, false, false, false, nil); err != nil {
		return fmt.Errorf("can not declare broker queue: %w", err)
	}

	logger := logging.FromContext(ctx).Named("Dispatcher.consumingVideoUploading")
	messages, err := channel.Consume(QueueUploading, "", true, false, false, false, nil)
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
			if !errors.Is(err, context.Canceled) {
				logger.Errorf("json unmarshal: %v", err)
				if _, err = sender.Send(tgbotapi.NewMessage(payload.ChatID, SendingMessageError)); err != nil {
					logger.Errorf("send message: %v", err)
					continue
				}
				continue
			}
			continue
		}

		s.mtx.Lock()
		s.Jobs = append(s.Jobs, Job{
			Kind:    JobKindUploading,
			Payload: Payload{VideoID: payload.VideoID, ChatID: payload.ChatID},
		})
		s.mtx.Unlock()

		if err = s.upload(ctx, sender, payload); err != nil {
			if !errors.Is(err, context.Canceled) {
				logger.Errorf("uploading video: %v", err)

				if _, err = sender.Send(tgbotapi.NewMessage(payload.ChatID, SendingMessageError)); err != nil {
					logger.Errorf("send message: %v", err)
					continue
				}
				continue
			}
			continue
		}
		s.deleteJob(payload.VideoID, JobKindUploading)
	}

	return nil
}

func (s *Dispatcher) deleteJob(videoID string, kind JobKind) {
	s.mtx.Lock()
	defer s.mtx.Unlock()
	for idx, job := range s.Jobs {
		if job.VideoID == videoID && job.Kind == kind {
			s.Jobs = append(s.Jobs[:idx], s.Jobs[idx+1:]...)
			break
		}
	}
}

func provideFSM() *botstate.StateMachine {
	return botstate.NewStateMachine(
		botstate.States{
			botstate.Default: botstate.State{
				Action: &DefaultAction{},
				Events: botstate.Events{
					ParseVideoEvent: ParsingVideoState,
				},
			},
			ParsingVideoState: botstate.State{
				Action: &ParsingAction{},
				Events: botstate.Events{
					GoToDefaultEvent: botstate.Default,
				},
			},
		},
	)
}

const (
	ParseVideoEvent   botstate.EventType = "parse_video"
	GoToDefaultEvent  botstate.EventType = "go_to_default"
	ParsingVideoState botstate.StateType = "parsing_video"
)

type Payload struct {
	Mime    string `json:"mime"`
	Quality string `json:"quality"`
	VideoID string `json:"video_id"`
	ChatID  int64  `json:"chat_id"`
}

type DefaultAction struct{}

func (p *DefaultAction) Execute(_ botstate.EventContext) botstate.EventType {
	return botstate.Noop
}

type ParsingCtx struct {
	ctx           context.Context
	broker        AMQPConnection
	tg            TelegramSender
	youtubeClient YoutubeClient
	logger        *zap.SugaredLogger
	message       string
	chatID        int64
}

type ParsingAction struct{}

func (p *ParsingAction) Execute(eventCtx botstate.EventContext) botstate.EventType {
	ctx := eventCtx.(ParsingCtx)
	logger := ctx.logger.Named("ParsingAction.Execute")
	nextState := GoToDefaultEvent

	video, err := ctx.youtubeClient.GetVideoContext(ctx.ctx, ctx.message)
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

	channel, err := ctx.broker.Chan()
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
