.PHONY: help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Builds a local version of Monday from sources
	go build -ldflags "-X main.Version=sources-$(shell git rev-parse --short=5 HEAD)" -o monday ./cmd && mv monday /usr/local/bin

build-binary: ## Builds a single binary of Monday from sources
	go build -ldflags "-X main.Version=sources-$(shell git rev-parse --short=5 HEAD)" -o monday ./cmd

docker-build: ## Builds a docker image of Monday from sources
	docker build -t monday --build-arg Version=$(shell git rev-parse --short=5 HEAD) .

mocks: ## Generate mocks for tests
	@echo "> generating mocks..."

	# Monday
	mockery -name=View -dir=pkg/ui/ -output internal/tests/mocks/ui
	mockery -name=Hostfile -dir=pkg/hostfile/ -output internal/tests/mocks/hostfile
	mockery -name=Proxy -dir=pkg/proxy/ -output internal/tests/mocks/proxy
	mockery -name=Runner -dir=pkg/run/ -output internal/tests/mocks/run
	mockery -name=Forwarder -dir=pkg/forward/ -output internal/tests/mocks/forward
	mockery -name=Watcher -dir=pkg/watch/ -output internal/tests/mocks/watch

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
