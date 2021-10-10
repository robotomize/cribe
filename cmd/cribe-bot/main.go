package main

import (
	"fmt"
	"log"
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

	logger := logging.NewLogger(cfg.LogLevel)
	ctx = logging.WithLogger(ctx, logger)

	defer env.RabbitMQ().Close() //nolint

	dispatcher := bot.NewDispatcher(env)
	if err := dispatcher.Run(ctx, env.Config()); err != nil {
		logger.Fatalf("bot dispatcher: %v", err)
	}
}
