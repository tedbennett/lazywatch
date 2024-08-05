package proxy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"

	"github.com/tedbennett/lazywatch/config"
)

type Waiter interface {
	Wait()
}

type ProxyConfig interface {
	Ports() config.Ports
	Client() *http.Client
}

func NewServer(cfg ProxyConfig, waiter Waiter) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/{path...}", proxyRequest)

	ctx := context.Background()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Ports().Proxy),
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			ctx = context.WithValue(ctx, "config", cfg)
			ctx = context.WithValue(ctx, "waiter", waiter)
			return ctx
		},
	}
	return server
}

func proxyRequest(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	config, ok := r.Context().Value("config").(ProxyConfig)
	if !ok {
		fmt.Fprint(w, "Failed to initialize config")
		return
	}
	waiter, ok := r.Context().Value("waiter").(Waiter)
	if !ok {
		fmt.Fprint(w, "Failed to initialize waiter")
		return
	}

	// If the command is invalidated, re-run it. Otherwise, progress immediately.
	waiter.Wait()

	url := fmt.Sprintf("http://localhost:%s/%s", config.Ports().Server, path)
	req, _ := http.NewRequest(r.Method, url, r.Body)
	req.Header = r.Header
	res, err := config.Client().Do(req)
	if err != nil {
		fmt.Println("Failed to call server")
		return
	}
	for name, value := range res.Header {
		if len(value) > 0 {
			w.Header().Set(name, value[0])
		}
	}
	w.WriteHeader(res.StatusCode)
	buf := make([]byte, 32*1024)
	for {
		n, err := res.Body.Read(buf)

		w.Write(buf[:n])

		if err != nil {
			// TODO: Handle errors better here / write to intermediate buffer
			if errors.Is(err, io.EOF) {
				break
			}
			fmt.Fprintf(w, "Unexpected read in response body")
		}
	}
}
