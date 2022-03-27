package watch

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/eko/monday/pkg/build"
	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/forward"
	"github.com/eko/monday/pkg/run"
	"github.com/eko/monday/pkg/setup"
	"github.com/eko/monday/pkg/write"
	"github.com/golang/mock/gomock"
	watcherlib "github.com/radovskyb/watcher"
	"github.com/stretchr/testify/assert"
)

func TestNewWatcher(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	setuper := setup.NewMockSetuper(ctrl)
	builder := build.NewMockBuilder(ctrl)
	writer := write.NewMockWriter(ctrl)
	runner := run.NewMockRunner(ctrl)
	forwarder := forward.NewMockForwarder(ctrl)

	project := getProjectMock()

	watchConfig := &config.GlobalWatch{
		Exclude: []string{"test-directory"},
	}

	// When
	w := NewWatcher(setuper, builder, writer, runner, forwarder, watchConfig, project)

	// Then
	assert.IsType(t, new(watcher), w)
	assert.Implements(t, new(Watcher), w)

	assert.Equal(t, writer, w.writer)
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
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	setuper := setup.NewMockSetuper(ctrl)
	setuper.EXPECT().SetupAll().Times(1)

	builder := build.NewMockBuilder(ctrl)
	builder.EXPECT().BuildAll().Times(1)

	writer := write.NewMockWriter(ctrl)
	writer.EXPECT().WriteAll().Times(1)

	runner := run.NewMockRunner(ctrl)
	runner.EXPECT().RunAll().Times(1)

	forwarder := forward.NewMockForwarder(ctrl)
	forwarder.EXPECT().ForwardAll(ctx).Times(1)

	project := getProjectMock()

	dir, _ := os.Getwd()
	writerDirectory := dir + "/../../internal/test/write"

	watcher := NewWatcher(setuper, builder, writer, runner, forwarder, &config.GlobalWatch{
		Exclude: []string{writerDirectory},
	}, project)

	// When - Then
	watcher.Watch(ctx)

	// Wait 1 second to ensure the watch is effective
	time.Sleep(1 * time.Second)
}

func TestWatchWhenFileChange(t *testing.T) {
	// Given
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	setuper := setup.NewMockSetuper(ctrl)
	setuper.EXPECT().SetupAll().Times(1)

	builder := build.NewMockBuilder(ctrl)
	builder.EXPECT().BuildAll().Times(1)

	writer := write.NewMockWriter(ctrl)
	writer.EXPECT().WriteAll().Times(1)

	runner := run.NewMockRunner(ctrl)
	runner.EXPECT().RunAll().Times(1)

	forwarder := forward.NewMockForwarder(ctrl)
	forwarder.EXPECT().ForwardAll(ctx).Times(1)

	project := getProjectMock()

	watcher := NewWatcher(setuper, builder, writer, runner, forwarder, &config.GlobalWatch{}, project)
	watcher.Watch(ctx)

	// When
	time.Sleep(time.Duration(1 * time.Second)) // Wait 1 second to be sure filesystem is watching

	dir, _ := os.Getwd()
	filepath := dir + "/../../internal/test/watcher-test"

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	setuper := setup.NewMockSetuper(ctrl)
	builder := build.NewMockBuilder(ctrl)
	writer := write.NewMockWriter(ctrl)
	runner := run.NewMockRunner(ctrl)
	forwarder := forward.NewMockForwarder(ctrl)

	project := getProjectMock()

	watcher := NewWatcher(setuper, builder, writer, runner, forwarder, &config.GlobalWatch{}, project)

	// When - Then
	watcher.Stop()
}

func getProjectMock() *config.Project {
	dir, _ := os.Getwd()
	path := dir + "/../../internal/test/"

	return &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			{
				Name:  "test-app",
				Path:  path,
				Watch: true,
			},
		},
	}
}
