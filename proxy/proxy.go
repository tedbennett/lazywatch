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


func NewServer(cfg *config.Config) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/*", proxyRequest)


	ctx := context.Background()
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", cfg.Ports.Proxy),
		Handler: mux,
		BaseContext: func(l net.Listener) context.Context {
			ctx = context.WithValue(ctx, "config", cfg)
			return ctx
		},
	}
	return server
}

func proxyRequest(w http.ResponseWriter, r *http.Request) {
	config, ok := r.Context().Value("config").(*config.Config)
	if !ok {
		fmt.Fprint(w, "Failed to initialize config")
		return
	}


	req, _ := http.NewRequest(r.Method, fmt.Sprintf("http://localhost:%s%s", config.Ports.Server, r.URL.RawPath), r.Body)
	res, err := config.Client.Do(req)
	if err != nil {
		fmt.Println("Failed to call server")
		return
	}
	for name, value := range res.Header {
		if len(value) > 0 {
			w.Header().Set(name, value[0])
		}
	}
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
