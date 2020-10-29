package run

import (
	"os/exec"
	"syscall"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/helper"
	"github.com/eko/monday/pkg/log"
	"github.com/eko/monday/pkg/proxy"
	"github.com/eko/monday/pkg/ui"
)

type Runner interface {
	RunAll()
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
	conf         *config.GlobalRun
}

// NewRunner instanciates a Runner struct from configuration data
func NewRunner(view ui.View, proxy proxy.Proxy, project *config.Project, conf *config.GlobalRun) *runner {
	return &runner{
		proxy:        proxy,
		projectName:  project.Name,
		applications: project.Applications,
		cmds:         make(map[string]*exec.Cmd, 0),
		view:         view,
		conf:         conf,
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
	var run = application.Run

	if run == nil {
		r.view.Writef("❌  Please declare a 'run' section for application %s\n", application.Name)
		return
	}

	r.view.Writef("🏁  Running local app '%s' (%s)...\n", application.Name, application.Path)

	applicationPath := application.GetPath()

	stdoutStream := log.NewStreamer(log.StdOut, application.Name, r.view)
	stderrStream := log.NewStreamer(log.StdErr, application.Name, r.view)

	cmd := helper.BuildCmd([]string{run.Command}, applicationPath, stdoutStream, stderrStream)

	// Merge global environment variables with given ones
	var envs = run.Env
	if r.conf != nil {
		envs = helper.MergeMapString(run.Env, r.conf.Env)
	}

	helper.AddEnvVariables(cmd, envs)
	if err := helper.AddEnvVariablesFromFile(cmd, run.GetEnvFile()); err != nil {
		r.view.Writef("❌  %v\n", err)
		return
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
	if len(application.Run.StopCommands) > 0 {
		cmd := helper.BuildCmd(application.Run.StopCommands, application.GetPath(), nil, nil)
		if err := cmd.Run(); err != nil {
			r.view.Writef("❌  Cannot run stop command for application '%s': %v\n", application.Name, err)
		}

		cmd.Wait()
	}
}
