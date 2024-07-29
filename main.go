package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tedbennett/lazywatch/command"
	"github.com/tedbennett/lazywatch/config"
	"github.com/tedbennett/lazywatch/proxy"
	"github.com/tedbennett/lazywatch/watcher"
)

func main() {
	config, err := config.ParseConfig()
	if err != nil {
		panic(err)
	}

	events := make(chan interface{})
	health := func() bool { return true }
	runner, waiter := command.NewRunnerAndWaiter(config.Command, events, health)
	go runner.Listen(events)
	go runner.StartCommand()

	// Invalidate the current command on FS changes
	go watcher.StartWatching(config.Directory, runner.Invalidate)

	server := proxy.NewServer(config, waiter)

	go func() {
		fmt.Printf("Listening on port %s\n", config.Ports().Proxy)
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
	runner.KillCommand()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	server.Shutdown(ctx)
}
