package runner

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/eko/monday/internal/config"
	"github.com/eko/monday/pkg/proxy"
)

// Runner is the struct that manage running local applications
type Runner struct {
	proxy        *proxy.Proxy
	projectName  string
	applications []*config.Application
	cmds         map[string]*exec.Cmd
}

// NewRunner instancites a Runner struct from configuration data
func NewRunner(proxy *proxy.Proxy, project *config.Project) *Runner {
	return &Runner{
		proxy:        proxy,
		projectName:  project.Name,
		applications: project.Applications,
		cmds:         make(map[string]*exec.Cmd, 0),
	}
}

// RunAll runs all local applications in separated goroutines
func (r *Runner) RunAll() {
	for _, application := range r.applications {
		go r.Run(application)

		if application.Hostname != "" {
			proxyForward := proxy.NewProxyForward(application.Name, application.Hostname, "", "")
			r.proxy.AddProxyForward(application.Name, proxyForward)
		}
	}
}

// SetupAll runs setup commands for all applications in case their directory does not already exists
func (r *Runner) SetupAll() {
	var wg sync.WaitGroup

	for _, application := range r.applications {
		wg.Add(1)
		r.setup(application, &wg)
	}

	wg.Wait()

	fmt.Print("✅  Setup complete!\n\n")
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
	cmd.Env = os.Environ()

	// Add environment variables
	for key, value := range application.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	r.cmds[application.Name] = cmd

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
	if cmd, ok := r.cmds[application.Name]; ok {
		if err := cmd.Process.Kill(); err != nil {
			fmt.Printf("❌  Unable to kill application: %v\n", err)
		}
	}

	go r.Run(application)
}

// Stop stops all the currently active local applications
func (r *Runner) Stop() error {
	for _, cmd := range r.cmds {
		cmd.Process.Kill()
	}

	return nil
}

func (r *Runner) checkApplicationExecutableEnvironment(application *config.Application) error {
	applicationPath := application.GetPath()

	// Check application path exists
	if _, err := os.Stat(applicationPath); os.IsNotExist(err) {
		return fmt.Errorf("Unable to find application path: %s", applicationPath)
	}

	return nil
}

// Setup runs setup commands for a specified application
func (r *Runner) setup(application *config.Application, wg *sync.WaitGroup) error {
	defer wg.Done()

	if err := r.checkApplicationExecutableEnvironment(application); err == nil {
		return nil
	}

	fmt.Printf("⚙️  Please wait while setup of application '%s'...\n", application.Name)

	stdoutStream := NewLogstreamer(StdOut, application.Name)
	stderrStream := NewLogstreamer(StdErr, application.Name)

	var setup = strings.Join(application.Setup, "; ")

	setup = strings.Replace(setup, "~", "$HOME", -1)
	setup = os.ExpandEnv(setup)

	commands := strings.Join(application.Setup, "\n")
	fmt.Printf("👉  Running commands:\n%s\n\n", commands)

	cmd := exec.Command("/bin/sh", "-c", setup)
	cmd.Stdout = stdoutStream
	cmd.Stderr = stderrStream
	cmd.Env = os.Environ()

	setup = os.ExpandEnv(setup)

	cmd.Run()

	return nil
}
