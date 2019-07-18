package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestApplicationGetPathWhenAbsoluePath(t *testing.T) {
	// Given
	application := Application{
		Path: "/tmp/this/is/a/test",
	}

	// When
	path := application.GetPath()

	// Path
	assert.Equal(t, "/tmp/this/is/a/test", path)
}

func TestApplicationGetPathWhenGoPath(t *testing.T) {
	// Given
	os.Setenv("GOPATH", "/tmp/gopath")

	application := Application{
		Executable: "go",
		Path:       "fake.github.com/user/repository",
	}

	// When
	path := application.GetPath()

	// Path
	assert.Equal(t, "/tmp/gopath/src/fake.github.com/user/repository", path)
}

func TestForwardIsProxified(t *testing.T) {
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
