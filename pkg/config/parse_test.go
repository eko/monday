package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {
	testCases := []struct {
		name              string
		sources           []Source
		expectedErr       string
		expectedErrPrefix string
		expectedIgnored   []string
		check             func(t *testing.T, conf *Config)
	}{
		{
			name: "anchors shared across files are resolved and the anchor keys are ignored",
			sources: []Source{
				{Name: "monday.local.yaml", Content: []byte(`
<: &api-local
  name: api
  hostname: api.svc.local
  run:
    command: go run main.go
`)},
				{Name: "monday.projects.yaml", Content: []byte(`
projects:
  - name: api
    local:
      - *api-local
`)},
			},
			check: func(t *testing.T, conf *Config) {
				require.Len(t, conf.Projects, 1)
				require.Len(t, conf.Projects[0].Applications, 1)
				assert.Equal(t, "api.svc.local", conf.Projects[0].Applications[0].Hostname)
				assert.Equal(t, "go run main.go", conf.Projects[0].Applications[0].Run.Command)
			},
		},
		{
			name: "projects split across several files are merged",
			sources: []Source{
				{Name: "monday.team-a.yaml", Content: []byte("projects:\n  - name: a\n")},
				{Name: "monday.team-b.yaml", Content: []byte("projects:\n  - name: b\n  - name: c\n")},
			},
			check: func(t *testing.T, conf *Config) {
				assert.Equal(t, []string{"a", "b", "c"}, conf.GetProjectNames())
			},
		},
		{
			name: "unknown top-level keys are ignored and reported",
			sources: []Source{
				{Name: "monday.yaml", Content: []byte(`
anchors:
  api: &api-local
    name: api
    run:
      command: go run main.go
projects:
  - name: api
    local:
      - *api-local
`)},
			},
			expectedIgnored: []string{"anchors"},
			check: func(t *testing.T, conf *Config) {
				require.Len(t, conf.Projects, 1)
				assert.Equal(t, "api", conf.Projects[0].Applications[0].Name)
			},
		},
		{
			name: "unknown nested field is reported with file, line and suggestion",
			sources: []Source{
				{Name: "monday.forward.yaml", Content: []byte("<: &db-forward\n  name: db\n  type: kubernetes\n")},
				{Name: "/home/user/monday.local.yaml", Content: []byte(`
<: &api-local
  name: api
  hostnme: api.svc.local
  run:
    command: go run main.go
    envs:
      FOO: bar
`)},
				{Name: "monday.projects.yaml", Content: []byte(`
projects:
  - name: api
    local:
      - *api-local
    forward:
      - *db-forward
`)},
			},
			expectedErr: `monday.local.yaml:4: unknown field "hostnme" in application (did you mean "hostname"?)` + "\n" +
				`monday.local.yaml:7: unknown field "envs" in run (did you mean "env"?)`,
		},
		{
			name: "unknown field inside a forward referenced through an anchor",
			sources: []Source{
				{Name: "monday.yaml", Content: []byte(`
<: &db-forward
  name: db
  type: kubernetes
  values:
    label:
      app: db
    ports:
      - 5432:5432
projects:
  - name: db
    forward:
      - *db-forward
`)},
			},
			expectedErr: `monday.yaml:6: unknown field "label" in forward values (did you mean "labels"?)`,
		},
		{
			name: "legacy application fields are rejected",
			sources: []Source{
				{Name: "monday.yaml", Content: []byte(`
<: &api-local
  name: api
  executable: go
  args:
    - run
projects:
  - name: api
    local:
      - *api-local
`)},
			},
			expectedErr: `monday.yaml:4: unknown field "executable" in application` + "\n" +
				`monday.yaml:5: unknown field "args" in application`,
		},
		{
			name: "duplicated settings across files are reported",
			sources: []Source{
				{Name: "monday.a.yaml", Content: []byte("gopath: /a\n")},
				{Name: "monday.b.yaml", Content: []byte("projects:\n  - name: b\ngopath: /b\n")},
			},
			expectedErr: `monday.b.yaml:3: mapping key "gopath" already defined at monday.a.yaml:1`,
		},
		{
			name: "syntax error is located in the right file",
			sources: []Source{
				{Name: "monday.a.yaml", Content: []byte("projects:\n  - name: a\n")},
				{Name: "monday.b.yaml", Content: []byte("watch:\n\texclude: []\n")},
			},
			expectedErrPrefix: "monday.b.yaml:2: ",
		},
		{
			name: "type mismatch is located",
			sources: []Source{
				{Name: "monday.yaml", Content: []byte("projects:\n  - name: a\n    local: not-a-list\n")},
			},
			expectedErr: "monday.yaml:3: cannot unmarshal !!str `not-a-list` into []*config.Application",
		},
		{
			name:    "empty configuration",
			sources: []Source{{Name: "monday.yaml", Content: []byte("# nothing yet\n")}},
			check: func(t *testing.T, conf *Config) {
				assert.Empty(t, conf.Projects)
			},
		},
		{
			name:        "configuration must be a mapping",
			sources:     []Source{{Name: "monday.yaml", Content: []byte("- a\n- b\n")}},
			expectedErr: "monday.yaml:1: the configuration must be a YAML mapping",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			conf, ignored, err := Parse(testCase.sources)

			if testCase.expectedErr != "" {
				assert.EqualError(t, err, testCase.expectedErr)
				assert.Nil(t, conf)
				return
			}

			if testCase.expectedErrPrefix != "" {
				require.Error(t, err)
				assert.True(t, strings.HasPrefix(err.Error(), testCase.expectedErrPrefix), err.Error())
				assert.Nil(t, conf)
				return
			}

			require.NoError(t, err)

			if testCase.expectedIgnored == nil {
				assert.Empty(t, ignored)
			} else {
				assert.Equal(t, testCase.expectedIgnored, ignored)
			}

			if testCase.check != nil {
				testCase.check(t, conf)
			}
		})
	}
}

// TestParseExampleDirectory keeps the documented examples loadable
func TestParseExampleDirectory(t *testing.T) {
	// Given
	dir, err := os.Getwd()
	require.NoError(t, err)

	files, err := filepath.Glob(dir + "/../../example/monday.*.yaml")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	sources := make([]Source, 0, len(files))
	for _, file := range files {
		content, err := os.ReadFile(file)
		require.NoError(t, err)

		sources = append(sources, Source{Name: file, Content: content})
	}

	// When
	conf, ignored, err := Parse(sources)

	// Then
	require.NoError(t, err)
	assert.Empty(t, ignored)
	assert.NoError(t, conf.Validate())

	assert.Equal(t, []string{"full", "graphql", "forward-only", "forward-composieux-website"}, conf.GetProjectNames())
}

func TestEditDistance(t *testing.T) {
	testCases := []struct {
		a, b string
		want int
	}{
		{a: "hostname", b: "hostname", want: 0},
		{a: "hostnme", b: "hostname", want: 1},
		{a: "label", b: "labels", want: 1},
		{a: "", b: "abc", want: 3},
		{a: "kitten", b: "sitting", want: 3},
	}

	for _, testCase := range testCases {
		t.Run(testCase.a+"/"+testCase.b, func(t *testing.T) {
			assert.Equal(t, testCase.want, editDistance(testCase.a, testCase.b))
		})
	}
}
