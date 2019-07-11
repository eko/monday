<h1 align="center"><img src="misc/logo.jpg" title="Monday: dev tool for local app and port-forwarding" alt="Monday: dev tool for local app and port-forwarding"></h1>

[![TravisBuildStatus](https://api.travis-ci.org/eko/monday.svg?branch=master)](https://travis-ci.org/eko/monday)
[![GoDoc](https://godoc.org/github.com/eko/monday?status.png)](https://godoc.org/github.com/eko/monday)
[![GoReportCard](https://goreportcard.com/badge/github.com/eko/monday)](https://goreportcard.com/report/github.com/eko/monday)


Your new microservice local environment friend. This CLI tool allows you to define a configuration to do (or mix) both local applications (Go, NodeJS, Rust or others) and also forward other applications over Kubernetes in case you don't want to run them locally.

![Schema](https://github.com/eko/monday/blob/master/misc/schema.jpg?raw=true)

## What Monday can do for you?

✅ Run your local applications

✅ Hot reload your applications automatically when a change is made locally

✅ Port-forward an application on Kubernetes (targeting a pod via label) or over SSH

✅ Auto reconnect when a port-forward connection is lost

✅ Forward multiple times the same port locally, using an hostname

## Installation

### One-liner

You can download and setup Monday binary by running the following command on your terminal:

```bash
$ curl http://composieux.fr/getmonday.sh | sh
```

### Download binary

You can download the latest version of the binary built for your architecture here:

* Architecture **i386** [
    [Darwin](https://github.com/eko/monday/releases/latest/download/monday-darwin-386) /
    [Linux](https://github.com/eko/monday/releases/latest/download/monday-linux-386)
]
* Architecture **amd64** [
    [Darwin](https://github.com/eko/monday/releases/latest/download/monday-darwin-amd64) /
    [Linux](https://github.com/eko/monday/releases/latest/download/monday-linux-amd64)
]
* Architecture **arm** [
    [Linux](https://github.com/eko/it/releases/latest/download/monday-linux-arm)
]

### From sources

Optionally, you can download and build it from the sources. You have to retrieve the project sources by using one of the following way:
```bash
$ go get -u github.com/eko/monday
# or
$ git clone https://github.com/eko/monday.git
```

Install the needed vendors:

```
$ GO111MODULE=on go mod vendor
```

Then, build the binary (here, an example to run on Raspberry PI ARM architecture):
```bash
$ go build -o monday .
```

## Usage

First, you have to initialize monday and edit your configuration file (you have a [configuration example file here](https://raw.githubusercontent.com/eko/monday/master/example.yaml)).
Run the following command and edit the `~/monday.yaml` configuration file just created for you:

⚠️ *Important note*: Because Monday tries to be your best dev tool and manage things for you, you have to give it some chances to help you in editing host file and manipulating network interface for IP/port mapping.

That's why I suggest to add your current user to the `/etc/hosts` file access list and run Monday using the following alias:

```bash
sudo chmod +a "$USER allow read,write" /etc/hosts
alias monday='sudo -E -u $USER monday'
```

```bash
$ monday init
```

Once your configuration file is ready, you can simply run Monday:

```bash
$ monday
```

When you want to edit your configuration again, simply run this command to open it in your favorite editor:

```bash
$ monday edit
```

## Configuration example

Here is a configuration example on a single file that allows you to see all the things you could do with Monday.

Please note that you can also split this configuration in multiple files by respecting the following pattern: `~/monday.<something>.yaml`, for instance:
* `~/monday.localapps.yaml`
* `~/monday.forwards.yaml`
* `~/monday.projects.yaml`

This will help you in having smaller and more readable configuration files.

```yaml
# Settings

gopath: /Users/vincent/golang # Optional, default to user's $GOPATH env var

# Local applications

<: &graphql-local
  name: graphql
  path: github.com/acme/graphql # Will find in GOPATH (as executable is "go")
  watch: true # Default: false (do not watch directory)
  executable: go
  args:
    - run
    - cmd/main.go

<: &grpc-api-local
  name: grpc-api
  path: github.com/acme/grpc-api # Will find in GOPATH (as executable is "go")
  watch: true # Default: false (do not watch directory)
  executable: go
  args:
    - run
    - main.go

<: &elasticsearch-local
  name: elasticsearch
  path: /Users/vincent/dev/docker
  executable: docker
  args:
    - start
    - -i
    - elastic

# Kubernetes forwards

<: &kubernetes-context preprod

<: &graphql-forward
  name: graphql
  type: kubernetes
  values:
    context: *kubernetes-context
    namespace: backend
    labels:
      app: graphql
    hostname: graphql.svc.local # Optional
    ports:
     - 8080:8000

<: &grpc-api-forward
  name: grpc-api
  type: kubernetes
  values:
    context: *kubernetes-context
    namespace: backend
    labels:
      app: grpc-api
    hostname: grpc-api.svc.local # Optional
    ports:
     - 8080:8080

<: &composieux-fr
  name: composieux-fr
  type: ssh
  values:
    remote: vincent@composieux.fr # SSH <user>@<hostname>
    hostname: composieux.fr.svc.local # Optional
    ports:
     - 8080:80

# Projects

projects:
 - name: full
   local:
    - *graphql-local
    - *grpc-api-local
    - *elasticsearch-local

 - name: graphql
   local:
    - *graphql-local
   forward:
    - *grpc-api-forward

 - name: forward-only
   forward:
    - *graphql-forward
    - *grpc-api-forward

```

## Run tests

Test suite can be run with:

```bash
$ go test -v ./...
```
