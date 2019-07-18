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
