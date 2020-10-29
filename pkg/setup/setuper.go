package setup

import (
	"strings"
	"sync"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/helper"
	"github.com/eko/monday/pkg/log"
	"github.com/eko/monday/pkg/ui"
)

type Setuper interface {
	SetupAll()
	Setup(application *config.Application)
}

// setuper is the struct that manage the setuper of local applications
type setuper struct {
	projectName  string
	applications []*config.Application
	view         ui.View
	conf         *config.GlobalSetup
}

// NewSetuper instanciates a setuper struct from configuration data
func NewSetuper(view ui.View, project *config.Project, conf *config.GlobalSetup) *setuper {
	return &setuper{
		projectName:  project.Name,
		applications: project.Applications,
		view:         view,
		conf:         conf,
	}
}

// SetuperAll runs setup commands for all applications in case their directory does not already exists
func (s *setuper) SetupAll() {
	var wg sync.WaitGroup

	for _, application := range s.applications {
		wg.Add(1)

		go func(application *config.Application) {
			defer wg.Done()
			s.Setup(application)
		}(application)
	}

	wg.Wait()
}

// Setuper runs setup commands for a specified application
func (s *setuper) Setup(application *config.Application) {
	if err := helper.CheckPathExists(application.GetPath()); err == nil {
		return
	}

	var setup = application.Setup

	if setup == nil || len(setup.Commands) == 0 {
		return
	}

	s.view.Writef("⚙️  Setuping application '%s'...\n", application.Name)

	stdoutStream := log.NewStreamer(log.StdOut, application.Name, s.view)
	stderrStream := log.NewStreamer(log.StdErr, application.Name, s.view)

	commands := strings.Join(setup.Commands, "\n")
	s.view.Writef("👉  Running commands:\n%s\n\n", commands)

	cmd := helper.BuildCmd(setup.Commands, "", stdoutStream, stderrStream)

	// Merge global environment variables with given ones
	var envs = setup.Env
	if s.conf != nil {
		envs = helper.MergeMapString(setup.Env, s.conf.Env)
	}

	helper.AddEnvVariables(cmd, envs)
	if err := helper.AddEnvVariablesFromFile(cmd, setup.GetEnvFile()); err != nil {
		s.view.Writef("❌  %v\n", err)
		return
	}

	if err := cmd.Run(); err != nil {
		s.view.Writef("❌  Cannot run setup command for application '%s': %v\n", application.Name, err)
		return
	}

	s.view.Write("\n✅  Setup of application complete!\n\n")
}
