// Package fsnotifydeep extends fsnotify to make it work recursively. During an Add,
// all subdirectories are added as well. For a Remove, the same. As the contents
// of the monitored directories changes, fsnotify is updated accordingly.
//
// In addition, fsnotifydeep adds a Filter feature to Watchers, allowing the client
// to specify a function literal that determines if an event should appear on the
// Events channel or be quietly dropped.
//
// The primary downside of this package is a performance hit. The exact amount
// will depend on the size of the filesystem under monitor.
package fsnotifydeep

import (
	"os"
	"path/filepath"

	"gopkg.in/fsnotify.v1"
)

// Watcher watches a set of files, delivering events to a channel.
type Watcher struct {
	Events chan fsnotify.Event
	Errors chan error

	internal *fsnotify.Watcher
	filter   FsnotifyFilter
	die      chan bool
}

// An FsnotifyFilter function accepts an event and returns true if that
// event passes the filter and should be accepted or false if the event
// should be dropped without notifying the client.
type FsnotifyFilter func(fsnotify.Event) bool

// NewWatcher establishes a new watcher, via fsnotify, begins waiting for events
// on the filesystem
func NewWatcher() (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{make(chan fsnotify.Event), make(chan error), w, nil, make(chan bool)}

	go func() {
		//defer watcher.Close()

		for {
			select {
			case evt := <-watcher.internal.Events:
				// Add/remove as needed
				// TODO check if we need to handle rename here too
				if evt.Op == fsnotify.Create {
					// For creates, make sure we add EVERYTHING under directories
					info, err := os.Stat(evt.Name)
					if err != nil {
						watcher.Errors <- err
					} else if info.IsDir() {
						// We already know it's a directory, so directly call addToWatch
						// Avoids an extra stat in Add
						if err := addToWatch(watcher.internal, evt.Name); err != nil {
							watcher.Errors <- err
						}
					}
				} else if evt.Op == fsnotify.Remove {
					if err := watcher.Remove(evt.Name); err != nil {
						watcher.Errors <- err
					}
				}

				// Pass to parent to handle as usual
				if watcher.filter == nil || watcher.filter(evt) {
					watcher.Events <- evt
				}
			case err := <-watcher.internal.Errors:
				watcher.Errors <- err
			case <-watcher.die:
				return
			}
		}
	}()

	return watcher, nil
}

// Add stops watching the named file or directory.
func (w *Watcher) Add(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if info.IsDir() {
		addToWatch(w.internal, path)
	} else {
		w.internal.Add(path)
	}

	return nil
}

// Close removes all watches and closes the events channel.
func (w *Watcher) Close() error {
	close(w.die)
	err := w.internal.Close()
	close(w.Events)
	close(w.Errors)
	return err
}

// Remove stops watching the named file or directory.
func (w *Watcher) Remove(path string) error {
	info, err := os.Stat(path)
	switch {
	case os.IsNotExist(err):
		// TODO not sure this is necessary, since it will also return IsNotExist errors
		//w.internal.Remove(path)
		return nil
	case err != nil:
		return err
	case info.IsDir():
		return removeFromWatch(w.internal, path)
	default:
		return w.internal.Remove(path)
	}
}

// Filter adds a filter to watcher, causing only events which pass the filter
// function to be presented on the Events channel. If a nil filter is used,
// all events will be published (this is the default for a new Watcher).
func (w *Watcher) Filter(filter FsnotifyFilter) {
	w.filter = filter
}

// Adds everything under the given path to the watcher
func addToWatch(watcher *fsnotify.Watcher, path string) error {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return watcher.Add(path)
		}
		return nil
	})

	return err
}

// Remove everything under the given path from the watcher
func removeFromWatch(watcher *fsnotify.Watcher, path string) error {
	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return watcher.Remove(path)
		}
		return nil
	})

	return err
}
