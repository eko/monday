package run

import (
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/log"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/ui"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewRunner(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	proxyfier := proxy.NewMockProxy(ctrl)

	project := getMockedProjectWithApplication()

	// When
	r := NewRunner(view, proxyfier, project)

	// Then
	assert.IsType(t, new(runner), r)
	assert.Implements(t, new(Runner), r)

	assert.Equal(t, proxyfier, r.proxy)
	assert.Equal(t, project.Name, r.projectName)
	assert.Equal(t, project.Applications, r.applications)
}

func TestRunAll(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	execCommand = mockExecCommand

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("🏁  Running local app '%s' (%s)...\n", "test-app", "/")
	view.EXPECT().Write(log.ColorGreen + "test-app" + log.ColorWhite + " OK Arguments Seems -to=work\n")

	proxyfier := proxy.NewMockProxy(ctrl)

	project := getMockedProjectWithApplication()

	runner := NewRunner(view, proxyfier, project)

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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	execCommand = mockExecCommand

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("🏁  Running local app '%s' (%s)...\n", "test-app", "/")
	view.EXPECT().Write(log.ColorGreen + "test-app" + log.ColorWhite + " OK Arguments Seems -to=work\n")

	proxyfier := proxy.NewMockProxy(ctrl)

	project := getMockedProjectWithApplication()

	runner := NewRunner(view, proxyfier, project)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	execCommand = mockExecCommand

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("⚙️  Please wait while setup of application '%s'...\n", "test-app")
	view.EXPECT().Writef("👉  Running commands:\n%s\n\n", "echo Starting test command setup...\necho ...and a second setup command to confirm it works")
	view.EXPECT().Write(log.ColorGreen + "test-app" + log.ColorWhite + " Starting test command setup...\n")
	view.EXPECT().Write(log.ColorGreen + "test-app" + log.ColorWhite + " ...and a second setup command to confirm it works\n")
	view.EXPECT().Write("\n✅  Setup complete!\n\n")

	proxyfier := proxy.NewMockProxy(ctrl)

	project := &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			{
				Name: "test-app",
				Path: "/unkown/directory",
				Setup: []string{
					"echo Starting test command setup...",
					"echo ...and a second setup command to confirm it works",
				},
			},
		},
	}

	runner := NewRunner(view, proxyfier, project)

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
			{
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
