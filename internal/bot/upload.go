package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/robotomize/cribe/internal/db"
	"github.com/robotomize/cribe/internal/storage"
	"github.com/streadway/amqp"
)

func (s *Dispatcher) upload(ctx context.Context, payload Payload) error {
	encoded, err := json.Marshal(Payload{
		VideoID: payload.VideoID,
		ChatID:  payload.ChatID,
	})
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}

	channel, err := s.broker.Chan()
	if err != nil {
		return fmt.Errorf("can not create broker channel: %w", err)
	}

	defer channel.Close()

	if _, err = channel.QueueDeclare("fetching", true, false, false, false, nil); err != nil {
		return fmt.Errorf("can not declare broker queue: %w", err)
	}

	metadata, err := s.metadataDB.FetchByMetadata(ctx, payload.VideoID, payload.Mime, payload.Quality)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			if err = channel.Publish(
				"", "fetching", false, false, amqp.Publishing{
					ContentType: "application/json",
					Body:        encoded,
				},
			); err != nil {
				return fmt.Errorf("publish: %w", err)
			}
			return nil
		}
		return fmt.Errorf("fetching metadata: %w", err)
	}

	if metadata.FileID != "" {
		config := tgbotapi.NewVideoShare(payload.ChatID, metadata.FileID)
		config.Caption = metadata.Params.Title
		if _, err = s.tg.Send(config); err != nil {
			return fmt.Errorf("send message with video: %w", err)
		}

		return nil
	}

	file, err := s.storage.GetObject(ctx, s.opts.Bucket, payload.VideoID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			if err = channel.Publish(
				"", "fetching", false, false, amqp.Publishing{
					ContentType: "application/json",
					Body:        encoded,
				},
			); err != nil {
				return fmt.Errorf("publish: %w", err)
			}
			return nil
		}

		return fmt.Errorf("get object from storage: %w", err)
	}

	params := map[string]string{
		"chat_id":              strconv.Itoa(int(payload.ChatID)),
		"width":                strconv.Itoa(metadata.Params.Width),
		"height":               strconv.Itoa(metadata.Params.Height),
		"duration":             strconv.Itoa(metadata.Params.Duration),
		"thumb":                metadata.Params.Thumb,
		"caption":              metadata.Params.Title,
		"disable_notification": "true",
	}
	resp, err := s.tg.UploadFileWithContext(ctx, "sendVideo", params, "video", tgbotapi.FileBytes{
		Name:  metadata.Params.Title,
		Bytes: file,
	})
	if err != nil {
		return fmt.Errorf("upload file: %w", err)
	}

	if !resp.Ok {
		return fmt.Errorf("can not upload video: %v %v", resp.ErrorCode, resp.Description)
	}

	var message tgbotapi.Message
	if err = json.Unmarshal(resp.Result, &message); err != nil {
		return fmt.Errorf("unmarshal upload file result raw message: %w", err)
	}

	metadata.FileID = message.Video.FileID
	if err = s.metadataDB.Save(ctx, metadata); err != nil {
		return fmt.Errorf("saving metadata: %w", err)
	}

	if err = s.storage.DeleteObject(ctx, s.opts.Bucket, metadata.VideoID); err != nil {
		return fmt.Errorf("delete object from storage: %w", err)
	}

	return nil
}
