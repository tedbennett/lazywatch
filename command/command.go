package command

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Worker struct {
}

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
	return cmd
}

type CommandRunner struct {
	mu *sync.Mutex
	c  *sync.Cond

	command Command
	process *os.Process
	healthy func() bool

	invalidated bool
}

type CommandWaiter struct {
	mu *sync.Mutex
	c  *sync.Cond

	events chan<- interface{}
}

func NewRunnerAndWaiter(command Command, events chan interface{}, healthy func() bool) (*CommandRunner, *CommandWaiter) {
	mu := &sync.Mutex{}
	c := sync.NewCond(mu)

	runner := &CommandRunner{
		mu:          mu,
		c:           c,
		command:     command,
		process:     nil,
		healthy:     healthy,
		invalidated: false,
	}

	waiter := &CommandWaiter{
		mu:     mu,
		c:      c,
		events: events,
	}

	return runner, waiter
}

func (cw *CommandWaiter) Wait() {
	cw.mu.Lock()
	// Tell the Runner to start building if needed
	cw.events <- nil
	// Wait for server to be up
	cw.c.Wait()
	cw.mu.Unlock()
}

func (cr *CommandRunner) StartCommand() {
	cmd := newCmd(cr.command)

	cmd.Start()
	cr.process = cmd.Process
	cmd.Wait()
}

func (cr *CommandRunner) KillCommand() error {
	if cr.process != nil {
		if err := syscall.Kill(-cr.process.Pid, syscall.SIGINT); err != nil {
			return fmt.Errorf("Failed to kill process group")
		}
	}
	return nil
}

// Take the lock and mark the current command as invalidated
func (cr *CommandRunner) Invalidate() {
	cr.mu.Lock()
	fmt.Println("Changes made...")
	cr.invalidated = true
	cr.mu.Unlock()
}

func (cr *CommandRunner) HandleEvent() {
	cr.mu.Lock()
	if cr.invalidated {
		// Kill the previous process and it's children
		if cr.process != nil {
			cr.KillCommand()
		}

		// Start the new process
		go cr.StartCommand()

		// Wait until the command is healthy
		for {
			if cr.healthy() {
				break
			}
			time.Sleep(time.Millisecond * 50)
		}
		// Notify listeners that the command has been restarted
		cr.invalidated = false
	}
	cr.c.Broadcast()
	cr.mu.Unlock()
}

func (cr *CommandRunner) Listen(events <-chan interface{}) {
	for range events {
		cr.HandleEvent()
	}
}
