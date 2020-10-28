package watch

import (
	"fmt"
	"os"
	"time"

	"github.com/eko/monday/pkg/build"
	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/forward"
	"github.com/eko/monday/pkg/run"
	radovskyb_watcher "github.com/radovskyb/watcher"
)

var (
	excludeDirectories = []string{".git", "node_modules", "vendor"}
)

type Watcher interface {
	Watch()
	Stop() error
}

// Watcher monitors health of the currently forwarded ports and launched applications.
type watcher struct {
	builder      build.Builder
	runner       run.Runner
	forwarder    forward.Forwarder
	conf         *config.Watcher
	project      *config.Project
	fileWatchers map[string]*radovskyb_watcher.Watcher
}

// NewWatcher initializes a watcher instance monitoring services using both runner and forwarder
func NewWatcher(builder build.Builder, runner run.Runner, forwarder forward.Forwarder, conf *config.Watcher, project *config.Project) *watcher {
	if conf != nil && len(conf.Exclude) > 0 {
		excludeDirectories = append(excludeDirectories, conf.Exclude...)
	}

	return &watcher{
		builder:      builder,
		runner:       runner,
		forwarder:    forwarder,
		conf:         conf,
		project:      project,
		fileWatchers: make(map[string]*radovskyb_watcher.Watcher, 0),
	}
}

// Watch runs both local applications and forwarded ones and ensure they keep running.
// It also relaunch them in case of file changes.
func (w *watcher) Watch() {
	w.runner.SetupAll()
	w.builder.BuildAll()

	go w.runner.RunAll()
	go w.forwarder.ForwardAll()

	for _, application := range w.project.Applications {
		if !application.Watch {
			continue
		}

		go w.watchApplication(application)
	}
}

// Stop stops all currently active file watchers on local running applications
func (w *watcher) Stop() error {
	for _, fileWatcher := range w.fileWatchers {
		fileWatcher.Close()
	}

	return nil
}

func (w *watcher) watchApplication(application *config.Application) error {
	fileWatcher := radovskyb_watcher.New()
	fileWatcher.SetMaxEvents(1)
	fileWatcher.IgnoreHiddenFiles(true)
	fileWatcher.FilterOps(radovskyb_watcher.Write, radovskyb_watcher.Create, radovskyb_watcher.Remove)

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
				fmt.Printf("👓  Watcher has detected a file change: %v\n", event)
				w.builder.Build(application)
				w.runner.Restart(application)
			case err := <-fileWatcher.Error:
				fmt.Printf("❌  An error has occured while file watching: %v\n", err)
			}
		}
	}()

	<-done

	return nil
}
