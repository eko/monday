.PHONY: help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## Builds a local version of Monday from sources
	go build -ldflags "-X main.Version=sources-$(shell git rev-parse --short=5 HEAD)" -o monday ./cmd && mv monday /usr/local/bin

generate-mocks: ## Generate mocks for tests
	@echo "> generating mocks..."
	mockery -name=ProxyInterface -dir=pkg/proxy/ -output internal/tests/mocks
	mockery -name=RunnerInterface -dir=pkg/runner/ -output internal/tests/mocks
	mockery -name=ForwarderInterface -dir=pkg/forwarder/ -output internal/tests/mocks
	mockery -name=WatcherInterface -dir=pkg/watcher/ -output internal/tests/mocks
