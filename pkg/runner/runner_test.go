package runner

import (
	"testing"

	"github.com/eko/monday/internal/config"
	"github.com/eko/monday/internal/tests/mocks"
	"github.com/stretchr/testify/assert"
)

func TestNewRunner(t *testing.T) {
	// Given
	proxy := &mocks.ProxyInterface{}

	project := &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			&config.Application{
				Name:       "test-app",
				Path:       "/dev/null",
				Executable: "echo",
				Args: []string{
					"OK",
				},
			},
		},
	}

	// When
	runner := NewRunner(proxy, project)

	// Then
	assert.IsType(t, new(Runner), runner)

	assert.Equal(t, proxy, runner.proxy)
	assert.Equal(t, project.Name, runner.projectName)
	assert.Equal(t, project.Applications, runner.applications)
}

func newRunner(project *config.Project) *Runner {
	proxy := &mocks.ProxyInterface{}

	return NewRunner(proxy, project)
}
