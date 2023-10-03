package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

    todoHttp "github.com/kvalv/htmx-demo/adapters/http"
)

func main() {
	srv := todoHttp.NewTodoHandler()
	go func() {
		port := "3000"
		fmt.Println("Listening on port " + port)
		if err := http.ListenAndServe(":"+port, srv); err != nil {
			log.Println(err)
		}
	}()
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt)
	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		panic(err)
	}
	os.Exit(0)
}
