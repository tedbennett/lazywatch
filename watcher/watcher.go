package watcher

import (
	"fmt"

	"github.com/tedbennett/inotifywaitgo/inotifywaitgo"
)

func StartWatching(directory string, callback func()) {
	events, errors := initialize(directory)
	run(events, errors, callback)
}

func initialize(directory string) (chan inotifywaitgo.FileEvent, chan error) {

	events := make(chan inotifywaitgo.FileEvent)
	errors := make(chan error)

	go inotifywaitgo.WatchPath(&inotifywaitgo.Settings{
		Dir:        directory,
		FileEvents: events,
		ErrorChan:  errors,
		Options: &inotifywaitgo.Options{
			Recursive: true,
			Events: []inotifywaitgo.EVENT{
				inotifywaitgo.MODIFY,
			},
			Monitor: true,
		},
		Verbose: false,
	})

	return events, errors
}

func run(events chan inotifywaitgo.FileEvent, errors chan error, callback func()) {
	for {
		select {
		case event := <-events:
			for _, e := range event.Events {
				switch e {
				case inotifywaitgo.MODIFY:
					fallthrough
				case inotifywaitgo.CREATE:
					callback()
				}
			}

		case err := <-errors:
			fmt.Printf("Error: %s\n", err)
		}
	}
}
