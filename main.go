package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"

	todoHttp "github.com/kvalv/htmx-demo/adapters/http"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog"
)

func main() {
    log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
	srv := todoHttp.NewTodoHandler(ctx)
	go func() {
		port := "3000"
		if err := http.ListenAndServe(":"+port, srv); err != nil {
			log.Error().Err(err).Msg("failed to listen and serve")
            return
		}
		log.Info().Msg("Listening on port " + port)
	}()
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)
	<-done
    cancel()
	if err := srv.Shutdown(); err != nil {
		panic(err)
	}
	os.Exit(0)
}
