package watcher

import (
	"testing"

	"github.com/eko/monday/internal/config"
	"github.com/eko/monday/internal/tests/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewWatcher(t *testing.T) {
	// Given
	runner := &mocks.RunnerInterface{}
	forwarder := &mocks.ForwarderInterface{}

	project := &config.Project{
		Name: "My project name",
		Forwards: []*config.Forward{
			&config.Forward{
				Name: "test-kubernetes-forward",
				Type: "kubernetes",
				Values: config.ForwardValues{
					Namespace: "test",
					Labels: map[string]string{
						"app": "my-test-app",
					},
				},
			},
		},
	}

	watcherConfig := &config.Watcher{
		Exclude: []string{"test-directory"},
	}

	// When
	watcher := NewWatcher(runner, forwarder, watcherConfig, project)

	// Then
	assert.IsType(t, new(Watcher), watcher)

	assert.Equal(t, runner, watcher.runner)
	assert.Equal(t, forwarder, watcher.forwarder)
	assert.Equal(t, project, watcher.project)

	assert.Len(t, watcher.fileWatchers, 0)
}

func newWatcher(watcherConfig *config.Watcher, project *config.Project) *Watcher {
	runner := &mocks.RunnerInterface{}
	forwarder := &mocks.ForwarderInterface{}

	return NewWatcher(runner, forwarder, watcherConfig, project)
}
