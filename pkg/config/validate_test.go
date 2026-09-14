package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func validProject() *Project {
	return &Project{
		Name: "api",
		Applications: []*Application{
			{
				Name:     "api",
				Hostname: "api.svc.local",
				Run:      &Run{Command: "go run main.go"},
				Files: []*File{
					{Type: FileTypeCopy, From: ".env.dist", To: ".env"},
					{Type: FileTypeContent, To: "config.yml", Content: "foo: bar"},
				},
			},
		},
		Forwards: []*Forward{
			{
				Name: "db",
				Type: ForwarderKubernetes,
				Values: ForwardValues{
					Labels: map[string]string{"app": "db"},
					Ports:  []string{"5432:5432"},
				},
			},
			{
				Name:   "website",
				Type:   ForwarderSSH,
				Values: ForwardValues{Remote: "user@host", Ports: []string{"8080:80"}},
			},
			{
				Name:   "search",
				Type:   ForwarderProxy,
				Values: ForwardValues{ProxyHostname: "search.example.com", Ports: []string{"9200:443"}},
			},
		},
	}
}

func TestValidate(t *testing.T) {
	testCases := []struct {
		name     string
		mutate   func(conf *Config)
		expected []string
	}{
		{
			name:   "valid configuration",
			mutate: func(conf *Config) {},
		},
		{
			name:     "no project",
			mutate:   func(conf *Config) { conf.Projects = nil },
			expected: []string{"no project defined, see the configuration examples at https://github.com/eko/monday/tree/master/example"},
		},
		{
			name: "duplicated and unnamed projects",
			mutate: func(conf *Config) {
				conf.Projects = append(conf.Projects, validProject(), &Project{})
			},
			expected: []string{
				`project "api": defined more than once`,
				"project #3: missing name",
			},
		},
		{
			name: "application without run command",
			mutate: func(conf *Config) {
				conf.Projects[0].Applications[0].Run = nil
			},
			expected: []string{`project "api": local application "api": missing run command`},
		},
		{
			name: "global application without name",
			mutate: func(conf *Config) {
				conf.Applications = []*Application{{Run: &Run{Command: "true"}}}
			},
			expected: []string{"global: local application #1: missing name"},
		},
		{
			name: "invalid files",
			mutate: func(conf *Config) {
				conf.Projects[0].Applications[0].Files = []*File{
					{Type: "symlink", To: "x"},
					{Type: FileTypeCopy},
				}
			},
			expected: []string{
				`project "api": local application "api": file #1: unknown type "symlink" (available: content, copy)`,
				`project "api": local application "api": file #2: missing destination path 'to'`,
				`project "api": local application "api": file #2: missing source path 'from'`,
			},
		},
		{
			name: "unknown forward type",
			mutate: func(conf *Config) {
				conf.Projects[0].Forwards[0].Type = "kube"
			},
			expected: []string{`project "api": forward "db": unknown type "kube" (available: kubernetes, kubernetes-remote, proxy, ssh, ssh-remote)`},
		},
		{
			name: "forward ports",
			mutate: func(conf *Config) {
				conf.Projects[0].Forwards[0].Values.Ports = nil
				conf.Projects[0].Forwards[1].Values.Ports = []string{"8080", "a:b"}
			},
			expected: []string{
				`project "api": forward "db": no port to forward, expected a list of "<local>:<remote>" mappings`,
				`project "api": forward "website": invalid port mapping "8080", expected "<local>:<remote>"`,
				`project "api": forward "website": invalid port mapping "a:b", expected "<local>:<remote>"`,
			},
		},
		{
			name: "forward type specific values",
			mutate: func(conf *Config) {
				conf.Projects[0].Forwards[0].Values.Labels = nil
				conf.Projects[0].Forwards[1].Values.Remote = ""
				conf.Projects[0].Forwards[2].Values.ProxyHostname = ""
			},
			expected: []string{
				`project "api": forward "db": missing labels selecting the pods to forward`,
				`project "api": forward "website": missing remote (<user>@<hostname>)`,
				`project "api": forward "search": missing proxy_hostname`,
			},
		},
		{
			name: "global forward without name",
			mutate: func(conf *Config) {
				conf.Forwards = []*Forward{{Type: ForwarderKubernetes}}
			},
			expected: []string{"global: forward #1: missing name"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			conf := &Config{Projects: []*Project{validProject()}}
			testCase.mutate(conf)

			err := conf.Validate()

			if testCase.expected == nil {
				assert.NoError(t, err)
				return
			}

			assert.EqualError(t, err, strings.Join(testCase.expected, "\n"))
		})
	}
}
