package run

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	mocks "github.com/eko/monday/internal/tests/mocks/proxy"
	uimocks "github.com/eko/monday/internal/tests/mocks/ui"
	"github.com/eko/monday/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNewRunner(t *testing.T) {
	// Given
	view := &uimocks.View{}
	proxy := &mocks.Proxy{}

	project := getMockedProjectWithApplication()

	// When
	r := NewRunner(view, proxy, project)

	// Then
	assert.IsType(t, new(runner), r)
	assert.Implements(t, new(Runner), r)

	assert.Equal(t, proxy, r.proxy)
	assert.Equal(t, project.Name, r.projectName)
	assert.Equal(t, project.Applications, r.applications)
}

func TestRunAll(t *testing.T) {
	// Given
	execCommand = mockExecCommand

	view := &uimocks.View{}
	view.On("Write", mock.Anything)
	view.On("Writef", mock.Anything, mock.Anything, mock.Anything)

	proxy := &mocks.Proxy{}

	project := getMockedProjectWithApplication()

	runner := NewRunner(view, proxy, project)

	// When
	runner.RunAll()

	// Then
	// Wait for goroutine to launch application and be available
	for i := 0; i < 50; i++ {
		if _, ok := runner.cmds["test-app"]; ok {
			break
		}

		time.Sleep(time.Duration(100 * time.Millisecond))
	}

	// Check for application to be runned properly
	if cmd, ok := runner.cmds["test-app"]; ok {
		runCommand := strings.Replace(strings.Join(cmd.Args, " "), "echo <runner>", "runner", -1)
		assert.Equal(t, "echo OK Arguments Seems -to=work", runCommand)
	} else {
		t.Fatal("Cannot retrieve just launched application command execution")
	}
}

func TestStop(t *testing.T) {
	// Given
	execCommand = mockExecCommand

	view := &uimocks.View{}
	view.On("Write", mock.Anything)
	view.On("Writef", mock.Anything, mock.Anything, mock.Anything)

	proxy := &mocks.Proxy{}

	project := getMockedProjectWithApplication()

	runner := NewRunner(view, proxy, project)
	runner.RunAll()

	// Wait for goroutine to launch application and be available
	for i := 0; i < 50; i++ {
		if _, ok := runner.cmds["test-app"]; ok {
			break
		}

		time.Sleep(time.Duration(100 * time.Millisecond))
	}

	// When
	runner.Stop()

	// Then
	if cmd, ok := runner.cmds["test-app"]; ok {
		runCommand := strings.Replace(strings.Join(cmd.Args, " "), "echo <runner>", "runner", -1)
		assert.Equal(t, "echo OK Arguments Seems -to=work", runCommand)

		assert.True(t, cmd.ProcessState.Exited())
	} else {
		t.Fatal("Cannot retrieve just launched application command execution")
	}
}

func TestSetupAll(t *testing.T) {
	// Given
	execCommand = mockExecCommand

	view := &uimocks.View{}
	view.On("Write", mock.Anything)
	view.On("Writef", mock.Anything, mock.Anything, mock.Anything)

	proxy := &mocks.Proxy{}

	project := &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			&config.Application{
				Name: "test-app",
				Path: "/unkown/directory",
				Setup: []string{
					"echo Starting test command setup...",
					"echo ...and a second setup command to confirm it works",
				},
			},
		},
	}

	runner := NewRunner(view, proxy, project)

	// When
	runner.SetupAll()

	// Then
	assert.True(t, hasSetup)
}

func mockExecCommand(command string, args ...string) *exec.Cmd {
	args = append([]string{"<runner>"}, args...)
	return exec.Command("echo", args...)
}

func getMockedProjectWithApplication() *config.Project {
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
			},
		},
	}
}
