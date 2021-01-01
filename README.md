<h1 align="center"><img src="misc/logo.jpg" title="Monday: dev tool for local app and port-forwarding" alt="Monday: dev tool for local app and port-forwarding"></h1>

![Test (master)](https://github.com/eko/monday/workflows/Test%20(master)/badge.svg)
[![GoDoc](https://godoc.org/github.com/eko/monday?status.png)](https://godoc.org/github.com/eko/monday)
[![GoReportCard](https://goreportcard.com/badge/github.com/eko/monday)](https://goreportcard.com/report/github.com/eko/monday)

Your new microservice development environment friend. This CLI tool allows you to define a configuration to work with both local applications (Go, NodeJS, Rust or others) and forward some other applications over Kubernetes in case you don't want to run them locally.

The Monday GUI (available for macOS and Linux) application is now also available here: [https://github.com/eko/monday-app](https://github.com/eko/monday-app)

[<img src="https://github.com/eko/monday/blob/master/misc/schema.jpg?raw=true" />](https://youtu.be/6hyCkqHYFQ8)

## What Monday can do for you?

✅  Define a unified way to setup applications for all your developers

✅  Run your local applications

✅  Hot reload your applications automatically when a change is made locally

✅  Port-forward an application locally using a remote one on Kubernetes (targeting a pod via label) or over SSH

✅  Forward traffic of a remote application over Kubernetes, SSH or TCP locally (see example forward types)

✅  Auto reconnect when a port-forward connection is lost

✅  Forward multiple times the same port locally, using an hostname

✅  Monitor your local and/or forwarded applications

## Installation

### Homebrew (macOS)

```bash
$ brew install eko/homebrew-tap/monday
```

This will install the latest available release

### Download binary

You can download the latest version of the binary built for your architecture here:

* Architecture **i386** [
    [Linux](https://github.com/eko/monday/releases/latest/download/monday-linux-386)
]
* Architecture **amd64** [
    [Darwin](https://github.com/eko/monday/releases/latest/download/monday-darwin-amd64) /
    [Linux](https://github.com/eko/monday/releases/latest/download/monday-linux-amd64)
]
* Architecture **arm** [
    [Darwin](https://github.com/eko/monday/releases/latest/download/monday-darwin-arm) /
    [Linux](https://github.com/eko/monday/releases/latest/download/monday-linux-arm)
]

### From sources

Optionally, you can download and build it from the sources. You have to retrieve the project sources by using one of the following way:
```bash
$ go get -u github.com/eko/monday
# or
$ git clone https://github.com/eko/monday.git
```

Then, build the binary using the available target in Makefile:
```bash
$ make build
```

## Configuration: Define your projects

Configuration of Monday lives in one or multiple YAML files, depending on how you want to organize your files.

By default, `monday init` will initiates a `~/monday.yaml` file. You can customize the configuration directory by setting the `MONDAY_CONFIG_PATH` environment variable.

Please note that you can also split this configuration in multiple files by respecting the following pattern: `~/monday.<something>.yaml`, for instance:
* `~/monday.localapps.yaml`
* `~/monday.forwards.yaml`
* `~/monday.projects.yaml`

This will help you navigate more easily in your configuration files.

### Define a local project

Here is an example of a local application:

```yaml
<: &graphql-local
  name: graphql
  path: $GOPATH/src/github.com/eko/graphql
  watch: true
  hostname: graphql.svc.local # Project will be available using this hostname on your machine
  setup: # Setup, installation step in case specified path does not exists
    - go get github.com/eko/graphql
  build: # Optionally, you can define a build section to build your application before running it
    commands:
      - go build -o ./build/graphql-app cmd/ # Here, just build the Go application
    env:
      CGO_ENABLED: on
  run:
    command: ./build/graphql-app # Then, run it using this built binary
    env: # Optional, in case you want to specify some environment variables for this app
      HTTP_PORT: 8005
    env_file: "github.com/eko/graphql/.env" # Or via a .env file also
  files: # Optional, you can also declare some files content with dynamic values coming from your project YAML or simply copy files
    - type: content
      to: $GOPATH/src/github.com/eko/graphql/my_file
      content: |
        This is my file content and here are the current project applications:
          {{- range $app := .Applications }}
          Name: {{ $app.Name }}
          {{- end }}
    - type: copy
      from: $GOPATH/src/github.com/eko/graphql/.env.dist
      to: $GOPATH/src/github.com/eko/graphql/.env
```

Then, imagine this GraphQL instance needs to call a user-api but we want to forward it from a Kubernetes environment, we will define it as follows.

### Define a port-forwarded project

```yaml
<: &user-api-forward
  name: user-api
  type: kubernetes
  values:
    context: staging # This is your kubernetes cluster (kubectl config context name)
    namespace: backend
    labels:
      app: user-api
    hostname: user-api.svc.local # API will be available under this hostname
    ports:
     - 8080:8080
```

Well, you have defined both a local app and an application that needs to be forwarded, now just create the project!

### Define a project with both local app and a port-forwarded one

```yaml
 - name: graphql
   local:
    - *graphql-local
   forward:
    - *user-api-forward
```

Your project configuration is ready, you can now work easily with your microservices.

For an overview of what's possible to do with configuration file, please look at the [configuration example directory here](https://github.com/eko/monday/tree/master/example).

To learn more about the configuration, please take a look at the [Configuration Wiki page](https://github.com/eko/monday/wiki/Configuration).

## Usage: Run your projects!
[![Monday Asciinema](https://asciinema.org/a/aB9ZkCmJS6m1b4uv8Dio1i59U.svg)](https://asciinema.org/a/aB9ZkCmJS6m1b4uv8Dio1i59U)

First, you have to initialize monday and edit your configuration file (you have a [configuration example directory here](https://github.com/eko/monday/tree/master/example)).
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
$ monday [--ui]
```

Note the `--ui` option that will allow you to enable the user interface (you can also define a `MONDAY_ENABLE_UI` environment variable to enable it).

Or, you can run a specific project directly by running:

```bash
$ monday run [--ui] <project name>
```

When you want to edit your configuration again, simply run this command to open it in your favorite editor:

```bash
$ monday edit
```


## Environment variables

The following environment variables can be used to tweak your Monday configuration:


| Environment variable         | Description                                                                               |
|:----------------------------:|-------------------------------------------------------------------------------------------|
| MONDAY_CONFIG_PATH           | Specify the configuration path where your YAML files can be found                         |
| MONDAY_EDITOR                | Specify which editor you want to use in order to edit configuration files                 |
| MONDAY_EDITOR_ARGS           | Specify the editor arguments you want to pass (separated by coma), example: -t,--wite     |
| MONDAY_ENABLE_UI             | Specify that you want to use the terminal UI instead of simply logging to stdout          |
| MONDAY_KUBE_CONFIG           | Specify the location of your Kubernetes config file  (if not in your home directory)      |

## Community

You can [join the community Slack space](https://join.slack.com/t/mondaytool/shared_invite/enQtNzE3NDAxNzIxNTQyLTBmNGU5YzAwNjRjY2IxY2MwZmM5Njg5N2EwY2NjYzEwZWExNWYyYTlmMzg5ZTBjNDRiOTUwYzM3ZDBhZTllOGM) to discuss about your issues, new features or anything else regarding Monday.

## Run tests

Test suite can be run with:

```bash
$ go test -v ./...
```
