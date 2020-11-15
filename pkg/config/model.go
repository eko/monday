package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	ForwarderKubernetes       = "kubernetes"
	ForwarderKubernetesRemote = "kubernetes-remote"
	ForwarderProxy            = "proxy"
	ForwarderSSH              = "ssh"
	ForwarderSSHRemote        = "ssh-remote"
)

var (
	// AvailableForwarders lists all ready-to-use forwarders
	AvailableForwarders = map[string]bool{
		ForwarderKubernetes:       true,
		ForwarderKubernetesRemote: true,
		ForwarderProxy:            true,
		ForwarderSSH:              true,
		ForwarderSSHRemote:        true,
	}

	// ProxifiedForwarders lists all forwarders that will use the proxy
	ProxifiedForwarders = map[string]bool{
		ForwarderKubernetes:       true,
		ForwarderKubernetesRemote: true,
		ForwarderProxy:            true,
		ForwarderSSH:              true,
	}
)

// Config represents the root configuration item
type Config struct {
	// Global packages configurations
	Build *GlobalBuild `yaml:"build"`
	Run   *GlobalRun   `yaml:"run"`
	Setup *GlobalSetup `yaml:"setup"`
	Watch *GlobalWatch `yaml:"watch"`

	// Global applications and forward list. If specified, these will always be launched with any project
	Applications []*Application `yaml:"local"`
	Forwards     []*Forward     `yaml:"forward"`

	// Other global configuration values
	GoPath     string `yaml:"gopath"`
	KubeConfig string `yaml:"kubeconfig"`

	// Projects
	Projects []*Project `yaml:"projects"`
}

// GlobalBuild represents the global configuration values for the file builder component
type GlobalBuild struct {
	Env map[string]string `yaml:"env"`
}

// GlobalRun represents the global configuration values for the file runner component
type GlobalRun struct {
	Env map[string]string `yaml:"env"`
}

// GlobalSetup represents the global configuration values for the file setuper component
type GlobalSetup struct {
	Env map[string]string `yaml:"env"`
}

// GlobalWatch represents the global configuration values for the file watcher component
type GlobalWatch struct {
	Exclude []string `yaml:"exclude"`
}

// Project represents a project name, that could be a group of multiple projects
type Project struct {
	Name         string         `yaml:"name"`
	Applications []*Application `yaml:"local"`
	Forwards     []*Forward     `yaml:"forward"`
}

// PrependApplications prepends some global local applications to the current project.
func (p *Project) PrependApplications(applications []*Application) {
	p.Applications = append(applications, p.Applications...)
}

// PrependForwards prepends some global forwards to the current project.
func (p *Project) PrependForwards(forwards []*Forward) {
	p.Forwards = append(forwards, p.Forwards...)
}

// Application represents application information
type Application struct {
	Name       string      `yaml:"name"`
	Path       string      `yaml:"path"`
	Hostname   string      `yaml:"hostname"`
	Watch      bool        `yaml:"watch"`
	Setup      *Setup      `yaml:"setup"`
	Build      *Build      `yaml:"build"`
	Run        *Run        `yaml:"run"`
	Files      []*File     `yaml:"files"`
	Monitoring *Monitoring `yaml:"monitoring"`
}

// Build represents application build information
type Build struct {
	Type     string            `yaml:"type"`
	Path     string            `yaml:"path"`
	Commands []string          `yaml:"commands"`
	Env      map[string]string `yaml:"env"`
	EnvFile  string            `yaml:"env_file"`
}

// GetEnvFile returns the filename guessed with current application environment
func (b *Build) GetEnvFile() string {
	if b.EnvFile == "" {
		return ""
	}

	return getValueByExecutionContext(b.EnvFile)
}

// GetPath returns the path dependending on overrided value or not
func (b *Build) GetPath() string {
	return getValueByExecutionContext(b.Path)
}

// GetPath returns the path dependending on overrided value or not
func (a *Application) GetPath() string {
	return getValueByExecutionContext(a.Path)
}

// File represents a file that have to be written
type File struct {
	Type    string `yaml:"type"`
	From    string `yaml:"from"`
	To      string `yaml:"to"`
	Content string `yaml:"content"`
}

// GetFrom returns the copy from file path dependending on overrided value or not
func (f *File) GetFrom() string {
	return expandValueFromEnvironment(f.From)
}

// GetTo returns the output file path dependending on overrided value or not
func (f *File) GetTo() string {
	return expandValueFromEnvironment(f.To)
}

type Forward struct {
	Name       string        `yaml:"name"`
	Type       string        `yaml:"type"`
	Values     ForwardValues `yaml:"values"`
	Monitoring *Monitoring   `yaml:"monitoring"`
}

// IsProxified indicates if the current forward rule will use the proxy
func (f *Forward) IsProxified() bool {
	if value, ok := ProxifiedForwarders[f.Type]; ok && value {
		return !f.Values.DisableProxy
	}

	return false
}

// ForwardValues represents the available values for each forward type
type ForwardValues struct {
	Context         string            `yaml:"context"`
	Namespace       string            `yaml:"namespace"`
	Labels          map[string]string `yaml:"labels"`
	ForwardHostname string            `yaml:"forward_hostname"`
	Hostname        string            `yaml:"hostname"`
	ProxyHostname   string            `yaml:"proxy_hostname"`
	DisableProxy    bool              `yaml:"disable_proxy"`
	Ports           []string          `yaml:"ports"`
	Remote          string            `yaml:"remote"`
	Args            []string          `yaml:"args"`
}

// Run represents application run information
type Run struct {
	Path         string            `yaml:"path"`
	Command      string            `yaml:"command"`
	Env          map[string]string `yaml:"env"`
	EnvFile      string            `yaml:"env_file"`
	StopCommands []string          `yaml:"stop_commands"`
}

// GetEnvFile returns the filename guessed with current application environment
func (r *Run) GetEnvFile() string {
	if r.EnvFile == "" {
		return ""
	}

	return getValueByExecutionContext(r.EnvFile)
}

// Setup represents application setup information
type Setup struct {
	Commands []string          `yaml:"commands"`
	Env      map[string]string `yaml:"env"`
	EnvFile  string            `yaml:"env_file"`
}

// GetEnvFile returns the filename guessed with current application environment
func (s *Setup) GetEnvFile() string {
	if s.EnvFile == "" {
		return ""
	}

	return getValueByExecutionContext(s.EnvFile)
}

// Monitoring represents application monitoring information
type Monitoring struct {
	Port string `yaml:"port"`
	URL  string `yaml:"url"`
}

func expandValueFromEnvironment(path string) string {
	if strings.Contains(path, "~") {
		path = strings.Replace(path, "~", "$HOME", -1)
	}

	return os.ExpandEnv(path)
}

func getValueByExecutionContext(path string) string {
	path = expandValueFromEnvironment(path)

	// First try to use the given directory, else, try with the Go's $GOPATH
	if _, err := os.Stat(path); os.IsNotExist(err) {
		path = os.ExpandEnv(fmt.Sprintf("$GOPATH/src/%s", path))
	}

	return path
}
