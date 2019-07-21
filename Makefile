.PHONY: help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Builds a local version of Monday from sources
	go build -ldflags "-X main.Version=sources-$(shell git rev-parse --short=5 HEAD)" -o monday ./cmd && mv monday /usr/local/bin

generate-mocks: ## Generate mocks for tests
	@echo "> generating mocks..."

	# Monday
	mockery -name=HostfileInterface -dir=pkg/hostfile/ -output internal/tests/mocks/hostfile
	mockery -name=ProxyInterface -dir=pkg/proxy/ -output internal/tests/mocks/proxy
	mockery -name=RunnerInterface -dir=pkg/runner/ -output internal/tests/mocks/runner
	mockery -name=ForwarderInterface -dir=pkg/forwarder/ -output internal/tests/mocks/forwarder
	mockery -name=WatcherInterface -dir=pkg/watcher/ -output internal/tests/mocks/watcher

	# Kubernetes AppsV1
	mockery -name=Interface -dir=vendor/k8s.io/client-go/kubernetes/ -output internal/tests/mocks/kubernetes/client
	mockery -name=AppsV1Interface -dir=vendor/k8s.io/client-go/kubernetes/typed/apps/v1/ -output internal/tests/mocks/kubernetes/client
	mockery -name=DeploymentsGetter -dir=vendor/k8s.io/client-go/kubernetes/typed/apps/v1/ -output internal/tests/mocks/kubernetes/client
	mockery -name=DeploymentInterface -dir=vendor/k8s.io/client-go/kubernetes/typed/apps/v1/ -output internal/tests/mocks/kubernetes/client

	# Kubernetes CoreV1
	mockery -name=CoreV1Interface -dir=vendor/k8s.io/client-go/kubernetes/typed/core/v1/ -output internal/tests/mocks/kubernetes/client
	mockery -name=PodInterface -dir=vendor/k8s.io/client-go/kubernetes/typed/core/v1/ -output internal/tests/mocks/kubernetes/client

	# Kubernetes REST Client
	mockery -name=Interface -dir=vendor/k8s.io/client-go/rest/ -output internal/tests/mocks/kubernetes/rest
