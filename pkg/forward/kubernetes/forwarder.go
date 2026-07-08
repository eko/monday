package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/log"
	"github.com/eko/monday/pkg/ui"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
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

	// podMonitorInterval is the interval at which the forwarded pod is checked to still
	// be alive and ready. When it is not anymore (e.g. redeployment), the port-forward
	// is closed so it can reconnect to a fresh pod
	podMonitorInterval = 2 * time.Second
)

var (
	defaultKubeConfigPath = fmt.Sprintf("%s/%s", os.Getenv("HOME"), "/.kube/config")

	// ErrNoSelectorLabel is returned when no selector label is provided in the configuration file.
	ErrNoSelectorLabel = errors.New("please provide a selector of labels in order to use Kubernetes forwarding")
)

type DeploymentBackup struct {
	OldImage   string
	OldPorts   []apiv1.ContainerPort
	Deployment *appsv1.Deployment
}

type Forwarder struct {
	view           ui.View
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

	mux         sync.Mutex
	deployments map[string]*DeploymentBackup
	// stopChannel closes the currently active port-forward attempt, it is renewed on
	// each connection attempt so the forwarder can safely reconnect
	stopChannel chan struct{}
	stopped     bool
	// readyChannel is signaled once, when the first port-forward connection is ready
	readyChannel chan struct{}
	readyOnce    sync.Once

	// onStateChange, when set, receives the forward state transitions
	onStateChange func(state ui.ForwardState, message string)
}

func NewForwarder(view ui.View, forwardType, name, context, namespace string, ports []string, labels map[string]string) (*Forwarder, error) {
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
		view:           view,
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
		readyChannel:   make(chan struct{}),
	}, nil
}

// GetForwardType returns the type of the forward specified in the configuration (ssh, ssh-remote, kubernetes, ...)
func (f *Forwarder) GetForwardType() string {
	return f.forwardType
}

// OnStateChange registers a callback receiving the forward state transitions
func (f *Forwarder) OnStateChange(callback func(state ui.ForwardState, message string)) {
	f.onStateChange = callback
}

func (f *Forwarder) reportState(state ui.ForwardState, message string) {
	if f.onStateChange != nil {
		f.onStateChange(state, message)
	}
}

// GetReadyChannel returns the channel closed once the first port-forward connection is ready
func (f *Forwarder) GetReadyChannel() chan struct{} {
	return f.readyChannel
}

// GetStopChannel returns the channel closing the currently active port-forward connection
func (f *Forwarder) GetStopChannel() chan struct{} {
	f.mux.Lock()
	defer f.mux.Unlock()

	return f.stopChannel
}

// Forward method executes the local or remote port-forward depending on the given type
func (f *Forwarder) Forward(ctx context.Context) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic occured while forwarding %q: %v", f.name, r)
		}
	}()

	selector := f.getSelector()

	if selector == "" {
		return ErrNoSelectorLabel
	}

	switch f.forwardType {
	case config.ForwarderKubernetes:
		err := f.forwardLocal(ctx, selector)
		if err != nil {
			return err
		}

	case config.ForwarderKubernetesRemote:
		err := f.forwardRemote(ctx, selector)
		if err != nil {
			return err
		}
	}

	return nil
}

// Stop stops the current forwarder
func (f *Forwarder) Stop(ctx context.Context) error {
	f.mux.Lock()
	f.stopped = true
	if f.stopChannel != nil {
		close(f.stopChannel)
		f.stopChannel = nil
	}
	f.mux.Unlock()

	// Close port-forwards currently active connections
	for _, portForwarder := range f.portForwarders {
		portForwarder.Close()
	}

	deploymentsClient := f.clientSet.AppsV1().Deployments(f.namespace)

	// Reset currently active remote-forward deployment proxies
	for _, backup := range f.deployments {
		selector := f.getSelector()

		deployments, err := deploymentsClient.List(ctx, metav1.ListOptions{LabelSelector: selector})
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

		_, err = deploymentsClient.Update(ctx, &deployment, metav1.UpdateOptions{})
		if err != nil {
			f.view.Writef("❌  An error has occured while stopping/resetting a deployment: %v\n", err)
		}
	}

	return nil
}

// isPodRunning returns true when a pod is running and is not terminating
func isPodRunning(pod *apiv1.Pod) bool {
	return pod.Status.Phase == apiv1.PodRunning && pod.DeletionTimestamp == nil
}

// isPodReady returns true when a running pod also reports the Ready condition
func isPodReady(pod *apiv1.Pod) bool {
	if !isPodRunning(pod) {
		return false
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == apiv1.PodReady {
			return condition.Status == apiv1.ConditionTrue
		}
	}

	return false
}

// selectPod returns the best pod to forward to: a ready one when available,
// elsewhere a running (non-terminating) one
func selectPod(pods []apiv1.Pod) (*apiv1.Pod, bool) {
	var runningPod *apiv1.Pod

	for i := range pods {
		pod := &pods[i]

		if isPodReady(pod) {
			return pod, true
		}

		if runningPod == nil && isPodRunning(pod) {
			runningPod = pod
		}
	}

	return runningPod, runningPod != nil
}

func (f *Forwarder) forwardLocal(ctx context.Context, selector string) error {
	pods, err := f.clientSet.CoreV1().Pods(f.namespace).List(
		ctx,
		metav1.ListOptions{LabelSelector: selector},
	)
	if err != nil {
		return fmt.Errorf("Unable to find pods for selector '%s': %w", selector, err)
	}

	if len(pods.Items) < 1 {
		return fmt.Errorf("No pod available for selector '%s'", selector)
	}

	runningPod, found := selectPod(pods.Items)
	if !found {
		return fmt.Errorf("No runnning pod available for selector '%s'", selector)
	}

	// Renew the per-attempt channels: the previous ones were closed by the
	// Kubernetes client on the last connection and cannot be reused
	f.mux.Lock()
	if f.stopped {
		f.mux.Unlock()
		return nil
	}
	stopChannel := make(chan struct{})
	readyChannel := make(chan struct{})
	f.stopChannel = stopChannel
	f.mux.Unlock()

	request := f.restClient.Post().Resource("pods").Namespace(f.namespace).Name(runningPod.Name).SubResource("portforward")

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

	stdoutStream := log.NewStreamer(log.StdOut, runningPod.Name, f.view)
	stderrStream := log.NewStreamer(log.StdErr, runningPod.Name, f.view)

	fw, err := portforward.New(dialer, f.ports, stopChannel, readyChannel, stdoutStream, stderrStream)
	if err != nil {
		return err
	}

	f.portForwarders[f.name] = fw

	// Signal the persistent ready channel once the attempt is ready and start
	// monitoring the forwarded pod to reconnect as soon as it goes away
	monitorDone := make(chan struct{})
	defer close(monitorDone)

	go func() {
		select {
		case <-readyChannel:
			f.readyOnce.Do(func() {
				close(f.readyChannel)
			})
			f.reportState(ui.StateReady, fmt.Sprintf("forwarding to pod '%s'", runningPod.Name))
			f.monitorPod(ctx, runningPod.Name, stopChannel, monitorDone)
		case <-monitorDone:
		}
	}()

	if err := fw.ForwardPorts(); err != nil {
		return err
	}

	f.mux.Lock()
	stopped := f.stopped
	f.mux.Unlock()

	if stopped {
		return nil
	}

	return fmt.Errorf("connection to pod '%s' has been closed", runningPod.Name)
}

// monitorPod watches the forwarded pod and closes the current port-forward when the
// pod is deleted or terminating (e.g. on redeployment), triggering a reconnection to
// a fresh pod
func (f *Forwarder) monitorPod(
	ctx context.Context,
	podName string,
	stopChannel chan struct{},
	monitorDone chan struct{},
) {
	ticker := time.NewTicker(podMonitorInterval)
	defer ticker.Stop()

	closeForward := func(reason string) {
		f.reportState(ui.StateReconnecting, fmt.Sprintf("pod '%s' %s", podName, reason))
		f.view.Writef("🔁  Pod '%s' %s: reconnecting port-forward '%s' to a fresh pod...\n", podName, reason, f.name)

		f.mux.Lock()
		defer f.mux.Unlock()

		if f.stopChannel == stopChannel {
			close(f.stopChannel)
			f.stopChannel = nil
		}
	}

	for {
		select {
		case <-monitorDone:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			pod, err := f.clientSet.CoreV1().Pods(f.namespace).Get(ctx, podName, metav1.GetOptions{})
			if apierrors.IsNotFound(err) {
				closeForward("has been deleted")
				return
			}
			if err != nil {
				// Temporary API error: keep the current connection, the port-forward
				// itself will fail if the cluster is really unreachable
				continue
			}

			if pod.DeletionTimestamp != nil || pod.Status.Phase == apiv1.PodSucceeded || pod.Status.Phase == apiv1.PodFailed {
				closeForward("is terminating")
				return
			}
		}
	}
}

func (f *Forwarder) forwardRemote(ctx context.Context, selector string) error {
	deploymentsClient := f.clientSet.AppsV1().Deployments(f.namespace)

	deployments, err := deploymentsClient.List(
		ctx,
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
		f.view.Writef("📡  Setting up proxy on application '%s', please wait some seconds for pod to be ready...\n", deployment.Name)

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

	_, err = deploymentsClient.Update(ctx, &deployment, metav1.UpdateOptions{})
	if err != nil {
		f.view.Write(err.Error())
	}

	time.Sleep(time.Duration(5 * time.Second))

	// Deployment has been updated with proxy, now forward ports locally
	return f.forwardLocal(ctx, selector)
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
