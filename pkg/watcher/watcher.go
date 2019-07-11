package watcher

import (
	"fmt"
	"time"

	"github.com/eko/monday/internal/config"
	"github.com/eko/monday/pkg/forwarder"
	"github.com/eko/monday/pkg/runner"
	"github.com/radovskyb/watcher"
)

// Watcher monitors health of the currently forwarded ports and launched applications.
type Watcher struct {
	runner       *runner.Runner
	forwarder    *forwarder.Forwarder
	project      *config.Project
	fileWatchers map[string]*watcher.Watcher
}

// NewWatcher initializes a watcher instance monitoring services using both runner and forwarder
func NewWatcher(runner *runner.Runner, forwarder *forwarder.Forwarder, project *config.Project) *Watcher {
	return &Watcher{
		runner:       runner,
		forwarder:    forwarder,
		project:      project,
		fileWatchers: make(map[string]*watcher.Watcher, 0),
	}
}

// Watch runs both local applications and forwarded ones and ensure they keep running.
// It also relaunch them in case of file changes.
func (w *Watcher) Watch() {
	w.forwarder.ForwardAll()
	w.runner.RunAll()

	for _, application := range w.project.Applications {
		if !application.Watch {
			continue
		}

		go w.watchApplication(application)
	}
}

func (w *Watcher) watchApplication(application *config.Application) error {
	fileWatcher := watcher.New()
	fileWatcher.SetMaxEvents(1)
	fileWatcher.FilterOps(watcher.Write, watcher.Create, watcher.Remove)

	if err := fileWatcher.AddRecursive(application.GetPath()); err != nil {
		fmt.Printf("❌  Unable to watch directory of application '%s': %v\n", application.Name, err)
	}

	go func() {
		_ = fileWatcher.Start(time.Millisecond * 100)
	}()

	done := make(chan bool)

	go func() {
		for {
			select {
			case event := <-fileWatcher.Event:
				fmt.Printf("👓  Watcher has detected a file change: %v", event)
				w.runner.Restart(application)
			case err := <-fileWatcher.Error:
				fmt.Printf("❌  An error has occured while file watching: %v", err)
			}
		}
	}()

	<-done

	return nil
}
