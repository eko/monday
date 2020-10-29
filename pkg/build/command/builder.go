package command

import (
	"fmt"
	"strings"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/helper"
	"github.com/eko/monday/pkg/log"
	"github.com/eko/monday/pkg/ui"
)

const (
	BuilderType = "command"
)

func Build(application *config.Application, view ui.View) error {
	var build = application.Build

	var buildPath = build.GetPath()

	// Fallback on application path if no build path filled
	if build.Path == "" {
		buildPath = application.GetPath()
	}

	commandList := strings.Join(build.Commands, "\n")
	view.Writef("👉  Running commands:\n%s\n", commandList)

	stdoutStream := log.NewStreamer(log.StdOut, application.Name, view)
	stderrStream := log.NewStreamer(log.StdErr, application.Name, view)

	cmd := helper.BuildCmd(build.Commands, buildPath, stdoutStream, stderrStream)

	helper.AddEnvVariables(cmd, build.Env)
	if err := helper.AddEnvVariablesFromFile(cmd, application.GetEnvFile()); err != nil {
		return err
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cannot run the build command for application %s on path %s: %v", application.Name, buildPath, err)
	}

	return nil
}
