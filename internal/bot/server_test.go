package bot

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/kkdai/youtube/v2"
	"github.com/robotomize/cribe/internal/db"
	"github.com/robotomize/cribe/internal/srvenv"
	"github.com/robotomize/cribe/internal/storage"
)

func TestDispatcher_fetch(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name             string
		err              error
		payload          Payload
		video            *youtube.Video
		videoErr         error
		metadata         db.Metadata
		metadataErr      error
		saveMetaErr      error
		storageCreateErr error
		storageGetObject []byte
		storageGetErr    error
		streamVideoErr   error
		publishErr       error
	}{
		{
			name:     "test_video_error",
			payload:  Payload{ChatID: 1, VideoID: "videID"},
			videoErr: errors.New("mock error"),
			err:      errors.New("mock error"),
		},
		{
			name:     "test_video_not_valid_id_error",
			payload:  Payload{ChatID: 1, VideoID: "videID"},
			videoErr: errors.New("mock error"),
			err:      errors.New("mock error"),
		},
		{
			name:    "test_format_without_video_err",
			payload: Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:     errors.New("mock error"),
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
				},
				Formats: []youtube.Format{
					{
						Quality: "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
		},
		{
			name:    "test_fetch_meta_unknown_error",
			payload: Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:     errors.New("mock error"),
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
				},
				Formats: []youtube.Format{
					{
						AudioChannels: 2,
						Quality:       "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
			metadataErr: errors.New("mock error"),
		},
		{
			name:           "test_fetch_meta_not_found_stream_error",
			payload:        Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:            errors.New("mock error"),
			streamVideoErr: errors.New("mock error"),
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
				},
				Formats: []youtube.Format{
					{
						AudioChannels: 2,
						Quality:       "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
			metadataErr: db.ErrNotFound,
		},
		{
			name:             "test_fetch_meta_not_found_storage_create_error",
			payload:          Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:              errors.New("mock error"),
			storageCreateErr: errors.New("mock error"),
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
				},
				Formats: []youtube.Format{
					{
						AudioChannels: 2,
						Quality:       "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
			metadataErr: db.ErrNotFound,
		},
		{
			name:        "test_fetch_meta_not_found_insert_error",
			payload:     Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:         errors.New("mock error"),
			saveMetaErr: errors.New("mock error"),
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
				},
				Formats: []youtube.Format{
					{
						AudioChannels: 2,
						Quality:       "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
			metadataErr: db.ErrNotFound,
		},
		{
			name:       "test_fetch_meta_not_found_publish_err",
			payload:    Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:        errors.New("mock error"),
			publishErr: errors.New("mock error"),
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
				},
				Formats: []youtube.Format{
					{
						AudioChannels: 2,
						Quality:       "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
			metadataErr: db.ErrNotFound,
		},
		{
			name:    "test_fetch_meta_not_found_publish",
			payload: Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:     errors.New("mock error"),
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
					{
						URL:    "http://google.ru",
						Width:  200,
						Height: 200,
					},
					{
						URL:    "http://google.ru",
						Width:  400,
						Height: 400,
					},
				},
				Formats: []youtube.Format{
					{
						AudioChannels: 2,
						Quality:       "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
			metadataErr: db.ErrNotFound,
		},
		{
			name:          "test_fetch_meta_not_found_get_object_unknown_error",
			payload:       Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:           errors.New("mock error"),
			storageGetErr: errors.New("mock error"),
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
					{
						URL:    "http://google.ru",
						Width:  200,
						Height: 200,
					},
					{
						URL:    "http://google.ru",
						Width:  400,
						Height: 400,
					},
				},
				Formats: []youtube.Format{
					{
						AudioChannels: 2,
						Quality:       "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
		},
		{
			name:          "test_fetch_meta_not_found_get_object_not_found",
			payload:       Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:           errors.New("mock error"),
			storageGetErr: storage.ErrNotFound,
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
					{
						URL:    "http://google.ru",
						Width:  200,
						Height: 200,
					},
					{
						URL:    "http://google.ru",
						Width:  400,
						Height: 400,
					},
				},
				Formats: []youtube.Format{
					{
						AudioChannels: 2,
						Quality:       "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
		},
		{
			name:             "test_fetch_meta_not_found_get_object_exist",
			payload:          Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:              errors.New("mock error"),
			storageGetObject: []byte("hello file"),
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
					{
						URL:    "http://google.ru",
						Width:  200,
						Height: 200,
					},
					{
						URL:    "http://google.ru",
						Width:  400,
						Height: 400,
					},
				},
				Formats: []youtube.Format{
					{
						AudioChannels: 2,
						Quality:       "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
		},
		{
			name:             "test_fetch_publish",
			payload:          Payload{ChatID: 1, VideoID: "rFejpH_tAHM"},
			err:              errors.New("mock error"),
			storageGetObject: []byte("hello file"),
			metadata:         db.Metadata{FileID: "1234"},
			video: &youtube.Video{
				ID:    "1234",
				Title: "title-1234",
				Thumbnails: youtube.Thumbnails{
					{
						URL:    "http://google.ru",
						Width:  300,
						Height: 300,
					},
					{
						URL:    "http://google.ru",
						Width:  200,
						Height: 200,
					},
					{
						URL:    "http://google.ru",
						Width:  400,
						Height: 400,
					},
				},
				Formats: []youtube.Format{
					{
						AudioChannels: 2,
						Quality:       "hd720",
					},
				},
				DASHManifestURL: "",
				HLSManifestURL:  "",
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps := newDeps(t)
			deps.youtubeClient.EXPECT().GetVideo(tc.payload.VideoID).Return(tc.video, tc.videoErr).AnyTimes()
			deps.metadata.
				EXPECT().
				FetchByMetadata(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.metadata, tc.metadataErr).
				AnyTimes()
			deps.metadata.
				EXPECT().
				Save(gomock.Any(), gomock.Any()).
				Return(tc.saveMetaErr).AnyTimes()
			deps.storage.
				EXPECT().
				CreateObject(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.storageCreateErr).
				AnyTimes()
			deps.storage.
				EXPECT().
				GetObject(gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.storageGetObject, tc.storageGetErr).
				AnyTimes()
			deps.amqp.EXPECT().Chan().Return(NewMockAMQPChannel(deps.ctrl), nil).AnyTimes()
			deps.channel.
				EXPECT().
				Publish(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
				Return(tc.publishErr).
				AnyTimes()
			deps.channel.EXPECT().Close().Return(nil).AnyTimes()
			stringReader := strings.NewReader("12345")
			stringReadCloser := io.NopCloser(stringReader)

			deps.youtubeClient.
				EXPECT().
				GetStream(gomock.Any(), gomock.Any()).
				Return(stringReadCloser, int64(stringReader.Len()), tc.streamVideoErr).
				AnyTimes()

			d := NewDispatcher(&srvenv.Env{})
			d.youtubeClient = deps.youtubeClient
			d.metadataDB = deps.metadata
			d.storage = deps.storage

			err := d.fetch(context.Background(), deps.channel, tc.payload)
			if (err != nil) && tc.err == nil {
				t.Errorf("got: %t, expected: %t", err != nil, tc.err == nil)
			}
		})
	}
}

func newDeps(t testing.TB) *deps {
	ctrl := gomock.NewController(t)
	return &deps{
		ctrl:          ctrl,
		amqp:          NewMockAMQPConnection(ctrl),
		channel:       NewMockAMQPChannel(ctrl),
		metadata:      NewMockMetadataDB(ctrl),
		youtubeClient: NewMockYotuber(ctrl),
		storage:       NewMockBlob(ctrl),
	}
}

type deps struct {
	ctrl          *gomock.Controller
	youtubeClient *MockYotuber
	amqp          *MockAMQPConnection
	channel       *MockAMQPChannel
	metadata      *MockMetadataDB
	storage       *MockBlob
}
