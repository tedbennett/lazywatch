package command

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Command struct {
	executable string
	args       []string
}

func NewCommand(executable string, args []string) Command {
	return Command{executable: executable, args: args}
}

func newCmd(command Command) *exec.Cmd {
	cmd := exec.Command(command.executable, command.args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

// Describes a struct that keeps
type TaskRunner interface {
	// Starts a new task
	// If a previous one exists, it should stop that first
	Start() error
	Kill() error
}

type Coordinator struct {
	mu *sync.Mutex
	c  *sync.Cond

	runner      TaskRunner
	invalidated bool

	errors chan error
	logger chan string
}

type Notifier struct {
	mu *sync.Mutex
	c  *sync.Cond

	events chan<- interface{}
}

func NewCoordinator(runner TaskRunner) *Coordinator {
	mu := &sync.Mutex{}
	return &Coordinator{
		mu:          mu,
		c:           sync.NewCond(mu),
		runner:      runner,
		invalidated: false,
	}
}

func NewNotifier(coord *Coordinator, events chan<- interface{}) *Notifier {
	return &Notifier{
		mu:     coord.mu,
		c:      coord.c,
		events: events,
	}
}

func (cw *Notifier) Wait() {
	// Tell the Runner to start building if needed
	cw.events <- nil
	// Lock after sending event due to possible deadlock where we're
	// waiting for the lock and can't send an event
	cw.mu.Lock()
	// Wait for server to be up
	cw.c.Wait()
	cw.mu.Unlock()
}

type CommandRunner struct {
	process     *os.Process
	command     Command
	healthcheck HealthChecker
}

func NewCommandRunner(command Command, healthcheck HealthChecker) *CommandRunner {
	return &CommandRunner{
		command:     command,
		healthcheck: healthcheck,
		process:     nil,
	}
}

func (cr *CommandRunner) Start() error {
	err := cr.Kill()
	if err != nil {
		return err
	}
	go func() {
		cmd := newCmd(cr.command)

		cmd.Start()
		cr.process = cmd.Process
		cmd.Wait()
	}()

	// Wait until the command is healthy
	for {
		// Sleep first since the killed server might still be listening
		time.Sleep(time.Millisecond * 50)
		if cr.healthcheck.Check() {
			fmt.Println("Healthy")
			break
		}
		fmt.Println("Not healthy")
	}
	return nil
}

func (cr *CommandRunner) Kill() error {
	if cr.process != nil {
		if err := syscall.Kill(-cr.process.Pid, syscall.SIGINT); err != nil {
			return fmt.Errorf("Failed to kill process group")
		}
	}
	return nil
}

// Take the lock and mark the current command as invalidated
func (cr *Coordinator) Invalidate() {
	cr.mu.Lock()
	fmt.Println("Changes made...")
	cr.invalidated = true
	cr.mu.Unlock()
}

func (cr *Coordinator) HandleEvent() {
	cr.mu.Lock()
	if cr.invalidated {
		fmt.Println("Restarting server")
		cr.runner.Start()
		// Notify listeners that the command has been restarted
		cr.invalidated = false
	}
	cr.c.Broadcast()
	cr.mu.Unlock()
}

func (cr *Coordinator) Listen(events <-chan interface{}) {
	for range events {
		cr.HandleEvent()
	}
}

type HealthChecker interface {
	Check() bool
}

type HTTPHealthChecker struct {
	url    string
	client *http.Client
}

func NewHTTPHealthChecker(url string, client *http.Client) *HTTPHealthChecker {
	return &HTTPHealthChecker{url, client}
}

func (hc *HTTPHealthChecker) Check() bool {
	res, err := hc.client.Get(hc.url)
	if err == nil && res.StatusCode == http.StatusOK {
		return true
	}
	return false
}
