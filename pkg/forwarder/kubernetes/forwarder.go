package kubernetes

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type Forwarder struct {
	clientConfig *restclient.Config
	clientSet    *kubernetes.Clientset
	context      string
	namespace    string
	ports        []string
	labels       map[string]string
}

func NewForwarder(context, namespace string, ports []string, labels map[string]string) (*Forwarder, error) {
	clientConfig, err := initializeClientConfig(context)
	if err != nil {
		return nil, err
	}

	clientSet, err := initializeClientSet(clientConfig)
	if err != nil {
		return nil, err
	}

	return &Forwarder{
		clientConfig: clientConfig,
		clientSet:    clientSet,
		context:      context,
		namespace:    namespace,
		labels:       labels,
		ports:        ports,
	}, nil
}

func (f *Forwarder) Forward() error {
	selector := f.getSelector()

	if selector == "" {
		return fmt.Errorf("Please provide a selector of labels in order to use Kubernetes forwarding")
	}

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

	client := f.clientSet.RESTClient()
	request := client.Post().Resource("pods").Namespace(f.namespace).Name(pod.Name).SubResource("portforward")

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

	stopChannel := make(chan struct{}, 1)
	readyChannel := make(chan struct{})

	l := NewLogstreamer(pod.Name)

	fw, err := portforward.New(dialer, f.ports, stopChannel, readyChannel, l, l)
	if err != nil {
		return err
	}

	err = fw.ForwardPorts()
	if err != nil {
		return err
	}

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

func initializeClientConfig(context string) (*restclient.Config, error) {
	overrides := &clientcmd.ConfigOverrides{CurrentContext: context}

	defaultConfigPath := fmt.Sprintf("%s/%s", os.Getenv("HOME"), "/.kube/config")

	clientConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: defaultConfigPath},
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
