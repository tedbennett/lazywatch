package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tedbennett/lazywatch/config"
	"github.com/tedbennett/lazywatch/proxy"
	"github.com/tedbennett/lazywatch/watcher"
)

func main() {
	config, err := config.ParseConfig()
	if err != nil {
		panic(err)
	}

	watcher.StartWatching(config.Directory, func() { fmt.Println("File changed") })

	server := proxy.NewServer(config)

	go func() {
		err = server.ListenAndServe()
		if err != nil {
			fmt.Println(err)
		}
	}()

	// Cleanup
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	<-done
	fmt.Print("Shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	server.Shutdown(ctx)
}
