package main

import (
	"github.com/robotomize/cribe/internal/bot"
	"github.com/robotomize/cribe/internal/logging"
	"github.com/robotomize/cribe/internal/shutdown"
	"github.com/robotomize/cribe/internal/srvenv"
)

func main() {
	ctx, cancel := shutdown.New()
	defer cancel()

	logger := logging.FromContext(ctx)
	env, err := srvenv.Setup(ctx)
	if err != nil {
		logger.Fatalf("setup: %v", err)
	}

	defer env.RabbitMQ().Close()

	dispatcher := bot.NewDispatcher(env)
	if err := dispatcher.Run(ctx, env.Config()); err != nil {
		logger.Fatalf("bot dispatcher: %v", err)
	}
}
