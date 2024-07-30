package command_test

import (
	"sync"
	"testing"

	"github.com/tedbennett/lazywatch/command"
)

func TestCommand(t *testing.T) {
	runner := TestRunner{t: t}
	events := make(chan interface{})
	coord := command.NewCoordinator(&runner)
	notif := command.NewNotifier(coord, events)
	coord.Invalidate()

	go coord.Listen(events)

	wg := sync.WaitGroup{}

	// Call wait a bunch of times
	wg.Add(100)
	for range 100 {
		go func() {
			notif.Wait()
			wg.Done()
		}()
	}
	wg.Wait()

	if runner.count != 1 {
		t.Fatalf("Invalidate called an unexpected number of times: %d", runner.count)
	}

}

type TestRunner struct {
	t *testing.T
	count int
}

func (tr *TestRunner) Start() error {
	tr.count += 1
	return nil
}

func (tr *TestRunner) Kill() error {
	return nil
}
