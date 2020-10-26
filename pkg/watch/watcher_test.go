package watch

import (
	"os"
	"testing"
	"time"

	forwardermocks "github.com/eko/monday/internal/tests/mocks/forward"
	runnermocks "github.com/eko/monday/internal/tests/mocks/run"
	"github.com/eko/monday/pkg/config"
	watcherlib "github.com/radovskyb/watcher"
	"github.com/stretchr/testify/assert"
)

func TestNewWatcher(t *testing.T) {
	// Given
	runner := &runnermocks.Runner{}
	forwarder := &forwardermocks.Forwarder{}

	project := getProjectMock()

	watcherConfig := &config.Watcher{
		Exclude: []string{"test-directory"},
	}

	// When
	w := NewWatcher(runner, forwarder, watcherConfig, project)

	// Then
	assert.IsType(t, new(watcher), w)
	assert.Implements(t, new(Watcher), w)

	assert.Equal(t, runner, w.runner)
	assert.Equal(t, forwarder, w.forwarder)
	assert.Equal(t, project, w.project)
	assert.Equal(t, excludeDirectories, []string{
		".git",
		"node_modules",
		"vendor",
		"test-directory",
	})

	assert.Len(t, w.fileWatchers, 0)
}

func TestWatch(t *testing.T) {
	// Given
	runner := &runnermocks.Runner{}
	runner.On("SetupAll").Once()
	runner.On("RunAll").Once()

	forwarder := &forwardermocks.Forwarder{}
	forwarder.On("ForwardAll").Once()

	project := getProjectMock()

	watcher := NewWatcher(runner, forwarder, &config.Watcher{}, project)

	// When - Then
	watcher.Watch()
}

func TestWatchWhenFileChange(t *testing.T) {
	// Given
	runner := &runnermocks.Runner{}
	runner.On("SetupAll").Once()
	runner.On("RunAll").Once()

	forwarder := &forwardermocks.Forwarder{}
	forwarder.On("ForwardAll").Once()

	project := getProjectMock()

	runner.On("Restart", project.Applications[0])

	watcher := NewWatcher(runner, forwarder, &config.Watcher{}, project)
	watcher.Watch()

	// When
	time.Sleep(time.Duration(1 * time.Second)) // Wait 1 second to be sure filesystem is watching

	dir, _ := os.Getwd()
	filepath := dir + "/../../internal/tests/watcher-test"

	// Create a file to trigger a file change and restart the application
	file, err := os.Create(filepath)
	if err != nil {
		t.Fatal(err)
	}
	file.Close()
	defer os.Remove(filepath)

	// Then
	assert.Len(t, watcher.fileWatchers, 1)

	if fileWatcher, ok := watcher.fileWatchers["test-app"]; ok {
		assert.IsType(t, new(watcherlib.Watcher), fileWatcher)
	} else {
		t.Fatal("Cannot find the fileWatcher concerning application test-app")
	}
}

func TestStop(t *testing.T) {
	// Given
	runner := &runnermocks.Runner{}
	runner.On("SetupAll").Once()
	runner.On("RunAll").Once()

	forwarder := &forwardermocks.Forwarder{}
	forwarder.On("ForwardAll").Once()

	project := getProjectMock()

	watcher := NewWatcher(runner, forwarder, &config.Watcher{}, project)
	watcher.Watch()

	// When - Then
	watcher.Stop()
}

func getProjectMock() *config.Project {
	dir, _ := os.Getwd()
	path := dir + "/../../internal/tests/"

	return &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			&config.Application{
				Name:  "test-app",
				Path:  path,
				Watch: true,
			},
		},
	}
}
