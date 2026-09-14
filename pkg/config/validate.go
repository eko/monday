package config

import (
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

var portsMappingRegexp = regexp.MustCompile(`^[0-9]+:[0-9]+$`)

// Validate checks the consistency of the configuration: every project,
// application and forward must hold the values needed to run it. All the
// problems found are returned at once, one per line.
func (c *Config) Validate() error {
	problems := make([]error, 0)

	report := func(format string, args ...any) {
		problems = append(problems, fmt.Errorf(format, args...))
	}

	if len(c.Projects) == 0 {
		report("no project defined, see the configuration examples at https://github.com/eko/monday/tree/master/example")
	}

	validateApplications("global", c.Applications, report)
	validateForwards("global", c.Forwards, report)

	names := make(map[string]bool)

	for i, project := range c.Projects {
		if project == nil || project.Name == "" {
			report("project #%d: missing name", i+1)
			continue
		}

		if names[project.Name] {
			report("project %q: defined more than once", project.Name)
		}
		names[project.Name] = true

		scope := fmt.Sprintf("project %q", project.Name)

		validateApplications(scope, project.Applications, report)
		validateForwards(scope, project.Forwards, report)
	}

	return errors.Join(problems...)
}

func validateApplications(
	scope string,
	applications []*Application,
	report func(format string, args ...any),
) {
	names := make(map[string]bool)

	for i, application := range applications {
		if application == nil || application.Name == "" {
			report("%s: local application #%d: missing name", scope, i+1)
			continue
		}

		if names[application.Name] {
			report("%s: local application %q: defined more than once", scope, application.Name)
		}
		names[application.Name] = true

		prefix := fmt.Sprintf("%s: local application %q", scope, application.Name)

		if application.Run == nil || application.Run.Command == "" {
			report("%s: missing run command", prefix)
		}

		for j, file := range application.Files {
			validateFile(fmt.Sprintf("%s: file #%d", prefix, j+1), file, report)
		}
	}
}

func validateFile(
	prefix string,
	file *File,
	report func(format string, args ...any),
) {
	if file == nil {
		report("%s: empty definition", prefix)
		return
	}

	switch file.Type {
	case FileTypeContent, FileTypeCopy:
		// Valid types

	default:
		report("%s: unknown type %q (available: %s, %s)", prefix, file.Type, FileTypeContent, FileTypeCopy)
	}

	if file.To == "" {
		report("%s: missing destination path 'to'", prefix)
	}

	if file.Type == FileTypeCopy && file.From == "" {
		report("%s: missing source path 'from'", prefix)
	}
}

func validateForwards(
	scope string,
	forwards []*Forward,
	report func(format string, args ...any),
) {
	for i, forward := range forwards {
		if forward == nil || forward.Name == "" {
			report("%s: forward #%d: missing name", scope, i+1)
			continue
		}

		prefix := fmt.Sprintf("%s: forward %q", scope, forward.Name)

		if !AvailableForwarders[forward.Type] {
			report("%s: unknown type %q (available: %s)", prefix, forward.Type, availableForwarderNames())
			continue
		}

		if len(forward.Values.Ports) == 0 {
			report("%s: no port to forward, expected a list of \"<local>:<remote>\" mappings", prefix)
		}

		for _, ports := range forward.Values.Ports {
			if !portsMappingRegexp.MatchString(ports) {
				report("%s: invalid port mapping %q, expected \"<local>:<remote>\"", prefix, ports)
			}
		}

		switch forward.Type {
		case ForwarderKubernetes, ForwarderKubernetesRemote:
			if len(forward.Values.Labels) == 0 {
				report("%s: missing labels selecting the pods to forward", prefix)
			}

		case ForwarderSSH, ForwarderSSHRemote:
			if forward.Values.Remote == "" {
				report("%s: missing remote (<user>@<hostname>)", prefix)
			}

		case ForwarderProxy:
			if forward.Values.ProxyHostname == "" {
				report("%s: missing proxy_hostname", prefix)
			}
		}
	}
}

func availableForwarderNames() string {
	names := make([]string, 0, len(AvailableForwarders))
	for name := range AvailableForwarders {
		names = append(names, name)
	}

	sort.Strings(names)

	return strings.Join(names, ", ")
}
