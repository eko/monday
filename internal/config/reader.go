package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v2"
)

const (
	// Filename is the name of the YAML configuration file
	Filename = "launcher.yaml"
)

var (
	Filepath = fmt.Sprintf("%s/%s", os.Getenv("HOME"), Filename)
)

// Load method loads the configuration from the YAML configuration file
func Load() (*Config, error) {
	if err := CheckConfigFileExists(); err != nil {
		return nil, err
	}

	file, err := ioutil.ReadFile(Filepath)
	if err != nil {
		log.Printf("Error while reading config file: #%v", err)
	}

	var conf Config
	err = yaml.Unmarshal(file, &conf)
	if err != nil {
		panic(fmt.Sprintf("An error has occured while reading configuration file:\n%v", err))
	}

	// Override GOPATH environment variable if defined in configuration
	if conf.GoPath != "" {
		os.Setenv("GOPATH", conf.GoPath)
	}

	return &conf, nil
}

// CheckConfigFileExists ensures that config file is present before going further
func CheckConfigFileExists() error {
	if _, err := os.Stat(Filepath); os.IsNotExist(err) {
		return errors.New("Configuration file not found in your home directory. If you run for the first time, please use 'init' command")
	}

	return nil
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
