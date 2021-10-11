package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/kkdai/youtube/v2"
	"github.com/robotomize/cribe/internal/db"
	"github.com/robotomize/cribe/internal/storage"
	"github.com/streadway/amqp"
)

func (s *Dispatcher) fetch(ctx context.Context, queue AMQPChannel, payload Payload) error {
	client := youtube.Client{}
	video, err := client.GetVideo(payload.VideoID)
	if err != nil {
		return fmt.Errorf("parsing video metadata: %w", err)
	}

	format := video.Formats.WithAudioChannels().FindByQuality("hd720")
	if format == nil {
		return errors.New("video format not found")
	}

	payload.Mime = format.MimeType
	payload.Quality = format.Quality

	encoded, err := json.Marshal(Payload{
		ChatID:  payload.ChatID,
		VideoID: payload.VideoID,
		Mime:    payload.Mime,
		Quality: payload.Quality,
	})
	if err != nil {
		return fmt.Errorf("marshal fetching payload: %w", err)
	}

	metadata, err := s.metadataDB.FetchByMetadata(ctx, payload.VideoID, payload.Mime, payload.Quality)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			stream, _, err := client.GetStream(video, format)
			if err != nil {
				return fmt.Errorf("get video stream: %w", err)
			}

			buf, err := io.ReadAll(stream)
			if err != nil {
				return fmt.Errorf("real stream: %w", err)
			}

			if err = s.storage.CreateObject(ctx, s.opts.Bucket, video.ID, buf); err != nil {
				return fmt.Errorf("create object to storage: %w", err)
			}

			// sort preview by telegram constrain
			sort.Slice(video.Thumbnails, func(i, j int) bool {
				w1, w2 := video.Thumbnails[i].Width, video.Thumbnails[j].Width
				h1, h2 := video.Thumbnails[i].Height, video.Thumbnails[j].Height

				return w1 > w2 && h1 > h2 && w1 <= 320 && h1 <= 320
			})

			var thumb string
			if len(video.Thumbnails) > 0 {
				idx := strings.Index(video.Thumbnails[0].URL, "?")
				thumb = video.Thumbnails[0].URL[:idx]
			}

			if err = s.metadataDB.Save(ctx, db.Metadata{
				VideoID: payload.VideoID,
				Quality: payload.Quality,
				Mime:    payload.Mime,
				Params: db.VideoParams{
					Title:    video.Title,
					Width:    format.Width,
					Height:   format.Height,
					Duration: int(video.Duration.Seconds()),
					Thumb:    thumb,
				},
				FileID:    "",
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}); err != nil {
				return fmt.Errorf("saving metadata: %w", err)
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

		return fmt.Errorf("fetch metadata: %w", err)
	}

	if metadata.FileID == "" {
		if _, err = s.storage.GetObject(ctx, s.opts.Bucket, payload.VideoID); err != nil {
			if errors.Is(err, storage.ErrNotFound) {
				format := video.Formats.WithAudioChannels().FindByQuality("hd720")
				if format == nil {
					return fmt.Errorf("video format not found: %w", err)
				}

				stream, _, err := client.GetStream(video, format)
				if err != nil {
					return fmt.Errorf("get video stream: %w", err)
				}

				buf, err := io.ReadAll(stream)
				if err != nil {
					return fmt.Errorf("real stream: %w", err)
				}

				if err = s.storage.CreateObject(ctx, s.opts.Bucket, payload.VideoID, buf); err != nil {
					return fmt.Errorf("create object to storage: %w", err)
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
			return fmt.Errorf("get object from storage: %w", err)
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
