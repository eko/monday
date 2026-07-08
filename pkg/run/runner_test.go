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
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewRunner(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	proxyfier := proxy.NewMockProxy(ctrl)

	project := getMockedProjectWithApplication()

	// When
	r := NewRunner(view, proxyfier, project, &config.GlobalRun{})

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

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("🏁  Running local app '%s' (%s)...\n", "test-app", "/")
	view.EXPECT().Write(log.Prefix(log.StdOut, "test-app") + "OK Arguments Seems -to=work\n")

	proxyfier := proxy.NewMockProxy(ctrl)

	project := getMockedProjectWithApplication()

	runner := NewRunner(view, proxyfier, project, &config.GlobalRun{})

	// When
	runner.RunAll()

	// Then
	// Wait for goroutine to launch application and be available
	cmd := waitForCommand(runner, "test-app")

	// Check for application to be runned properly
	if cmd == nil {
		t.Fatal("Cannot retrieve just launched application command execution")
	}

	runCommand := strings.Replace(strings.Join(cmd.Args, " "), "echo <runner>", "runner", -1)
	assert.Equal(t, "/bin/sh -c echo OK Arguments Seems -to=work", runCommand)
}

// waitForCommand waits for the application command to be registered by the
// runner goroutine, reading the commands map under its lock
func waitForCommand(r *runner, name string) *exec.Cmd {
	for i := 0; i < 50; i++ {
		r.mux.Lock()
		cmd, ok := r.cmds[name]
		r.mux.Unlock()

		if ok {
			return cmd
		}

		time.Sleep(time.Duration(100 * time.Millisecond))
	}

	return nil
}

func TestStop(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("🏁  Running local app '%s' (%s)...\n", "test-app", "/")
	view.EXPECT().Write(log.Prefix(log.StdOut, "test-app") + "OK Arguments Seems -to=work\n")

	// The killed application may report its termination before the test ends
	view.EXPECT().Writef(gomock.Any(), gomock.Any()).AnyTimes()

	proxyfier := proxy.NewMockProxy(ctrl)

	project := getMockedProjectWithApplication()

	runner := NewRunner(view, proxyfier, project, &config.GlobalRun{})
	runner.RunAll()

	// Wait for goroutine to launch application and be available
	cmd := waitForCommand(runner, "test-app")
	if cmd == nil {
		t.Fatal("Cannot retrieve just launched application command execution")
	}

	// When
	assert.Nil(t, runner.Stop())

	// Then: the application is stopped and removed from the active commands
	runCommand := strings.Replace(strings.Join(cmd.Args, " "), "echo <runner>", "runner", -1)
	assert.Equal(t, "/bin/sh -c echo OK Arguments Seems -to=work", runCommand)

	runner.mux.Lock()
	defer runner.mux.Unlock()
	assert.NotContains(t, runner.cmds, "test-app")
}

// TestStopWhenApplicationFailedToStart is a regression test for a nil pointer
// dereference crash: stopping an application whose process never started (e.g.
// an invalid command) must not panic
func TestStopWhenApplicationFailedToStart(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	proxyfier := proxy.NewMockProxy(ctrl)

	project := getMockedProjectWithApplication()

	runner := NewRunner(view, proxyfier, project, &config.GlobalRun{})

	// The command has never been started: its process is nil
	runner.cmds["test-app"] = exec.Command("/nonexistent-binary")

	// When / Then
	assert.NotPanics(t, func() {
		assert.Nil(t, runner.Stop())
	})

	assert.NotContains(t, runner.cmds, "test-app")
}

// TestStopWhenApplicationHasNoRunSection ensures stopping an application
// without a 'run' configuration section does not panic
func TestStopWhenApplicationHasNoRunSection(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	proxyfier := proxy.NewMockProxy(ctrl)

	project := &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			{Name: "no-run-app", Path: "/"},
		},
	}

	runner := NewRunner(view, proxyfier, project, &config.GlobalRun{})

	// When / Then
	assert.NotPanics(t, func() {
		assert.Nil(t, runner.Stop())
	})
}

func getMockedProjectWithApplication() *config.Project {
	return &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			{
				Name: "test-app",
				Path: "/",
				Run: &config.Run{
					Command: "echo OK Arguments Seems -to=work",
				},
			},
		},
	}
}
