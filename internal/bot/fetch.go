package bot

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/kkdai/youtube/v2"
	"github.com/robotomize/cribe/internal/hashing"
	"github.com/robotomize/cribe/internal/logging"
	"github.com/streadway/amqp"
)

func (s *Dispatcher) fetch(queue *amqp.Channel, videoID string, chatID int64) error {
	client := youtube.Client{}
	video, err := client.GetVideo(videoID)
	if err != nil {
		return fmt.Errorf("parsing video metadata: %w", err)
	}

	md5 := hashing.MD5HashFunc()
	hash, err := md5([]byte(video.ID))
	if err != nil {
		return fmt.Errorf("MD5 hashing file name: %w", err)
	}

	fileName := fmt.Sprintf("%s.%s", hex.EncodeToString(hash[:]), "mp4")
	localFileName := filepath.Join("/", "tmp", fileName)

	encoded, err := json.Marshal(&UploadPayload{
		ChatID:         chatID,
		Title:          video.Title,
		UploadFileName: fileName,
		LocalFileName:  localFileName,
		OriginFileName: video.ID,
		Caption:        video.Title,
	})
	if err != nil {
		return fmt.Errorf("marshal fetching payload: %w", err)
	}

	if _, err = os.Stat(localFileName); err != nil {
		var pathError *os.PathError
		if os.IsNotExist(err) {
			file, err := os.OpenFile(localFileName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0655)
			if err != nil {
				return fmt.Errorf("open file %s: %w", fileName, err)
			}

			defer file.Close()

			format := video.Formats.WithAudioChannels().FindByQuality("hd720")
			if format == nil {
				return fmt.Errorf("video format not found: %w", err)
			}

			stream, _, err := client.GetStream(video, format)
			if err != nil {
				return fmt.Errorf("get video stream: %w", err)
			}

			if _, err = io.Copy(file, stream); err != nil {
				return fmt.Errorf("copy from stream: %w", err)
			}

			if err := file.Sync(); err != nil {
				return fmt.Errorf("flush file: %w", err)
			}

			if err = queue.Publish(
				"", "uploading", false, false, amqp.Publishing{
					ContentType: "application/json",
					Body:        encoded,
				},
			); err != nil {
				return fmt.Errorf("publish to uploading queue: %w", err)
			}

			return nil
		}

		if !errors.As(err, &pathError) {
			return fmt.Errorf("stat file: %w", err)
		}
	}

	if err = queue.Publish(
		"", "uploading", false, false, amqp.Publishing{
			ContentType: "application/json",
			Body:        encoded,
		},
	); err != nil {
		return fmt.Errorf("publish to uploading queue: %w", err)
	}

	return nil
}

func (s *Dispatcher) fetchingPool(ctx context.Context, queue *amqp.Channel) error {
	logger := logging.FromContext(ctx).Named("Dispatcher.fetchingPool")
	messages, err := queue.Consume("fetching", "", true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("amqp consume fetching: %w", err)
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
					var payload FetchingPayload
					if err = json.Unmarshal(message.Body, &payload); err != nil {
						logger.Errorf("json unmarshal: %v", err)

						if _, err = s.client.Send(tgbotapi.NewMessage(payload.ChatID, SendingMessageError)); err != nil {
							logger.Errorf("send message: %v", err)
							continue
						}
						continue
					}

					if err = s.fetch(queue, payload.VideoID, payload.ChatID); err != nil {
						logger.Errorf("fetching video: %v", err)

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
