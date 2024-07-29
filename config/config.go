package config

import (
	"flag"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/tedbennett/lazywatch/command"
)

type Config struct {
	ports     Ports
	Directory string
	client    *http.Client
	Command   command.Command
}

func (c *Config) Ports() Ports {
	return c.ports
}

func (c *Config) Client() *http.Client {
	return c.client
}

type Ports struct {
	Proxy  string
	Server string
}

func ParseConfig() (*Config, error) {
	portsArg := flag.String("p", "3001:3000", "proxyPort:serverPort")
	dirArg := flag.String("d", "./", "directory to watch")
	cmdArg := flag.String("c", "go run main.go", "command to run")

	flag.Parse()

	ports, err := parsePorts(*portsArg)
	if err != nil {
		return nil, err
	}
	cmd := parseCommand(*cmdArg)

	return &Config{
		ports:     *ports,
		client:    http.DefaultClient,
		Directory: *dirArg,
		Command:   cmd,
	}, nil

}

func parseCommand(arg string) command.Command {
	slice := strings.Split(arg, " ")
	return command.NewCommand(slice[0], slice[1:])
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
