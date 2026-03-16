![txeh - /etc/hosts mangement](logo.png)


# Etc Hosts Management Utility & Go Library

[![CI](https://github.com/txn2/txeh/actions/workflows/ci.yml/badge.svg)](https://github.com/txn2/txeh/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/txn2/txeh/branch/master/graph/badge.svg)](https://codecov.io/gh/txn2/txeh)
[![Go Report Card](https://goreportcard.com/badge/github.com/txn2/txeh)](https://goreportcard.com/report/github.com/txn2/txeh)
[![Go Reference](https://pkg.go.dev/badge/github.com/txn2/txeh.svg)](https://pkg.go.dev/github.com/txn2/txeh)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/txn2/txeh/badge)](https://securityscorecards.dev/viewer/?uri=github.com/txn2/txeh)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

**[Documentation](https://txeh.txn2.com)** | **[Go Reference](https://pkg.go.dev/github.com/txn2/txeh)** | **[Releases](https://github.com/txn2/txeh/releases)**

### /etc/hosts Management

It is easy to open your [/etc/hosts] file in text editor and add or remove entries. However, if you make heavy use of [/etc/hosts] for software development or DevOps purposes, it can sometimes be difficult to automate and validate large numbers of host entries.

**txeh** was initially built as a golang library to support [kubefwd](https://github.com/txn2/kubefwd), a Kubernetes port-forwarding utility utilizing [/etc/hosts] heavily, to associate custom hostnames with multiple local loopback IP addresses and remove these entries when it terminates.

A computer's [/etc/hosts] file is a powerful utility for developers and system administrators to create localized, custom DNS entries. This small go library and utility were developed to encapsulate the complexity of working with [/etc/hosts] directly by providing a simple interface for adding and removing entries in a [/etc/hosts] file.

**Features:**
- Thread-safe operations (mutex-protected for concurrent use)
- IPv4 and IPv6 support
- CIDR range operations for bulk add/remove
- Preserves comments and file formatting
- Cross-platform (Linux, macOS, Windows)

## txeh Utility

### Install

MacOS [homebrew](https://brew.sh) users can `brew install txn2/tap/txeh`, otherwise see [releases](https://github.com/txn2/txeh/releases) for packages and binaries for a number of distros and architectures including Windows, Linux and Arm based systems.

#### Install with go install

When installing with Go please use the latest stable Go release. At least go1.24 or greater is required.

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
 \__/_/\_\___|_| |_|

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
  -d, --dryrun                   dry run, output to stdout (ignores quiet)
  -h, --help                     help for txeh
  -m, --max-hosts-per-line int   Max hostnames per line (0=auto, -1=unlimited, >0=explicit)
  -q, --quiet                    no output
  -r, --read string              (override) Path to read /etc/hosts file.
  -w, --write string             (override) Path to write /etc/hosts file.
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

# specify a separate read and write path
txeh add 127.1.27.100 dev3.example.com -r ./hosts.test -w ./hosts.test2

```

## Comments

txeh supports inline comments on host entries, useful for tracking which tool or purpose added specific entries (e.g., docker, development environments).

### How Comments Work

Comments use the standard hosts file format: `IP HOSTNAME [HOSTNAME...] # comment`

**Key behavior:**
- Comments are used for **grouping**: hosts with the same IP AND same comment are placed on the same line
- Hosts added **without** a comment only match lines **without** a comment
- Hosts added **with** a comment only match lines **with the same comment**
- Comments are **never modified** on existing lines
- If no matching line exists, a new line is created

### CLI Usage

```bash
# Add host with a comment
sudo txeh add 127.0.0.1 myapp --comment "local development"
# Result: 127.0.0.1        myapp # local development

# Add another host with the same comment (groups together)
sudo txeh add 127.0.0.1 myapp2 --comment "local development"
# Result: 127.0.0.1        myapp myapp2 # local development

# Add host with a different comment (new line)
sudo txeh add 127.0.0.1 api --comment "staging environment"
# Result: 127.0.0.1        api # staging environment

# Add host without a comment (only matches lines without comments)
sudo txeh add 127.0.0.1 test
# Result: goes to first 127.0.0.1 line that has NO comment, or creates new line
```

### Library Usage

```go
// Add hosts with a comment
hosts.AddHostWithComment("127.0.0.1", "myapp", "local development")
hosts.AddHostsWithComment("127.0.0.1", []string{"svc1", "svc2"}, "my project services")

// Add hosts without a comment (original behavior)
hosts.AddHost("127.0.0.1", "myhost")
hosts.AddHosts("127.0.0.1", []string{"a", "b", "c"})
```

### Example Scenario

Starting hosts file:
```
127.0.0.1        localhost
127.0.0.1        app1 app2 # dev services
```

| Command | Result |
|---------|--------|
| `txeh add 127.0.0.1 app3 -c "dev services"` | app3 added to "dev services" line |
| `txeh add 127.0.0.1 api -c "staging env"` | New line: `127.0.0.1 api # staging env` |
| `txeh add 127.0.0.1 test` | test added to localhost line (no comment) |

### Listing and Removing by Comment

List all hosts with a specific comment:
```bash
sudo txeh list bycomment "dev services"
```

Remove all entries with a specific comment (removes entire lines):
```bash
sudo txeh remove bycomment "dev services"
```

### Modifying Comments

Comments are never modified on existing lines. To change a comment, remove the hosts first and re-add them with the new comment:

```bash
# Remove all entries with the old comment
sudo txeh remove bycomment "old comment"

# Re-add with the new comment
sudo txeh add 127.0.0.1 app1 app2 --comment "new comment"
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

    "github.com/txn2/txeh"
)

func main() {
    // Load the system hosts file
    hosts, err := txeh.NewHostsDefault()
    if err != nil {
        panic(err)
    }

    // Add hosts (without comments)
    hosts.AddHost("127.100.100.100", "myapp")
    hosts.AddHost("127.100.100.101", "database")
    hosts.AddHosts("127.100.100.102", []string{"cache", "queue", "search"})

    // Add hosts with comments (for tracking/organization)
    hosts.AddHostWithComment("127.100.100.200", "api.local", "development services")
    hosts.AddHostsWithComment("127.100.100.201", []string{"web.local", "admin.local"}, "frontend apps")

    // Query existing entries
    addresses := hosts.ListAddressesByHost("myapp", true)
    fmt.Printf("myapp resolves to: %v\n", addresses)

    hostnames := hosts.ListHostsByIP("127.100.100.102")
    fmt.Printf("Hosts at 127.100.100.102: %v\n", hostnames)

    // List all hosts with a specific comment
    devHosts := hosts.ListHostsByComment("development services")
    fmt.Printf("Dev service hosts: %v\n", devHosts)

    // Remove entries
    hosts.RemoveHost("database")
    hosts.RemoveHosts([]string{"cache", "queue"})
    hosts.RemoveAddress("127.1.27.1")
    hosts.RemoveAddresses([]string{"127.1.27.15", "127.1.27.14"})

    // Remove all entries with specific comments
    hosts.RemoveByComment("frontend apps")
    hosts.RemoveByComments([]string{"old environment", "deprecated"})

    // RemoveCIDRs returns an error (CIDR parsing can fail)
    if err := hosts.RemoveCIDRs([]string{"10.0.0.0/8"}); err != nil {
        panic(err)
    }

    // Preview changes
    fmt.Println(hosts.RenderHostsFile())

    // Save changes
    err = hosts.Save()
    if err != nil {
        panic(err)
    }
    // Or save to a specific file: hosts.SaveAs("./custom.hosts")
}
```

## Build Release

Build test release:
```bash
goreleaser --skip-publish --clean --skip-validate
```

Build and release:
```bash
GITHUB_TOKEN=$GITHUB_TOKEN goreleaser --clean
```

### License

Apache License 2.0

---

Open source by [Craig Johnston](https://twitter.com/cjimti), sponsored by [Deasil Works, Inc.](https://deasil.works/)

[/etc/hosts]:https://en.wikipedia.org/wiki/Hosts_(file)
