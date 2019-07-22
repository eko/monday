package config

import (
	"fmt"
	"os"
	"strings"
)

const (
	ExecutableGo = "go"

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
	GoPath     string     `yaml:"gopath"`
	KubeConfig string     `yaml:"kubeconfig"`
	Projects   []*Project `yaml:"projects"`
	Watcher    *Watcher   `yaml:"watcher"`
}

// Project represents a project name, that could be a group of multiple projects
type Project struct {
	Name         string         `yaml:"name"`
	Applications []*Application `yaml:"local"`
	Forwards     []*Forward     `yaml:"forward"`
}

// Application represents application information
type Application struct {
	Name           string            `yaml:"name"`
	Path           string            `yaml:"path"`
	Executable     string            `yaml:"executable"`
	Args           []string          `yaml:"args"`
	StopExecutable string            `yaml:"stop_executable"`
	StopArgs       []string          `yaml:"stop_args"`
	Hostname       string            `yaml:"hostname"`
	Watch          bool              `yaml:"watch"`
	Env            map[string]string `yaml:"env"`
	Setup          []string          `yaml:"setup"`
}

// GetPath returns the path dependending on overrided value or not
func (a *Application) GetPath() string {
	path := a.Path

	if strings.Contains(a.Path, "~") {
		path = strings.Replace(a.Path, "~", "$HOME", -1)
	}

	switch a.Executable {
	case ExecutableGo:
		// First try to use the given directory, else, add the Go's $GOPATH
		if _, err := os.Stat(path); os.IsNotExist(err) {
			path = fmt.Sprintf("$GOPATH/src/%s", a.Path)
		}
	}

	path = os.ExpandEnv(path)

	return path
}

type Forward struct {
	Name   string        `yaml:"name"`
	Type   string        `yaml:"type"`
	Values ForwardValues `yaml:"values"`
}

// IsProxified indicates if the current forward rule will use the proxy
func (f *Forward) IsProxified() bool {
	if value, ok := ProxifiedForwarders[f.Type]; ok && value {
		return true
	}

	return false
}

// ForwardValues represents the available values for each forward type
type ForwardValues struct {
	Context       string            `yaml:"context"`
	Namespace     string            `yaml:"namespace"`
	Labels        map[string]string `yaml:"labels"`
	Hostname      string            `yaml:"hostname"`
	ProxyHostname string            `yaml:"proxy_hostname"`
	Ports         []string          `yaml:"ports"`
	Remote        string            `yaml:"remote"`
	Args          []string          `yaml:"args"`
}

// Watcher represents the configuration values for the file watcher component
type Watcher struct {
	Exclude []string `yaml:"exclude"`
}
