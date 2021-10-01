package bot

import (
	"context"
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robotomize/cribe/internal/logging"
	"net/http"
)

type Options struct {
	WebHookURL  string
	WebHookAddr string
	Token       string
}

type Dispatcher struct {
	opts Options
	tg   *tgbotapi.BotAPI
}

func (s *Dispatcher) Run(ctx context.Context) error {
	var updates tgbotapi.UpdatesChannel
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	logger := logging.FromContext(ctx)

	if s.opts.WebHookURL != "" {
		_, err := s.tg.SetWebhook(tgbotapi.NewWebhook(s.opts.WebHookURL + s.opts.Token))
		if err != nil {
			return fmt.Errorf("tg bot set webhook: %w", err)
		}

		info, err := s.tg.GetWebhookInfo()
		if err != nil {
			return fmt.Errorf("get webhook info: %w", err)
		}

		if info.LastErrorDate != 0 {
			logger.Errorf("Telegram callback failed: %s", info.LastErrorMessage)
		}

		updates = s.tg.ListenForWebhook("/" + s.opts.Token)
		go func() {
			if err := http.ListenAndServe(s.opts.WebHookAddr, nil); err != nil {
				logger.Fatalf("listen and serve http stopped: %v", err)
				cancel()
			}
		}()
	} else {
		resp, err := s.tg.RemoveWebhook()
		if err != nil {
			return fmt.Errorf("remove webhook: %w", err)
		}

		if !resp.Ok {
			if resp.ErrorCode > 0 {
				return fmt.Errorf(
					"remove webhook with error code %d and description %s", resp.ErrorCode, resp.Description,
				)
			}
			return fmt.Errorf("remove webhook response not ok=)")
		}

		upd := tgbotapi.NewUpdate(0)
		upd.Timeout = 30
		up, err := s.tg.GetUpdatesChan(upd)
		if err != nil {
			return fmt.Errorf("tg get updates chan: %w", err)
		}
		updates = up
	}

	defer s.tg.StopReceivingUpdates()
	if err := s.loop(ctx, updates); err != nil {
		return fmt.Errorf("execute loop: %w", err)
	}

	return nil
}

func (s *Dispatcher) loop(ctx context.Context, recv tgbotapi.UpdatesChannel) error {
	logger := logging.FromContext(ctx).Named("server.loop")
	_ = logger
	for update := range recv {
		if update.Message != nil {

		}

		if update.CallbackQuery != nil {

		}
	}

	return nil
}
