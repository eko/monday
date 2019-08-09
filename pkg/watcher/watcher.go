package watcher

import (
	"fmt"
	"os"
	"time"

	"github.com/policygenius/monday/pkg/config"
	"github.com/policygenius/monday/pkg/forwarder"
	"github.com/policygenius/monday/pkg/runner"
	"github.com/radovskyb/watcher"
)

var (
	excludeDirectories = []string{".git", "node_modules", "vendor"}
)

type WatcherInterface interface {
	Watch()
	Stop() error
}

// Watcher monitors health of the currently forwarded ports and launched applications.
type Watcher struct {
	runner       runner.RunnerInterface
	forwarder    forwarder.ForwarderInterface
	conf         *config.Watcher
	project      *config.Project
	fileWatchers map[string]*watcher.Watcher
}

// NewWatcher initializes a watcher instance monitoring services using both runner and forwarder
func NewWatcher(runner runner.RunnerInterface, forwarder forwarder.ForwarderInterface, conf *config.Watcher, project *config.Project) *Watcher {
	if conf != nil && len(conf.Exclude) > 0 {
		excludeDirectories = append(excludeDirectories, conf.Exclude...)
	}

	return &Watcher{
		runner:       runner,
		forwarder:    forwarder,
		conf:         conf,
		project:      project,
		fileWatchers: make(map[string]*watcher.Watcher, 0),
	}
}

// Watch runs both local applications and forwarded ones and ensure they keep running.
// It also relaunch them in case of file changes.
func (w *Watcher) Watch() {
	w.runner.SetupAll()
	w.runner.RunAll()

	w.forwarder.ForwardAll()

	for _, application := range w.project.Applications {
		if !application.Watch {
			continue
		}

		go w.watchApplication(application)
	}
}

// Stop stops all currently active file watchers on local running applications
func (w *Watcher) Stop() error {
	for _, fileWatcher := range w.fileWatchers {
		fileWatcher.Close()
	}

	return nil
}

func (w *Watcher) watchApplication(application *config.Application) error {
	fileWatcher := watcher.New()
	fileWatcher.SetMaxEvents(1)
	fileWatcher.IgnoreHiddenFiles(true)
	fileWatcher.FilterOps(watcher.Write, watcher.Create, watcher.Remove)

	w.fileWatchers[application.Name] = fileWatcher

	if err := fileWatcher.AddRecursive(application.GetPath()); err != nil {
		fmt.Printf("❌  Unable to watch directory of application '%s': %v\n", application.Name, err)
	}

	for _, directory := range excludeDirectories {
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			directory = fmt.Sprintf("%s/%s", application.GetPath(), directory)
		}

		fileWatcher.Ignore(directory)
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
