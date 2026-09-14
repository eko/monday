package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testConfigDir(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	require.NoError(t, err)

	return filepath.Clean(dir + "/../../internal/test/config")
}

func TestLoadSingleFile(t *testing.T) {
	// Given
	Filepath = testConfigDir(t) + "/monday.yaml"
	MultipleFilepath = testConfigDir(t) + "/monday.unknown.*.yaml"

	// When
	conf, err := Load()

	// Then
	require.NoError(t, err)
	assert.IsType(t, new(Config), conf)

	assert.Len(t, conf.Projects, 2)
	assert.Equal(t, conf.Watch.Exclude, []string{
		".git",
		"node_modules",
	})
}

func TestLoadMultipleFiles(t *testing.T) {
	// Given
	Filepath = testConfigDir(t) + "/monday.yaml"
	MultipleFilepath = testConfigDir(t) + "/monday.multiple.*.yaml"

	// When
	loaded, err := LoadDetailed()

	// Then
	require.NoError(t, err)
	assert.IsType(t, new(Config), loaded.Config)

	assert.Equal(t, []string{
		testConfigDir(t) + "/monday.multiple.forward.yaml",
		testConfigDir(t) + "/monday.multiple.local.yaml",
		testConfigDir(t) + "/monday.multiple.project.yaml",
	}, loaded.Files)

	assert.Empty(t, loaded.IgnoredKeys)

	assert.Len(t, loaded.Config.Projects, 4)
	assert.Equal(t, loaded.Config.Watch.Exclude, []string{
		".git",
		"node_modules",
		"/event/an/absolute/path/in/multiple/files",
	})

	// No merged file must have been written on disk
	_, err = os.Stat(testConfigDir(t) + "/unknown.yaml")
	assert.True(t, os.IsNotExist(err))
}

func TestLoadWhenCustomDirectory(t *testing.T) {
	// Given
	t.Setenv("MONDAY_CONFIG_PATH", testConfigDir(t))

	setConfigFilePaths() // Normally run during package init()

	// When
	loaded, err := LoadDetailed()

	// Then
	require.NoError(t, err)

	// The single "monday.yaml" file is ignored when multiple files exist
	assert.Len(t, loaded.Files, 3)
	assert.Len(t, loaded.Config.Projects, 4)
}

func TestLoadWhenNoConfigFile(t *testing.T) {
	// Given
	Filepath = testConfigDir(t) + "/does-not-exist.yaml"
	MultipleFilepath = testConfigDir(t) + "/does-not-exist.*.yaml"

	// When
	conf, err := Load()

	// Then
	assert.Nil(t, conf)
	assert.EqualError(t, err, "Configuration file not found. If you run for the first time, please use 'init' command")
}

func TestLoadWhenInvalidConfiguration(t *testing.T) {
	// Given
	dir := t.TempDir()

	err := os.WriteFile(dir+"/monday.yaml", []byte(`
<: &api-local
  name: api
  hostnme: api.svc.local
  run:
    command: go run main.go

projects:
  - name: api
    local:
      - *api-local
    forward:
      - name: db
        type: kube
        values:
          ports:
            - 5432:5432
`), 0o600)
	require.NoError(t, err)

	Filepath = dir + "/monday.yaml"
	MultipleFilepath = dir + "/monday.*.yaml"

	// When
	loaded, err := LoadDetailed()

	// Then
	assert.EqualError(t, err, `monday.yaml:4: unknown field "hostnme" in application (did you mean "hostname"?)`)
	assert.Equal(t, []string{dir + "/monday.yaml"}, loaded.Files)
	assert.Nil(t, loaded.Config)
}

func TestGetProjectNames(t *testing.T) {
	// Given
	Filepath = testConfigDir(t) + "/monday.yaml"
	MultipleFilepath = testConfigDir(t) + "/monday.multiple.*.yaml"

	conf, err := Load()
	require.NoError(t, err)

	// When
	projectNames := conf.GetProjectNames()

	// Then
	assert.Equal(t, []string{
		"full",
		"graphql",
		"forward-only",
		"forward-composieux-website",
	}, projectNames)
}

func TestGetProjectByName(t *testing.T) {
	// Given
	Filepath = testConfigDir(t) + "/monday.yaml"
	MultipleFilepath = testConfigDir(t) + "/monday.multiple.*.yaml"

	conf, err := Load()
	require.NoError(t, err)

	// When
	project, err := conf.GetProjectByName("forward-only")

	// Then
	assert.Nil(t, err)
	assert.Equal(t, &Project{
		Name: "forward-only",
		Forwards: []*Forward{
			{
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
			{
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
	Filepath = testConfigDir(t) + "/monday.yaml"
	MultipleFilepath = testConfigDir(t) + "/monday.unknown.*.yaml"

	conf, err := Load()
	require.NoError(t, err)

	// When
	project, err := conf.GetProjectByName("unknown-project")

	// Then
	assert.Nil(t, project)

	assert.NotNil(t, err)
	assert.Equal(t, "Unable to find project name 'unknown-project' in the configuration", err.Error())
}

func TestHostnames(t *testing.T) {
	// Given
	conf := &Config{
		Applications: []*Application{
			{Name: "grafana", Hostname: "grafana.svc.local"},
		},
		Forwards: []*Forward{
			{Name: "graylog", Type: ForwarderKubernetes},
		},
		Projects: []*Project{
			{
				Name: "api",
				Applications: []*Application{
					{Name: "api", Hostname: "api.svc.local"},
					{Name: "worker"},
				},
				Forwards: []*Forward{
					{Name: "db", Type: ForwarderKubernetes, Values: ForwardValues{Hostname: "db.svc.local"}},
					{Name: "db", Type: ForwarderKubernetes, Values: ForwardValues{Hostname: "db.svc.local"}},
					{Name: "website", Type: ForwarderSSHRemote},
					{Name: "cache", Type: ForwarderKubernetes, Values: ForwardValues{DisableProxy: true}},
				},
			},
		},
	}

	// When
	hostnames := conf.Hostnames()

	// Then
	assert.Equal(t, []string{
		"api.svc.local",
		"db.svc.local",
		"grafana.svc.local",
		"graylog",
	}, hostnames)
}
