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

// TEMP : TO REMOVE

func main() {
	config, err := config.ParseConfig()
	if err != nil {
		panic(err)
	}

	events := make(chan interface{})
	healthCheck := command.NewHTTPHealthChecker(fmt.Sprintf("http://localhost:%s/health", config.Ports().Server), config.Client())
	runner := command.NewCommandRunner(config.Command, healthCheck)
	coordinator := command.NewCoordinator(runner)
	notifier := command.NewNotifier(coordinator, events)
	go coordinator.Listen(events)
	go runner.Start()

	// Invalidate the current command on FS changes
	go watcher.StartWatching(config.Directory, coordinator.Invalidate)

	server := proxy.NewServer(config, notifier)

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
	runner.Kill()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()
	server.Shutdown(ctx)
}
