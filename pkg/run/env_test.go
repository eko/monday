package run

import (
	"os"
	"testing"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/ui"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestAddEnvVariables(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("⚙️   Running local app '%s' (%s)...\n", "test-app", "/")
	view.EXPECT().Write(ColorGreen + "test-app" + ColorWhite + " OK Arguments Seems -to=work\n")

	proxyfier := proxy.NewMockProxy(ctrl)

	project := getMockedProjectWithApplicationEnv()

	r := NewRunner(view, proxyfier, project)

	// When
	r.Run(project.Applications[0])

	// Then
	assert.IsType(t, new(runner), r)
	assert.Implements(t, new(Runner), r)

	assert.Len(t, r.cmds, 1)

	cmd := r.cmds["test-app"]

	assert.Contains(t, cmd.Env, "MY_ENVVAR_1=value")
	assert.Contains(t, cmd.Env, "MY_ENVVAR_2=My custom second value")
}

func TestAddEnvVariablesFromFile(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("⚙️   Running local app '%s' (%s)...\n", "test-app", "/")
	view.EXPECT().Write(ColorGreen + "test-app" + ColorWhite + " OK Arguments Seems -to=work\n")

	proxyfier := proxy.NewMockProxy(ctrl)

	project := getMockedProjectWithApplicationEnv()

	r := NewRunner(view, proxyfier, project)

	// When
	r.Run(project.Applications[0])

	// Then
	assert.IsType(t, new(runner), r)
	assert.Implements(t, new(Runner), r)

	assert.Len(t, r.cmds, 1)

	cmd := r.cmds["test-app"]

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
