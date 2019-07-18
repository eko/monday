package kubernetes

import (
	"io"
	"os"
	"testing"

	"github.com/eko/monday/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewForwarder(t *testing.T) {
	// Given
	name := "test-forward"
	context := "context-test"
	namespace := "platform"
	ports := []string{"8080:8080"}
	labels := map[string]string{
		"app": "my-test-app",
	}

	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	// When
	forwarder, err := NewForwarder(config.ForwarderKubernetes, name, context, namespace, ports, labels)

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.Equal(t, config.ForwarderKubernetes, forwarder.forwardType)
	assert.Equal(t, name, forwarder.name)
	assert.Equal(t, context, forwarder.context)
	assert.Equal(t, namespace, forwarder.namespace)
	assert.Equal(t, ports, forwarder.ports)

	assert.Len(t, forwarder.portForwarders, 0)
	assert.Len(t, forwarder.deployments, 0)
}

func TestGetForwardType(t *testing.T) {
	// Given
	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	forwarder, err := NewForwarder(config.ForwarderKubernetesRemote, "test-forward", "context-test", "platform", []string{"8080:8080"}, map[string]string{
		"app": "my-test-app",
	})

	// When
	forwardType := forwarder.GetForwardType()

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.Equal(t, config.ForwarderKubernetesRemote, forwardType)
}

func TestGetSelector(t *testing.T) {
	// Given
	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	forwarder, err := NewForwarder(config.ForwarderKubernetesRemote, "test-forward", "context-test", "platform", []string{"8080:8080"}, map[string]string{
		"app": "my-test-app",
	})

	// When
	selector := forwarder.getSelector()

	// Then
	assert.IsType(t, new(Forwarder), forwarder)
	assert.Nil(t, err)

	assert.Equal(t, "app=my-test-app", selector)
}

func TestGetReadyChannel(t *testing.T) {
	// Given
	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	forwarder, err := NewForwarder(config.ForwarderKubernetesRemote, "test-forward", "context-test", "platform", []string{"8080:8080"}, map[string]string{
		"app": "my-test-app",
	})

	// When
	channel := forwarder.GetReadyChannel()

	// Then
	assert.IsType(t, make(chan struct{}), channel)
	assert.Nil(t, err)
}

func TestGetStopChannel(t *testing.T) {
	// Given
	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	forwarder, err := NewForwarder(config.ForwarderKubernetesRemote, "test-forward", "context-test", "platform", []string{"8080:8080"}, map[string]string{
		"app": "my-test-app",
	})

	// When
	channel := forwarder.GetStopChannel()

	// Then
	assert.IsType(t, make(chan struct{}), channel)
	assert.Nil(t, err)
}

// Initializes a Kubernetes configuration for test environment
func initKubeConfig(t *testing.T) {
	directoryKubeConfig := "/tmp/.kube"
	defaultKubeConfigPath = directoryKubeConfig + "/config"

	err := os.MkdirAll(directoryKubeConfig, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	file, err := os.Create(defaultKubeConfigPath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	dir, _ := os.Getwd()
	configFile := dir + "/../../../internal/tests/forwarder/kubernetes/config"

	from, err := os.OpenFile(configFile, os.O_RDONLY, 0666)
	if err != nil {
		t.Fatal(err)
	}
	defer from.Close()

	_, err = io.Copy(file, from)
	if err != nil {
		t.Fatal(err)
	}
}
