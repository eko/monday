package run

import (
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/helper"
	"github.com/eko/monday/pkg/log"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/ui"
)

var (
	hasSetup = false
)

type Runner interface {
	RunAll()
	SetupAll()
	Run(application *config.Application)
	Restart(application *config.Application)
	Stop() error
}

// runner is the struct that manage running local applications
type runner struct {
	proxy        proxy.Proxy
	projectName  string
	applications []*config.Application
	cmds         map[string]*exec.Cmd
	view         ui.View
}

// NewRunner instancites a Runner struct from configuration data
func NewRunner(view ui.View, proxy proxy.Proxy, project *config.Project) *runner {
	return &runner{
		proxy:        proxy,
		projectName:  project.Name,
		applications: project.Applications,
		cmds:         make(map[string]*exec.Cmd, 0),
		view:         view,
	}
}

// RunAll runs all local applications in separated goroutines
func (r *runner) RunAll() {
	for _, application := range r.applications {
		go r.Run(application)

		if application.Hostname != "" {
			proxyForward := proxy.NewProxyForward(application.Name, application.Hostname, "", "", "")
			r.proxy.AddProxyForward(application.Name, proxyForward)
		}
	}
}

// SetupAll runs setup commands for all applications in case their directory does not already exists
func (r *runner) SetupAll() {
	var wg sync.WaitGroup

	for _, application := range r.applications {
		wg.Add(1)
		r.setup(application, &wg)
	}

	wg.Wait()

	if hasSetup {
		r.view.Write("\n✅  Setup complete!\n\n")
	}
}

// Run launches the application
func (r *runner) Run(application *config.Application) {
	if err := helper.CheckPathExists(application.GetPath()); err != nil {
		r.view.Writef("❌  %s\n", err.Error())
		return
	}

	r.run(application)
}

// Run launches the application
func (r *runner) run(application *config.Application) {
	r.view.Writef("🏁  Running local app '%s' (%s)...\n", application.Name, application.Path)

	applicationPath := application.GetPath()

	stdoutStream := log.NewStreamer(log.StdOut, application.Name, r.view)
	stderrStream := log.NewStreamer(log.StdErr, application.Name, r.view)

	cmd := exec.Command(application.Executable, application.Args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Dir = applicationPath
	cmd.Stdout = stdoutStream
	cmd.Stderr = stderrStream
	cmd.Env = os.Environ()

	helper.AddEnvVariables(cmd, application.Env)
	if err := helper.AddEnvVariablesFromFile(cmd, application.GetEnvFile()); err != nil {
		r.view.Writef("❌  %v\n", err)
	}

	r.cmds[application.Name] = cmd

	if err := cmd.Run(); err != nil {
		r.view.Writef("❌  Cannot run the application %s on path %s: %v\n", application.Name, applicationPath, err)
		return
	}
}

// Restart kills the current application launch (if it exists) and launch a new one
func (r *runner) Restart(application *config.Application) {
	r.stopApplication(application)
	go r.Run(application)
}

// Stop stops all the currently active local applications
func (r *runner) Stop() error {
	for _, application := range r.applications {
		r.stopApplication(application)
	}

	return nil
}

func (r *runner) stopApplication(application *config.Application) {
	if cmd, ok := r.cmds[application.Name]; ok {
		pgid, err := syscall.Getpgid(cmd.Process.Pid)
		if err == nil {
			syscall.Kill(-pgid, syscall.SIGKILL)
			cmd.Wait()
		}
	}

	// In case we have stop command, run it
	if application.StopExecutable != "" {
		cmd := exec.Command(application.StopExecutable, application.StopArgs...)
		if err := cmd.Run(); err != nil {
			r.view.Writef("❌  Cannot run stop command for application '%s': %v\n", application.Name, err)
		}

		cmd.Wait()
	}
}

// Setup runs setup commands for a specified application
func (r *runner) setup(application *config.Application, wg *sync.WaitGroup) error {
	defer wg.Done()

	if err := helper.CheckPathExists(application.GetPath()); err == nil {
		return nil
	}

	if len(application.Setup) == 0 {
		return nil
	}

	hasSetup = true

	r.view.Writef("⚙️  Please wait while setup of application '%s'...\n", application.Name)

	stdoutStream := log.NewStreamer(log.StdOut, application.Name, r.view)
	stderrStream := log.NewStreamer(log.StdErr, application.Name, r.view)

	commands := strings.Join(application.Setup, "\n")
	r.view.Writef("👉  Running commands:\n%s\n\n", commands)

	cmd := helper.BuildCmd(application.Setup, "", stdoutStream, stderrStream)
	if err := cmd.Run(); err != nil {
		r.view.Writef("❌  Cannot run build command for application '%s': %v\n", application.Name, err)
	}

	return nil
}
