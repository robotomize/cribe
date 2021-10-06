package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robotomize/cribe/internal/logging"
	"github.com/robotomize/cribe/internal/storage"
	"github.com/streadway/amqp"
)

func (s *Dispatcher) upload(payload UploadPayload) error {
	switch s.storage.(type) {
	case *storage.S3:
		buf, err := os.ReadFile(payload.LocalFileName)
		if err != nil {
			return fmt.Errorf("read file to buf: %v", err)
		}
		if err = s.storage.CreateObject(context.Background(), "crivevideo", payload.UploadFileName, buf); err != nil {
			return fmt.Errorf("create blob object: %w", err)
		}
		if _, err = s.client.Send(
			tgbotapi.NewMessage(
				payload.ChatID, payload.Title+"\n\n"+s.storage.(*storage.S3).PublicAccess(
					context.Background(), "crivevideo", payload.UploadFileName,
				),
			),
		); err != nil {
			return fmt.Errorf("send message with video: %w", err)
		}
	default:
		file, err := os.OpenFile(payload.LocalFileName, os.O_RDONLY, 0655)
		if err != nil {
			return fmt.Errorf("open local file: %v", err)
		}
		defer file.Close()

		config := tgbotapi.NewVideoUpload(
			payload.ChatID, tgbotapi.FileReader{
				Name:   payload.LocalFileName,
				Reader: file,
				Size:   -1,
			},
		)
		config.Caption = payload.Caption

		if _, err = s.client.Send(config); err != nil {
			return fmt.Errorf("send message with video: %w", err)
		}

		return nil
	}
	return nil
}

func (s *Dispatcher) uploadingPool(ctx context.Context, queue *amqp.Channel) error {
	logger := logging.FromContext(ctx).Named("Dispatcher.uploadingPool")
	messages, err := queue.Consume("uploading", "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("amqp consume uploading: %w", err)
	}

	for i := 0; i < workerNum; i++ {
		go func() {
			for {
				select {
				case <-ctx.Done():
					queue.Close()
				case message, ok := <-messages:
					if !ok {
						return
					}
					var payload UploadPayload
					if err = json.Unmarshal(message.Body, &payload); err != nil {
						logger.Errorf("json unmarshal: %v", err)
						if _, err = s.client.Send(tgbotapi.NewMessage(payload.ChatID, SendingMessageError)); err != nil {
							logger.Errorf("send message: %v", err)
							continue
						}
						continue
					}

					if err = s.upload(payload); err != nil {
						logger.Errorf("uploading video: %v", err)

						if _, err = s.client.Send(tgbotapi.NewMessage(payload.ChatID, SendingMessageError)); err != nil {
							logger.Errorf("send message: %v", err)
							continue
						}
						continue
					}
				}
			}
		}()
	}

	return nil
}
