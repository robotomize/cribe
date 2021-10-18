package localization

import (
	"embed"
	"path/filepath"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/robotomize/cribe/internal/logging"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v2"
)

var (
	//go:embed resources
	fs   embed.FS
	path = "resources"
)

// nolint
var once sync.Once
var localizationBundle *i18n.Bundle

func Bundle() *i18n.Bundle {
	return localizationBundle
}

func init() {
	logger := logging.DefaultLogger()
	once.Do(func() {
		localizationBundle = i18n.NewBundle(language.English)
		localizationBundle.RegisterUnmarshalFunc("yml", yaml.Unmarshal)
	})

	dir, err := fs.ReadDir(path)
	if err != nil {
		logger.Fatalf("fs.ReadDir: %v", err)
	}

	for _, entry := range dir {
		if !entry.IsDir() {
			bytes, err := fs.ReadFile(filepath.Join(path, entry.Name()))
			if err != nil {
				logger.Fatalf("fs.ReadFile: %v", err)
			}

			if _, err = localizationBundle.ParseMessageFileBytes(
				bytes, filepath.Join(path, entry.Name()),
			); err != nil {
				logger.Fatalf("unable parse localization file: %v", err)
			}
		}
	}
}
