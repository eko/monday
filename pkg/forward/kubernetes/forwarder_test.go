package kubernetes

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/ui"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/util/flowcontrol"
)

func TestNewForwarder(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	name := "test-forward"
	context := "context-test"
	namespace := "platform"
	ports := []string{"8080:8080"}
	labels := map[string]string{
		"app": "my-test-app",
	}

	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	view := ui.NewMockView(ctrl)

	// When
	forwarder, err := NewForwarder(view, config.ForwarderKubernetes, name, context, namespace, ports, labels)

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

func TestGetKubeConfigPathWhenDefault(t *testing.T) {
	// When
	configPath := getKubeConfigPath()

	// Then
	assert.Equal(t, configPath, defaultKubeConfigPath)
}

func TestGetKubeConfigPathWhenCustom(t *testing.T) {
	// Given
	os.Setenv("MONDAY_KUBE_CONFIG", "/tmp/custom/.kube/test.config")
	defer os.Setenv("MONDAY_KUBE_CONFIG", "")

	// When
	configPath := getKubeConfigPath()

	// Then
	assert.Equal(t, configPath, "/tmp/custom/.kube/test.config")
}

func TestGetForwardType(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	view := ui.NewMockView(ctrl)

	forwarder, err := NewForwarder(view, config.ForwarderKubernetesRemote, "test-forward", "context-test", "platform", []string{"8080:8080"}, map[string]string{
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	view := ui.NewMockView(ctrl)

	forwarder, err := NewForwarder(view, config.ForwarderKubernetesRemote, "test-forward", "context-test", "platform", []string{"8080:8080"}, map[string]string{
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	view := ui.NewMockView(ctrl)

	forwarder, err := NewForwarder(view, config.ForwarderKubernetesRemote, "test-forward", "context-test", "platform", []string{"8080:8080"}, map[string]string{
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	view := ui.NewMockView(ctrl)

	forwarder, err := NewForwarder(view, config.ForwarderKubernetesRemote, "test-forward", "context-test", "platform", []string{"8080:8080"}, map[string]string{
		"app": "my-test-app",
	})

	// When
	channel := forwarder.GetStopChannel()

	// Then
	assert.IsType(t, make(chan struct{}), channel)
	assert.Nil(t, err)
}

// newRestClientMock returns a REST client pointing to a local test server
func newRestClientMock(t *testing.T) restclient.Interface {
	testServer := httptest.NewServer(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)
		_, _ = res.Write([]byte("ok, port forward is asked"))
	}))
	t.Cleanup(testServer.Close)

	url, _ := url.Parse(testServer.URL)
	rateLimiter := flowcontrol.NewTokenBucketRateLimiter(2.0, 1)
	httpClient := &http.Client{}
	restClientMock, _ := restclient.NewRESTClient(url, "/1.0", restclient.ClientContentConfig{}, rateLimiter, httpClient)

	return restClientMock
}

func TestForwardTypeLocal(t *testing.T) {
	// Given
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	view := ui.NewMockView(ctrl)

	forwarder, err := NewForwarder(view, config.ForwarderKubernetes, "test-forward", "context-test", "backend", []string{"8080:8080"}, map[string]string{
		"app": "my-test-app",
	})
	if err != nil {
		t.Fatal(err)
	}

	// A pod matching the selector exists but is not running
	clientSet := fake.NewClientset(&corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-test-app-bd4sk",
			Namespace: "backend",
			Labels:    map[string]string{"app": "my-test-app"},
		},
	})

	forwarder.clientSet = clientSet
	forwarder.restClient = newRestClientMock(t)

	// When
	err = forwarder.Forward(ctx)

	// Then
	assert.Contains(t, err.Error(), "No runnning pod available for selector 'app=my-test-app'")
}

func TestForwardTypeRemote(t *testing.T) {
	// Given
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	initKubeConfig(t)
	defer os.Remove(defaultKubeConfigPath)

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("📡  Setting up proxy on application '%s', please wait some seconds for pod to be ready...\n", "my-remote-app-deployment")

	forwarder, err := NewForwarder(view, config.ForwarderKubernetesRemote, "test-remote-forward", "context-test", "backend", []string{"8080:8080"}, map[string]string{
		"app": "my-remote-app",
	})
	if err != nil {
		t.Fatal(err)
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-remote-app-deployment",
			Namespace: "backend",
			Labels:    map[string]string{"app": "my-remote-app"},
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Image: "acme.tld/my-remote-app",
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									HostPort:      8080,
									ContainerPort: 8080,
								},
							},
						},
					},
				},
			},
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-remote-app-bd4sk",
			Namespace: "backend",
			Labels:    map[string]string{"app": "my-remote-app"},
		},
	}

	clientSet := fake.NewClientset(deployment, pod)

	forwarder.clientSet = clientSet
	forwarder.restClient = newRestClientMock(t)

	// When
	err = forwarder.Forward(ctx)

	// Then: no running pod is available so the local forward part fails...
	assert.Contains(t, err.Error(), "No runnning pod available for selector 'app=my-remote-app'")

	// ...but the deployment has been backed up and patched with the proxy image
	if backup, ok := forwarder.deployments["test-remote-forward"]; ok {
		assert.Equal(t, "acme.tld/my-remote-app", backup.OldImage)
	} else {
		t.Fatal("Cannot retrieve backuped deployment image when doing remote-forward")
	}

	patched, err := clientSet.AppsV1().Deployments("backend").Get(ctx, "my-remote-app-deployment", metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, ProxyDockerImage, patched.Spec.Template.Spec.Containers[0].Image)
}

func TestSelectPod(t *testing.T) {
	now := metav1.Now()

	readyPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "ready-pod"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}

	runningNotReadyPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "running-not-ready-pod"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionFalse},
			},
		},
	}

	terminatingPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "terminating-pod", DeletionTimestamp: &now},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}

	pendingPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "pending-pod"},
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}

	testCases := []struct {
		name         string
		pods         []corev1.Pod
		expectedPod  string
		expectedBool bool
	}{
		{
			name:         "no pod",
			pods:         []corev1.Pod{},
			expectedBool: false,
		},
		{
			name:         "only pending pod",
			pods:         []corev1.Pod{pendingPod},
			expectedBool: false,
		},
		{
			name:         "prefers ready pod over running one",
			pods:         []corev1.Pod{runningNotReadyPod, readyPod},
			expectedPod:  "ready-pod",
			expectedBool: true,
		},
		{
			name:         "skips terminating pod",
			pods:         []corev1.Pod{terminatingPod, readyPod},
			expectedPod:  "ready-pod",
			expectedBool: true,
		},
		{
			name:         "falls back on running pod when none is ready",
			pods:         []corev1.Pod{pendingPod, runningNotReadyPod},
			expectedPod:  "running-not-ready-pod",
			expectedBool: true,
		},
		{
			name:         "only terminating pod",
			pods:         []corev1.Pod{terminatingPod},
			expectedBool: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			pod, found := selectPod(testCase.pods)

			assert.Equal(t, testCase.expectedBool, found)

			if testCase.expectedBool {
				assert.Equal(t, testCase.expectedPod, pod.Name)
			}
		})
	}
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
	configFile := dir + "/../../../internal/test/forwarder/kubernetes/config"

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
