![txeh - /etc/hosts mangement](logo.png)


# Etc Hosts Management Utility & Go Library

[![GoDoc](https://godoc.org/github.com/txn2/irsync/txeh?status.svg)](https://godoc.org/github.com/txn2/txeh)

### /etc/hosts Management

It is easy to open your [/etc/hosts] file in text editor and add or remove entries. However, if you make heavy use of [/etc/hosts] for software development or DevOps purposes, it can sometimes be difficult to automate and validate large numbers of host entries.

**txeh** was initially built as a golang library to support [kubefwd](https://github.com/txn2/kubefwd), a Kubernetes port-forwarding utility utilizing [/etc/hosts] heavily, to associate custom hostnames with multiple local loopback IP addresses and remove these entries when it terminates.

A computer's [/etc/hosts] file is a powerful utility for developers and system administrators to create localized, custom DNS entries. This small go library and utility were developed to encapsulate the complexity of working with [/etc/hosts] directly by providing a simple interface for adding and removing entries in a [/etc/hosts] file.

## txeh Utility

### Install

MacOS [homebrew](https://brew.sh) users can `brew install txn2/tap/txeh`, otherwise see [releases](https://github.com/txn2/txeh/releases) for packages and binaries for a number of distros and architectures including Windows, Linux and Arm based systems.

#### Install with go install

When installing with Go please use the latest stable Go release. At least go1.16 or greater is required.

To install use: `go install github.com/txn2/txeh/txeh@master`

If you are building from a local source clone, use `go install ./txeh` from the top-level directory of the clone.

go install will typically put the txeh binary inside the bin directory under go env GOPATH, see Go’s [Compile and install packages and dependencies](https://golang.org/cmd/go/#hdr-Compile_and_install_packages_and_dependencies) for more on this. You may need to add that directory to your $PATH if you encounter the error `txeh: command not found` after installation, you can find a guide for adding a directory to your PATH at https://gist.github.com/nex3/c395b2f8fd4b02068be37c961301caa7#file-path-md.

#### Compile and run from source

dependencies are vendored:
```
go run ./txeh/txeh.go
```

### Use

The txeh CLI application allows command line or scripted access to /etc/hosts file modification.

**Example CLI Usage**:
```bash
 _            _
| |___  _____| |__
| __\ \/ / _ \ '_ \
| |_ >  <  __/ | | |
 \__/_/\_\___|_| |_| v1.5.0

Add, remove and re-associate hostname entries in your /etc/hosts file.
Read more including usage as a Go library at https://github.com/txn2/txeh

Usage:
  txeh [flags]
  txeh [command]

Available Commands:
  add         Add hostnames to /etc/hosts
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  list        List hostnames or IP addresses
  remove      Remove a hostname or ip address
  show        Show hostnames in /etc/hosts
  version     Print the version number of txeh


Flags:
  -d, --dryrun         dry run, output to stdout (ignores quiet)
  -h, --help           help for txeh
  -q, --quiet          no output
  -r, --read string    (override) Path to read /etc/hosts file.
  -w, --write string   (override) Path to write /etc/hosts file.
```


```bash
# point the hostnames "test" and "test.two" to the local loopback
sudo txeh add 127.0.0.1 test test.two

# remove the hostname "test"
sudo txeh remove host test

# remove multiple hostnames
sudo txeh remove host test test2 test.two

# remove an IP address and all the hosts that point to it
sudo txeh remove ip 93.184.216.34

# remove multiple IP addresses
sudo txeh remove ip 93.184.216.34 127.1.27.1

# remove CIDR ranges
sudo txeh remove cidr 93.184.216.0/24 127.1.27.0/28

# quiet mode will suppress output
sudo txeh remove ip 93.184.216.34 -q

# dry run will print a rendered /etc/hosts with your changes without
# saving it.
sudo txeh remove ip 93.184.216.34 -d

# use quiet mode and dry-run to direct the rendered /etc/hosts file
# to another file
sudo txeh add 127.1.27.100 dev.example.com -q -d > hosts.test

# specify an alternate /etc/hosts file to read. writing will
# default to the specified read path.
txeh add 127.1.27.100 dev2.example.com -q -r ./hosts.test

# specify a separate read and write oath
txeh add 127.1.27.100 dev3.example.com -r ./hosts.test -w ./hosts.test2

```

## txeh Go Library

**Dependency:**
```bash
go get github.com/txn2/txeh
```

**Example Golang Implementation**:
```go

package main

import (
    "fmt"
    "strings"

    "github.com/txn2/txeh"
)

func main() {
    hosts, err := txeh.NewHostsDefault()
    if err != nil {
        panic(err)
    }

    hosts.AddHost("127.100.100.100", "test")
    hosts.AddHost("127.100.100.101", "logstash")
    hosts.AddHosts("127.100.100.102", []string{"a", "b", "c"})

    hosts.RemoveHosts([]string{"example", "example.machine", "example.machine.example.com"})
    hosts.RemoveHosts(strings.Fields("example2 example.machine2 example.machine.example.com2"))


    hosts.RemoveAddress("127.1.27.1")

    removeList := []string{
        "127.1.27.15",
        "127.1.27.14",
        "127.1.27.13",
    }

    hosts.RemoveAddresses(removeList)

    hfData := hosts.RenderHostsFile()

    // if you like to see what the outcome will
    // look like
    fmt.Println(hfData)

    hosts.Save()
    // or hosts.SaveAs("./test.hosts")
}

```

## Build Release

Build test release:
```bash
goreleaser --skip-publish --clean --skip-validate
```

Build and release:
```bash
GITHUB_TOKEN=$GITHUB_TOKEN goreleaser --rm-dist
```

### License

Apache License 2.0

[/etc/hosts]:https://en.wikipedia.org/wiki/Hosts_(file)
