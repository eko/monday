package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplicationGetPathWhenAbsoluePath(t *testing.T) {
	// Given
	tmpDirectory := os.TempDir()
	defer os.Remove(tmpDirectory)

	application := Application{
		Path: tmpDirectory,
	}

	// When
	path := application.GetPath()

	// Path
	assert.Equal(t, tmpDirectory, path)
}

func TestApplicationGetPathWhenGoPath(t *testing.T) {
	// Given
	os.Setenv("GOPATH", "/tmp/gopath")

	application := Application{
		Path: "fake.github.com/user/repository",
	}

	// When
	path := application.GetPath()

	// Path
	assert.Equal(t, "/tmp/gopath/src/fake.github.com/user/repository", path)
}

func TestForwardTypeIsProxified(t *testing.T) {
	// Given
	testCases := []struct {
		forwardType string
		expected    bool
	}{
		{forwardType: ForwarderKubernetes, expected: true},
		{forwardType: ForwarderKubernetesRemote, expected: true},
		{forwardType: ForwarderSSH, expected: true},
		{forwardType: ForwarderSSHRemote, expected: false},
	}

	// When - Then
	for _, testCase := range testCases {
		forward := Forward{
			Type: testCase.forwardType,
		}

		assert.Equal(t, testCase.expected, forward.IsProxified())
	}
}

func TestForwardConfigIsProxified(t *testing.T) {
	// Given
	testCases := []struct {
		forwardType  string
		disableProxy bool
		expected     bool
	}{
		{forwardType: ForwarderKubernetes, disableProxy: false, expected: true},
		{forwardType: ForwarderKubernetes, disableProxy: true, expected: false},
	}

	// When - Then
	for _, testCase := range testCases {
		values := ForwardValues{
			DisableProxy: testCase.disableProxy,
		}
		forward := Forward{
			Type:   testCase.forwardType,
			Values: values,
		}

		assert.Equal(t, testCase.expected, forward.IsProxified())
	}
}

func TestProjectPrependApplications(t *testing.T) {
	// Given
	project := &Project{
		Name: "My test project",
		Applications: []*Application{
			{Name: "My project app 1"},
			{Name: "My project app 2"},
		},
	}

	conf := &Config{
		Applications: []*Application{
			{Name: "My global app 1"},
			{Name: "My global app 2"},
		},
		Projects: []*Project{project},
	}

	// When
	project.PrependApplications(conf.Applications)

	// Then
	assert.Equal(t, []*Application{
		{Name: "My global app 1"},
		{Name: "My global app 2"},
		{Name: "My project app 1"},
		{Name: "My project app 2"},
	}, project.Applications)
}

func TestProjectPrependForwards(t *testing.T) {
	// Given
	project := &Project{
		Name: "My test project",
		Forwards: []*Forward{
			{Name: "My project forward 1"},
			{Name: "My project forward 2"},
		},
	}

	conf := &Config{
		Forwards: []*Forward{
			{Name: "My global forward 1"},
			{Name: "My global forward 2"},
		},
		Projects: []*Project{project},
	}

	// When
	project.PrependForwards(conf.Forwards)

	// Then
	assert.Equal(t, []*Forward{
		{Name: "My global forward 1"},
		{Name: "My global forward 2"},
		{Name: "My project forward 1"},
		{Name: "My project forward 2"},
	}, project.Forwards)
}
