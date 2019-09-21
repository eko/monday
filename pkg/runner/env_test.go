package runner

import (
	"os"
	"testing"

	mocks "github.com/eko/monday/internal/tests/mocks/proxy"
	uimocks "github.com/eko/monday/internal/tests/mocks/ui"
	"github.com/eko/monday/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAddEnvVariables(t *testing.T) {
	// Given
	view := &uimocks.ViewInterface{}
	view.On("Write", mock.Anything)
	view.On("Writef", mock.Anything, mock.Anything, mock.Anything)

	proxy := &mocks.ProxyInterface{}

	project := getMockedProjectWithApplicationEnv()

	runner := NewRunner(view, proxy, project)

	// When
	runner.Run(project.Applications[0])

	// Then
	assert.IsType(t, new(Runner), runner)
	assert.Len(t, runner.cmds, 1)

	cmd := runner.cmds["test-app"]

	assert.Contains(t, cmd.Env, "MY_ENVVAR_1=value")
	assert.Contains(t, cmd.Env, "MY_ENVVAR_2=My custom second value")
}

func TestAddEnvVariablesFromFile(t *testing.T) {
	// Given
	view := &uimocks.ViewInterface{}
	view.On("Write", mock.Anything)
	view.On("Writef", mock.Anything, mock.Anything, mock.Anything)

	proxy := &mocks.ProxyInterface{}

	project := getMockedProjectWithApplicationEnv()

	runner := NewRunner(view, proxy, project)

	// When
	runner.Run(project.Applications[0])

	// Then
	assert.IsType(t, new(Runner), runner)
	assert.Len(t, runner.cmds, 1)

	cmd := runner.cmds["test-app"]

	assert.Contains(t, cmd.Env, "MY_ENVFILE_VAR_1=this is ok")
	assert.Contains(t, cmd.Env, "MY_ENVFILE_VAR_2=this is really good")
	assert.Contains(t, cmd.Env, "MY_ENVFILE_VAR_3=great")
}

func getMockedProjectWithApplicationEnv() *config.Project {
	dir, _ := os.Getwd()

	return &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			&config.Application{
				Name:       "test-app",
				Path:       "/",
				Executable: "echo",
				Args: []string{
					"OK",
					"Arguments",
					"Seems",
					"-to=work",
				},
				Env: map[string]string{
					"MY_ENVVAR_1": "value",
					"MY_ENVVAR_2": "My custom second value",
				},
				EnvFile: dir + "/../../internal/tests/runner/test.env",
			},
		},
	}
}
