package main

import (
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"

	"github.com/robotomize/cribe/internal/bot"
	"github.com/robotomize/cribe/internal/buildinfo"
	"github.com/robotomize/cribe/internal/logging"
	"github.com/robotomize/cribe/internal/shutdown"
	"github.com/robotomize/cribe/internal/srvenv"
)

func main() {
	fmt.Fprintf(os.Stdout, buildinfo.Graffiti)
	_, _ = fmt.Fprintf(
		os.Stdout,
		buildinfo.GreetingCLI,
		buildinfo.Info.Tag(),
		buildinfo.Info.Time(),
		buildinfo.TgBloopURL,
		buildinfo.GithubBloopURL,
	)
	ctx, cancel := shutdown.New()
	defer cancel()
	env, err := srvenv.Setup(ctx)
	if err != nil {
		log.Fatalf("setup: %v", err)
	}

	cfg := env.Config()

	logger := logging.NewLogger(cfg.LogLevel).
		With("build_tag", buildinfo.Info.Tag()).
		With("build_time", buildinfo.Info.Time())
	ctx = logging.WithLogger(ctx, logger)

	defer env.AMQP().Close() //nolint

	mux := http.NewServeMux()
	mux.HandleFunc(
		"/health", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{"status": "ok"}`)
		},
	)
	mux.Handle("/debug/pprof/", http.Handler(http.DefaultServeMux))

	go func() {
		if err = http.ListenAndServe(cfg.Addr, mux); err != nil {
			logger.Errorf("listen and serve metrics: %v", err)
			cancel()
		}
	}()

	telegram := env.Telegram()
	dispatcher := bot.NewDispatcher(env)
	logger.Info("cribe-bot started")
	if err = dispatcher.Run(ctx, telegram, env.Config()); err != nil {
		logger.Fatalf("bot dispatcher: %v", err)
	}
}
