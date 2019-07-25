package kubernetes

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/eko/monday/internal/config"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
        _ "k8s.io/client-go/plugin/pkg/client/auth"
)

const (
	// RemoteSSHProxyPort is the SSH proxy port used by the 'ekofr/monday-proxy' docker image
	// to make a remote-forward on the Kubernetes pod to be able to next forward trafic locally
	RemoteSSHProxyPort = 5022

	// ProxyDockerImage is the path to the public Docker image acting as a proxy in the
	// Kubernetes cluster
	ProxyDockerImage = "ekofr/monday-proxy"

	// ProxyPortName is the name given to the SSH port used when deploying the proxy image into the
	// cluster
	ProxyPortName = "ssh-proxy"
)

var (
	defaultKubeConfigPath = fmt.Sprintf("%s/%s", os.Getenv("HOME"), "/.kube/config")
)

type DeploymentBackup struct {
	OldImage   string
	OldPorts   []apiv1.ContainerPort
	Deployment *appsv1.Deployment
}

type Forwarder struct {
	forwardType    string
	name           string
	clientConfig   *restclient.Config
	clientSet      kubernetes.Interface
	restClient     restclient.Interface
	context        string
	namespace      string
	ports          []string
	labels         map[string]string
	portForwarders map[string]*portforward.PortForwarder
	deployments    map[string]*DeploymentBackup
	stopChannel    chan struct{}
	readyChannel   chan struct{}
}

func NewForwarder(forwardType, name, context, namespace string, ports []string, labels map[string]string) (*Forwarder, error) {
	kubeConfigPath := getKubeConfigPath()

	clientConfig, err := initializeClientConfig(context, kubeConfigPath)
	if err != nil {
		return nil, err
	}

	clientSet, err := initializeClientSet(clientConfig)
	if err != nil {
		return nil, err
	}

	return &Forwarder{
		forwardType:    forwardType,
		name:           name,
		context:        context,
		namespace:      namespace,
		labels:         labels,
		ports:          ports,
		clientConfig:   clientConfig,
		clientSet:      clientSet,
		restClient:     clientSet.RESTClient(),
		portForwarders: make(map[string]*portforward.PortForwarder, 0),
		deployments:    make(map[string]*DeploymentBackup, 0),
		stopChannel:    make(chan struct{}, 1),
		readyChannel:   make(chan struct{}),
	}, nil
}

// GetForwardType returns the type of the forward specified in the configuration (ssh, ssh-remote, kubernetes, ...)
func (f *Forwarder) GetForwardType() string {
	return f.forwardType
}

// GetReadyChannel returns the Kubernetes go client channel for ready event
func (f *Forwarder) GetReadyChannel() chan struct{} {
	return f.readyChannel
}

// GetStopChannel returns the Kubernetes go client channel for stop event
func (f *Forwarder) GetStopChannel() chan struct{} {
	return f.readyChannel
}

// Forward method executes the local or remote port-forward depending on the given type
func (f *Forwarder) Forward() error {
	selector := f.getSelector()

	if selector == "" {
		return fmt.Errorf("Please provide a selector of labels in order to use Kubernetes forwarding")
	}

	switch f.forwardType {
	case config.ForwarderKubernetes:
		err := f.forwardLocal(selector)
		if err != nil {
			return err
		}

	case config.ForwarderKubernetesRemote:
		err := f.forwardRemote(selector)
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop stops the current forwarder
func (f *Forwarder) Stop() error {
	// Close port-forwards currently active connections
	for _, portForwarder := range f.portForwarders {
		portForwarder.Close()
	}

	deploymentsClient := f.clientSet.AppsV1().Deployments(f.namespace)

	// Reset currently active remote-forward deployment proxies
	for _, backup := range f.deployments {
		selector := f.getSelector()

		deployments, err := deploymentsClient.List(metav1.ListOptions{LabelSelector: selector})
		if err != nil {
			continue
		}

		if len(deployments.Items) < 1 {
			continue
		}

		// Take first pod matching at the moment, maybe we should take all?
		deployment := deployments.Items[0]

		deployment.Spec.Template.Spec.Containers[0].Image = backup.OldImage
		deployment.Spec.Template.Spec.Containers[0].Ports = backup.OldPorts

		_, err = deploymentsClient.Update(&deployment)
		if err != nil {
			fmt.Printf("❌  An error has occured while stopping/resetting a deployment: %v\n", err)
		}
	}

	return nil
}

func (f *Forwarder) forwardLocal(selector string) error {
	pods, err := f.clientSet.CoreV1().Pods(f.namespace).List(
		metav1.ListOptions{LabelSelector: selector},
	)
	if err != nil {
		return fmt.Errorf("Unable to find pods for selector '%s': %v", selector, err)
	}

	if len(pods.Items) < 1 {
		return fmt.Errorf("No pod available for selector '%s': %v", selector, err)
	}

	pod := pods.Items[0]

	request := f.restClient.Post().Resource("pods").Namespace(f.namespace).Name(pod.Name).SubResource("portforward")

	url := url.URL{
		Scheme:   request.URL().Scheme,
		Host:     request.URL().Host,
		Path:     buildPath(request),
		RawQuery: "timeout=30s",
	}

	transport, upgrader, err := spdy.RoundTripperFor(f.clientConfig)
	if err != nil {
		return err
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", &url)

	l := NewLogstreamer(pod.Name)

	fw, err := portforward.New(dialer, f.ports, f.stopChannel, f.readyChannel, l, l)
	if err != nil {
		return err
	}

	f.portForwarders[f.name] = fw

	err = fw.ForwardPorts()
	if err != nil {
		return err
	}

	return nil
}

func (f *Forwarder) forwardRemote(selector string) error {
	deploymentsClient := f.clientSet.AppsV1().Deployments(f.namespace)

	deployments, err := deploymentsClient.List(
		metav1.ListOptions{LabelSelector: selector},
	)
	if err != nil {
		return err
	}

	if len(deployments.Items) < 1 {
		return fmt.Errorf("No deployment available for selector '%s': %v", selector, err)
	}

	// Take first pod matching at the moment, maybe we should take all?
	deployment := deployments.Items[0]
	container := deployment.Spec.Template.Spec.Containers[0]

	if _, ok := f.deployments[f.name]; !ok {
		fmt.Printf("📡  Setting up proxy on application '%s', please wait some seconds for pod to be ready...\n", deployment.Name)

		f.deployments[f.name] = &DeploymentBackup{
			OldImage:   container.Image,
			OldPorts:   container.Ports,
			Deployment: &deployment,
		}
	}

	container.Image = ProxyDockerImage

	ports := make([]apiv1.ContainerPort, 0)

	for _, port := range container.Ports {
		if port.Name == ProxyPortName {
			continue
		}

		ports = append(ports, port)
	}

	ports = append(ports, apiv1.ContainerPort{
		Name:          ProxyPortName,
		Protocol:      apiv1.ProtocolTCP,
		ContainerPort: RemoteSSHProxyPort,
	})

	container.Ports = ports

	deployment.Spec.Template.Spec.Containers[0] = container
	deployment.Spec.Template.Spec.ReadinessGates = []apiv1.PodReadinessGate{}

	_, err = deploymentsClient.Update(&deployment)
	if err != nil {
		fmt.Println(err)
	}

	time.Sleep(time.Duration(5 * time.Second))

	// Deployment has been updated with proxy, now forward ports locally
	f.forwardLocal(selector)

	return nil
}

func (f *Forwarder) getSelector() string {
	selector := ""

	for label, value := range f.labels {
		if selector != "" {
			selector = selector + ","
		}

		selector = selector + fmt.Sprintf("%s=%s", label, value)
	}

	return selector
}

func initializeClientConfig(context string, kubeConfigPath string) (*restclient.Config, error) {
	overrides := &clientcmd.ConfigOverrides{CurrentContext: context}

	clientConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeConfigPath},
		overrides,
	).ClientConfig()
	if err != nil {
		return nil, err
	}

	return clientConfig, nil
}

func initializeClientSet(clientConfig *restclient.Config) (*kubernetes.Clientset, error) {
	clientSet, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return nil, err
	}

	return clientSet, nil
}

func buildPath(request *restclient.Request) string {
	parts := strings.Split(request.URL().Path, "/namespaces")
	return parts[0] + "/api/v1/namespaces" + parts[1]
}

func getKubeConfigPath() string {
	if value := os.Getenv("MONDAY_KUBE_CONFIG"); value != "" {
		return value
	}

	return defaultKubeConfigPath
}
