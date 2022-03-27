.PHONY: brew-bottle build build-binary docker-build mocks help

# Usage:
# VERSION=2.1.1 make brew-bottle
brew-bottle:
	brew bottle --no-rebuild --json --root-url "https://github.com/eko/monday/releases/download/v${VERSION}/" eko/tap/monday
	@rm monday--${VERSION}.big_sur*
	@echo Next steps:
	@echo "  1: put new value in /usr/local/Homebrew/Library/Taps/eko/homebrew-tap/Formula/monday.rb"
	@echo "  2: $ brew uninstall monday && brew install --build-bottle monday"
	@echo "  3: $ brew bottle --json --root-url "https://github.com/eko/monday/releases/download/v${VERSION}/" eko/tap/monday"

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
	mockgen -source=pkg/build/builder.go -destination=pkg/build/builder_mock.go -package=build
	mockgen -source=pkg/ui/view.go -destination=pkg/ui/view_mock.go -package=ui
	mockgen -source=pkg/hostfile/client.go -destination=pkg/hostfile/client_mock.go -package=hostfile
	mockgen -source=pkg/proxy/proxy.go -destination=pkg/proxy/proxy_mock.go -package=proxy
	mockgen -source=pkg/run/runner.go -destination=pkg/run/runner_mock.go -package=run
	mockgen -source=pkg/setup/setuper.go -destination=pkg/setup/setuper_mock.go -package=setup
	mockgen -source=pkg/forward/forwarder.go -destination=pkg/forward/forwarder_mock.go -package=forward
	mockgen -source=pkg/watch/watcher.go -destination=pkg/watch/watcher_mock.go -package=watch
	mockgen -source=pkg/write/writer.go -destination=pkg/write/writer_mock.go -package=write

	# Kubernetes AppsV1
	mockery --name=Interface --dir=vendor/k8s.io/client-go/kubernetes/ --output=internal/test/mocks/kubernetes/client
	mockery --name=AppsV1Interface --dir=vendor/k8s.io/client-go/kubernetes/typed/apps/v1/ --output=internal/test/mocks/kubernetes/client
	mockery --name=DeploymentsGetter --dir=vendor/k8s.io/client-go/kubernetes/typed/apps/v1/ --output=internal/test/mocks/kubernetes/client
	mockery --name=DeploymentInterface --dir=vendor/k8s.io/client-go/kubernetes/typed/apps/v1/ --output=internal/test/mocks/kubernetes/client

	# Kubernetes CoreV1
	mockery --name=CoreV1Interface --dir=vendor/k8s.io/client-go/kubernetes/typed/core/v1/ --output=internal/test/mocks/kubernetes/client
	mockery --name=PodInterface --dir=vendor/k8s.io/client-go/kubernetes/typed/core/v1/ --output=internal/test/mocks/kubernetes/client
