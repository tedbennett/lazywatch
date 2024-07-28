package config

import (
	"flag"
	"fmt"
	"net/http"
	"regexp"
)

type Config struct {
	Ports  Ports
	Client *http.Client
}

type Ports struct {
	Proxy  string
	Server string
}

func ParseConfig() (*Config, error) {
	portsArg := flag.String("p", "3001:3000", "proxyPort:serverPort")

	flag.Parse()

	ports, err := parsePorts(*portsArg)
	if err != nil {
		return nil, err
	}
	return &Config{Ports: *ports, Client: http.DefaultClient}, nil

}

func parsePorts(arg string) (*Ports, error) {
	re := regexp.MustCompile(`(\d+):(\d+)`)
	matches := re.FindStringSubmatch(arg)
	if len(matches) == 0 {
		return nil, fmt.Errorf("Failed to parse port argument")
	}
	return &Ports{
		Proxy:  matches[1],
		Server: matches[2],
	}, nil
}

func NewConfig(proxy string, server string) *Config {
	return &Config{Ports: Ports{proxy, server}}
}
