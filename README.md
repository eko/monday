<h1 align="center"><img src="misc/logo.jpg" title="Monday: dev tool for local app and port-forwarding" alt="Monday: dev tool for local app and port-forwarding"></h1>

[![TravisBuildStatus](https://api.travis-ci.org/eko/monday.svg?branch=master)](https://travis-ci.org/eko/monday)
[![GoDoc](https://godoc.org/github.com/eko/monday?status.png)](https://godoc.org/github.com/eko/monday)
[![GoReportCard](https://goreportcard.com/badge/github.com/eko/monday)](https://goreportcard.com/report/github.com/eko/monday)


Your new microservice development environment friend. This CLI tool allows you to define a configuration to work with both local applications (Go, NodeJS, Rust or others) and forward some other applications over Kubernetes in case you don't want to run them locally.

![Schema](https://github.com/eko/monday/blob/master/misc/schema.jpg?raw=true)

## What Monday can do for you?

✅ Define a unified way to setup applications for all your developers

✅ Run your local applications

✅ Hot reload your applications automatically when a change is made locally

✅ Port-forward an application locally using a remote one on Kubernetes (targeting a pod via label) or over SSH

✅ Forward traffic of a remote application over Kubernetes or SSH locally (see types ssh-remote & kubernetes-remote)

✅ Auto reconnect when a port-forward connection is lost

✅ Forward multiple times the same port locally, using an hostname

## Installation

### One-liner

You can download and setup Monday binary by running the following command on your terminal:

```bash
$ curl https://composieux.fr/getmonday.sh | sh
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

Then, build the binary using the available target in Makefile:
```bash
$ make build
```

## Usage

First, you have to initialize monday and edit your configuration file (you have a [configuration example file here](https://raw.githubusercontent.com/eko/monday/master/example.yaml)).
Run the following command and edit the `~/monday.yaml` configuration file just created for you:

⚠️ *Important note*: Because Monday tries to be your best dev tool and manage things for you, you have to give it some chances to help you in editing host file and manipulating network interface for IP/port mapping.

That's why I suggest to run Monday using the following alias:

```bash
alias monday='sudo -E monday'
```

```bash
$ monday init
```

Once your configuration file is ready, you can simply run Monday:

```bash
$ monday
```

Or, you can run a specific project directly by running:

```bash
$ monday run <project name>
```

When you want to edit your configuration again, simply run this command to open it in your favorite editor:

```bash
$ monday edit
```

## Configuration

Configuration of Monday lives in one or multiple YAML files, depending on how you want to organize your files.

By default, `monday init` will initiates a `~/monday.yaml` file.

Please note that you can also split this configuration in multiple files by respecting the following pattern: `~/monday.<something>.yaml`, for instance:
* `~/monday.localapps.yaml`
* `~/monday.forwards.yaml`
* `~/monday.projects.yaml`

This will help you in having smaller and more readable configuration files.

For an overview of what's possible with configuration file, please look at the [configuration example file here](https://raw.githubusercontent.com/eko/monday/master/example.yaml).

## Run tests

Test suite can be run with:

```bash
$ go test -v ./...
```
