package runner

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/eko/monday/internal/config"
)

// Runner is the struct that manage running local applications
type Runner struct {
	projectName  string
	applications []*config.Application
	runs         map[string]*exec.Cmd
}

// NewRunner instancites a Runner struct from configuration data
func NewRunner(project *config.Project) *Runner {
	return &Runner{
		projectName:  project.Name,
		applications: project.Applications,
		runs:         make(map[string]*exec.Cmd, 0),
	}
}

// RunAll runs all local applications in separated goroutines
func (r *Runner) RunAll() {
	for _, application := range r.applications {
		go r.Run(application)
	}
}

// Run launches the application
func (r *Runner) Run(application *config.Application) {
	if err := r.checkApplicationExecutableEnvironment(application); err != nil {
		fmt.Printf("❌  %s\n", err.Error())
		return
	}

	fmt.Printf("⚙️   Running local app '%s' (%s)...\n", application.Name, application.Path)

	applicationPath := application.GetPath()

	stdoutStream := NewLogstreamer(StdOut, application.Name)
	stderrStream := NewLogstreamer(StdErr, application.Name)

	cmd := exec.Command(application.Executable, application.Args...)
	cmd.Dir = applicationPath
	cmd.Stdout = stdoutStream
	cmd.Stderr = stderrStream

	r.runs[application.Name] = cmd

	if err := cmd.Start(); err != nil {
		fmt.Printf("❌  Cannot run the following application: %s: %v\n", applicationPath, err)
		return
	}

	if err := cmd.Wait(); err != nil {
		fmt.Printf("❌  Application '%s' returned an error: %v\n", applicationPath, err)
		return
	}
}

// Restart kills the current application launch (if it exists) and launch a new one
func (r *Runner) Restart(application *config.Application) {
	if cmd, ok := r.runs[application.Name]; ok {
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("❌  Unable to kill application: %v\n", err)
		}
	}

	go r.Run(application)
}

func (r *Runner) checkApplicationExecutableEnvironment(application *config.Application) error {
	applicationPath := application.GetPath()

	// Check application path exists
	if _, err := os.Stat(applicationPath); os.IsNotExist(err) {
		return fmt.Errorf("Unable to find application in your $GOPATH under: %s", applicationPath)
	}

	return nil
}
