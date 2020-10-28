package helper

import (
	"os"
	"os/exec"
	"testing"

	"github.com/eko/monday/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestAddEnvVariables(t *testing.T) {
	// Given
	application := getMockedApplication()

	cmd := &exec.Cmd{}

	// When
	AddEnvVariables(cmd, application.Env)

	// Then
	assert.Contains(t, cmd.Env, "MY_ENVVAR_1=value")
	assert.Contains(t, cmd.Env, "MY_ENVVAR_2=My custom second value")
}

func TestAddEnvVariablesFromFile(t *testing.T) {
	// Given
	application := getMockedApplication()

	cmd := &exec.Cmd{}

	// When
	AddEnvVariablesFromFile(cmd, application.EnvFile)

	// Then
	assert.Contains(t, cmd.Env, "MY_ENVFILE_VAR_1=this is ok")
	assert.Contains(t, cmd.Env, "MY_ENVFILE_VAR_2=this is really good")
	assert.Contains(t, cmd.Env, "MY_ENVFILE_VAR_3=great")
}

func getMockedApplication() *config.Application {
	dir, _ := os.Getwd()

	return &config.Application{
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
	}
}
