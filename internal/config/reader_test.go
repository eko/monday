package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadSingleFile(t *testing.T) {
	// Given
	dir, _ := os.Getwd()
	Filepath = dir + "/../tests/config/config.yaml"
	MultipleFilepath = dir + "/../tests/config/config.unknown.*.yaml"

	// When
	conf, err := Load()

	// Then
	assert.IsType(t, new(Config), conf)
	assert.Nil(t, err)

	assert.Len(t, conf.Projects, 4)
	assert.Equal(t, conf.Watcher.Exclude, []string{
		".git",
		"node_modules",
	})
}

func TestLoadMultipleFiles(t *testing.T) {
	// Given
	dir, _ := os.Getwd()
	Filepath = dir + "/../tests/config/unknown.yaml"
	MultipleFilepath = dir + "/../tests/config/config.multiple.*.yaml"

	// Remove single config created file after test
	defer os.Remove(Filepath)

	// When
	conf, err := Load()

	// Then
	assert.IsType(t, new(Config), conf)
	assert.Nil(t, err)

	assert.Len(t, conf.Projects, 4)
	assert.Equal(t, conf.Watcher.Exclude, []string{
		".git",
		"node_modules",
		"/event/an/absolute/path/in/multiple/files",
	})
}

func TestGetProjectNames(t *testing.T) {
	// Given
	dir, _ := os.Getwd()
	Filepath = dir + "/../tests/config/config.yaml"
	MultipleFilepath = dir + "/../tests/config/config.unknown.*.yaml"

	conf, err := Load()

	// When
	projectNames := conf.GetProjectNames()

	// Then
	assert.Nil(t, err)
	assert.Equal(t, []string{
		"full",
		"graphql",
		"forward-only",
		"forward-composieux-website",
	}, projectNames)
}

func TestGetProjectByName(t *testing.T) {
	// Given
	dir, _ := os.Getwd()
	Filepath = dir + "/../tests/config/config.yaml"
	MultipleFilepath = dir + "/../tests/config/config.unknown.*.yaml"

	conf, err := Load()

	// When
	project, err := conf.GetProjectByName("forward-only")

	// Then
	assert.Nil(t, err)
	assert.Equal(t, &Project{
		Name: "forward-only",
		Forwards: []*Forward{
			&Forward{
				Name: "graphql",
				Type: "kubernetes",
				Values: ForwardValues{
					Context:   "context-test",
					Namespace: "backend",
					Labels: map[string]string{
						"app": "graphql",
					},
					Hostname: "graphql.svc.local",
					Ports: []string{
						"8080:8000",
					},
				},
			},
			&Forward{
				Name: "grpc-api",
				Type: "kubernetes",
				Values: ForwardValues{
					Context:   "context-test",
					Namespace: "backend",
					Labels: map[string]string{
						"app": "grpc-api",
					},
					Hostname: "grpc-api.svc.local",
					Ports: []string{
						"8080:8080",
					},
				},
			},
		},
	}, project)
}

func TestGetProjectByNameWhenProjectNotFound(t *testing.T) {
	// Given
	dir, _ := os.Getwd()
	Filepath = dir + "/../tests/config/config.yaml"
	MultipleFilepath = dir + "/../tests/config/config.unknown.*.yaml"

	conf, err := Load()

	// When
	project, err := conf.GetProjectByName("unknown-project")

	// Then
	assert.Nil(t, project)

	assert.NotNil(t, err)
	assert.Equal(t, "Unable to find project name 'unknown-project' in the configuration", err.Error())
}
