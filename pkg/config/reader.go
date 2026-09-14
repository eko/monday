package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// Filename is the name single YAML configuration file name
	Filename = "monday.yaml"
	// MultipleFilenamePattern is the name pattern for multiple YAML configuration files
	MultipleFilenamePattern = "monday*.yaml"
)

var (
	defaultConfigPath = os.Getenv("HOME")

	// Filepath is the path of the YAML configuration files when you
	// just have a single config file
	Filepath string

	// MultipleFilepath is the path of the YAML configuration files when
	// you define multiple config files
	MultipleFilepath string
)

// Loaded is a configuration loaded from disk, with details about its sources
type Loaded struct {
	Config *Config

	// Files lists the configuration files that were merged, in order
	Files []string

	// IgnoredKeys lists the top-level keys that are not part of the
	// configuration and were ignored: they usually hold the YAML anchors
	// reused in projects
	IgnoredKeys []string
}

func init() {
	setConfigFilePaths()
}

// Load loads and validates the configuration from the YAML configuration files
func Load() (*Config, error) {
	loaded, err := LoadDetailed()
	if err != nil {
		return nil, err
	}

	return loaded.Config, nil
}

// LoadDetailed loads and validates the configuration from the YAML
// configuration files, and reports which files and keys were used. When an
// error is returned, the files that were read are still reported.
func LoadDetailed() (*Loaded, error) {
	files, err := ConfigFiles()
	if err != nil {
		return nil, err
	}

	loaded := &Loaded{Files: files}

	sources := make([]Source, 0, len(files))
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			return loaded, fmt.Errorf("unable to read the configuration file: %w", err)
		}

		sources = append(sources, Source{Name: file, Content: content})
	}

	conf, ignored, err := Parse(sources)
	loaded.IgnoredKeys = ignored
	if err != nil {
		return loaded, err
	}

	if err := conf.Validate(); err != nil {
		return loaded, err
	}

	applyEnvironment(conf)
	loaded.Config = conf

	return loaded, nil
}

// ConfigFiles returns the configuration files to load: the multiple
// "monday*.yaml" files when some exist, elsewhere the single "monday.yaml" file
func ConfigFiles() ([]string, error) {
	if files := FindMultipleConfigFiles(); len(files) > 0 {
		return files, nil
	}

	if err := CheckConfigFileExists(); err != nil {
		return nil, err
	}

	return []string{Filepath}, nil
}

// FindMultipleConfigFiles finds if multiple configuration files has been created
func FindMultipleConfigFiles() []string {
	matches, _ := filepath.Glob(MultipleFilepath)

	files := make([]string, 0, len(matches))
	for _, match := range matches {
		if filepath.Clean(match) == filepath.Clean(Filepath) {
			continue
		}

		files = append(files, match)
	}

	return files
}

// CheckConfigFileExists ensures that config file is present before going further
func CheckConfigFileExists() error {
	if _, err := os.Stat(Filepath); os.IsNotExist(err) {
		return errors.New("Configuration file not found. If you run for the first time, please use 'init' command")
	}

	return nil
}

// applyEnvironment exports the environment variables driven by the configuration
func applyEnvironment(conf *Config) {
	// Override GOPATH environment variable if defined in configuration
	if conf.GoPath != "" {
		os.Setenv("GOPATH", conf.GoPath)
	}

	// Set Kubeconfig filepath if defined in configuration
	if conf.KubeConfig != "" {
		os.Setenv("MONDAY_KUBE_CONFIG", conf.KubeConfig)
	}
}

// GetProjectNames returns the project names as a list
func (c *Config) GetProjectNames() []string {
	list := make([]string, 0)

	for _, project := range c.Projects {
		list = append(list, project.Name)
	}

	return list
}

// GetProjectByName returns a project configuration from its name
func (c *Config) GetProjectByName(name string) (*Project, error) {
	for _, project := range c.Projects {
		if project.Name == name {
			return project, nil
		}
	}

	return nil, fmt.Errorf("Unable to find project name '%s' in the configuration", name)
}

func getConfigPath() string {
	if value := os.Getenv("MONDAY_CONFIG_PATH"); value != "" {
		return value
	}

	return defaultConfigPath
}

func setConfigFilePaths() {
	Filepath = fmt.Sprintf("%s/%s", getConfigPath(), Filename)
	MultipleFilepath = fmt.Sprintf("%s/%s", getConfigPath(), MultipleFilenamePattern)
}
