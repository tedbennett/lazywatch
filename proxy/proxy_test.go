package proxy_test

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/tedbennett/lazywatch/config"
	"github.com/tedbennett/lazywatch/proxy"
)

// Global vars
var proxyServer *http.Server
var proxiedServer *http.Server

const (
	PROXY_PORT  = "9998"
	SERVER_PORT = "9999"
)

func setup() {
	waiter := &MockWaiter{}
	config := &MockConfig{client: http.DefaultClient}
	proxyServer = proxy.NewServer(config, waiter)
	proxiedServer = MockServer(fmt.Sprintf(":%s", SERVER_PORT))
	go proxyServer.ListenAndServe()
	go proxiedServer.ListenAndServe()
}

func TestMain(m *testing.M) {
	setup()
	os.Exit(m.Run())
}

func TestProxyServerRoot(t *testing.T) {
	res, err := http.Get(fmt.Sprintf("http://localhost:%s/", SERVER_PORT))
	if err != nil {
		t.Fatalf("Failed to fetch /: %s", err)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %s", err)
	}
	if res.StatusCode != 200 {
		t.Fatalf("Getting root failed: %s", resBody)
	}

	if string(resBody) != "Root" {
		t.Fatalf("Unexpected body in get /: %s", resBody)
	}
}

func TestProxyServerWildcard(t *testing.T) {
	tests := []string{"", "wildcard", "very/long/path/123"}

	for _, test := range tests {

		res, err := http.Get(fmt.Sprintf("http://localhost:%s/wildcard/%s", SERVER_PORT, test))
		if err != nil {
			t.Fatalf("Failed to fetch /: %s", err)
		}
		resBody, err := io.ReadAll(res.Body)
		if err != nil {
			t.Fatalf("Failed to read body: %s", err)
		}
		if res.StatusCode != 200 {
			t.Fatalf("Getting root failed: %s", resBody)
		}

		if string(resBody) != test {
			t.Fatalf("Unexpected body in get /%s: %s", test, resBody)
		}
	}
}

func TestProxyServerError(t *testing.T) {
	res, err := http.Get(fmt.Sprintf("http://localhost:%s/error", SERVER_PORT))
	if err != nil {
		t.Fatalf("Failed to fetch /: %s", err)
	}
	resBody, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %s", err)
	}
	if res.StatusCode != 500 {
		t.Fatalf("Getting root failed: %s", resBody)
	}

	if string(resBody) != "Error\n" {
		t.Fatalf("Unexpected body in get /error: %s", string(resBody))
	}
}

// Mocks

func MockServer(addr string) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("Root")) })
	mux.HandleFunc("/wildcard/{path...}", func(w http.ResponseWriter, r *http.Request) {
		path := r.PathValue("path")
		w.Write([]byte(path))
	})
	mux.HandleFunc("/error", func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Error", http.StatusInternalServerError)
	})

	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return server
}

type MockWaiter struct{}

func (*MockWaiter) Wait() { return }

type MockConfig struct{ client *http.Client }

func (*MockConfig) Ports() config.Ports     { return config.Ports{Server: SERVER_PORT, Proxy: PROXY_PORT} }
func (mc *MockConfig) Client() *http.Client { return mc.client }
