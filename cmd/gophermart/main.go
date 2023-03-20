package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"gofermart/internal/server"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app := server.NewApp()
	if err := app.Run(ctx); err != nil {
		log.Fatalf("%s", err.Error())
	}

}
