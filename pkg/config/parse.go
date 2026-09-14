package config

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// anchorsKey is the conventional top-level key holding the YAML anchors
	// reused across projects, it is never reported as ignored
	anchorsKey = "<"

	// maxSuggestionDistance is the maximum edit distance between an unknown
	// field and a known one to suggest it
	maxSuggestionDistance = 2
)

var (
	lineRegexp = regexp.MustCompile(`\bline (\d+)\b`)

	typeLabels = map[string]string{
		"Config":        "configuration",
		"GlobalBuild":   "build section",
		"GlobalRun":     "run section",
		"GlobalSetup":   "setup section",
		"GlobalWatch":   "watch section",
		"Project":       "project",
		"Application":   "application",
		"Build":         "build",
		"Run":           "run",
		"Setup":         "setup",
		"File":          "file",
		"Forward":       "forward",
		"ForwardValues": "forward values",
		"Monitoring":    "monitoring",
	}
)

// Source is the content of a configuration file, several sources are merged
// into a single YAML document so anchors can be shared between files
type Source struct {
	Name    string
	Content []byte
}

// document is the YAML document resulting from the merge of several sources,
// keeping track of where each source starts to report errors on the right file
type document struct {
	content []byte
	sources []Source
	offsets []int
}

// Parse merges the given sources into a single YAML document and decodes it
// strictly: an unknown field in a project, an application or a forward is
// reported with its file and line instead of being silently ignored.
//
// Top-level keys that are not part of the configuration are ignored, as they
// conventionally hold the YAML anchors reused in projects: they are returned
// so callers can display them.
func Parse(sources []Source) (*Config, []string, error) {
	doc := mergeSources(sources)

	var root yaml.Node
	if err := yaml.Unmarshal(doc.content, &root); err != nil {
		return nil, nil, doc.translate(err)
	}

	mapping, empty := documentMapping(&root)
	if empty {
		return &Config{}, nil, nil
	}

	if mapping.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("%s: the configuration must be a YAML mapping", doc.locate(mapping.Line))
	}

	filtered, ignored := filterTopLevel(mapping)

	if problems := unknownFields(filtered, reflect.TypeOf(Config{})); len(problems) > 0 {
		return nil, ignored, doc.translate(errors.New(strings.Join(problems, "\n")))
	}

	conf := &Config{}
	if err := filtered.Decode(conf); err != nil {
		return nil, ignored, doc.translate(err)
	}

	return conf, ignored, nil
}

func mergeSources(sources []Source) *document {
	doc := &document{sources: sources}

	var buffer bytes.Buffer
	line := 1

	for _, source := range sources {
		doc.offsets = append(doc.offsets, line)

		buffer.Write(source.Content)
		line += bytes.Count(source.Content, []byte{'\n'})

		if len(source.Content) > 0 && source.Content[len(source.Content)-1] != '\n' {
			buffer.WriteByte('\n')
			line++
		}
	}

	doc.content = buffer.Bytes()

	return doc
}

// locate translates a line number of the merged document into "file:line"
func (d *document) locate(line int) string {
	if len(d.sources) == 0 {
		return fmt.Sprintf("line %d", line)
	}

	index := sort.Search(len(d.offsets), func(i int) bool {
		return d.offsets[i] > line
	}) - 1

	if index < 0 {
		index = 0
	}

	return fmt.Sprintf("%s:%d", filepath.Base(d.sources[index].Name), line-d.offsets[index]+1)
}

// translate rewrites the line numbers of a YAML error into file and line
// references, and strips the library prefixes to keep messages readable
func (d *document) translate(err error) error {
	message := strings.TrimPrefix(err.Error(), "yaml: ")
	message = strings.TrimPrefix(message, "unmarshal errors:\n")

	lines := strings.Split(message, "\n")
	for i, line := range lines {
		lines[i] = lineRegexp.ReplaceAllStringFunc(strings.TrimSpace(line), func(match string) string {
			number, convErr := strconv.Atoi(strings.TrimPrefix(match, "line "))
			if convErr != nil {
				return match
			}

			return d.locate(number)
		})
	}

	return errors.New(strings.Join(lines, "\n"))
}

// documentMapping returns the root node of the document, and whether the
// document is empty (no content, or comments only)
func documentMapping(root *yaml.Node) (*yaml.Node, bool) {
	if root.Kind != yaml.DocumentNode || len(root.Content) == 0 {
		return nil, true
	}

	mapping := root.Content[0]
	if mapping.Kind == yaml.ScalarNode && mapping.Tag == "!!null" {
		return nil, true
	}

	return mapping, false
}

// filterTopLevel keeps the top-level keys that are part of the configuration
// and merges the sequences defined several times (for instance projects split
// across several files). The other keys are returned as ignored.
func filterTopLevel(mapping *yaml.Node) (*yaml.Node, []string) {
	known := yamlFields(reflect.TypeOf(Config{}))

	filtered := &yaml.Node{
		Kind:   yaml.MappingNode,
		Tag:    mapping.Tag,
		Line:   mapping.Line,
		Column: mapping.Column,
	}

	positions := make(map[string]int)
	seen := make(map[string]bool)
	ignored := make([]string, 0)

	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]

		if _, ok := known[key.Value]; !ok {
			if key.Value != anchorsKey && !seen[key.Value] {
				seen[key.Value] = true
				ignored = append(ignored, key.Value)
			}

			continue
		}

		if index, ok := positions[key.Value]; ok {
			previous := resolveAlias(filtered.Content[index])
			current := resolveAlias(value)

			if previous.Kind == yaml.SequenceNode && current.Kind == yaml.SequenceNode {
				filtered.Content[index] = mergeSequences(previous, current)
				continue
			}
		}

		positions[key.Value] = len(filtered.Content) + 1
		filtered.Content = append(filtered.Content, key, value)
	}

	return filtered, ignored
}

func mergeSequences(first, second *yaml.Node) *yaml.Node {
	content := make([]*yaml.Node, 0, len(first.Content)+len(second.Content))
	content = append(content, first.Content...)
	content = append(content, second.Content...)

	return &yaml.Node{
		Kind:    yaml.SequenceNode,
		Tag:     first.Tag,
		Line:    first.Line,
		Column:  first.Column,
		Content: content,
	}
}

// unknownFields walks the YAML node against the given Go type and reports the
// mapping keys that do not match any field, with a suggestion when a known
// field looks alike
func unknownFields(node *yaml.Node, typ reflect.Type) []string {
	problems := make([]string, 0)
	walkUnknownFields(node, typ, &problems)

	return problems
}

func walkUnknownFields(node *yaml.Node, typ reflect.Type, problems *[]string) {
	node = resolveAlias(node)
	if node == nil {
		return
	}

	for typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	switch typ.Kind() {
	case reflect.Struct:
		walkStructFields(node, typ, problems)

	case reflect.Slice, reflect.Array:
		if node.Kind != yaml.SequenceNode {
			return
		}

		for _, item := range node.Content {
			walkUnknownFields(item, typ.Elem(), problems)
		}

	case reflect.Map:
		if node.Kind != yaml.MappingNode {
			return
		}

		for i := 1; i < len(node.Content); i += 2 {
			walkUnknownFields(node.Content[i], typ.Elem(), problems)
		}
	}
}

func walkStructFields(node *yaml.Node, typ reflect.Type, problems *[]string) {
	switch node.Kind {
	case yaml.SequenceNode:
		// A merge key can reference a list of mappings
		for _, item := range node.Content {
			walkUnknownFields(item, typ, problems)
		}

		return

	case yaml.MappingNode:
		// Continue below

	default:
		return
	}

	fields := yamlFields(typ)

	for i := 0; i+1 < len(node.Content); i += 2 {
		key, value := node.Content[i], node.Content[i+1]

		if key.Tag == "!!merge" {
			walkUnknownFields(value, typ, problems)
			continue
		}

		field, ok := fields[key.Value]
		if !ok {
			*problems = append(*problems, fmt.Sprintf(
				"line %d: unknown field %q in %s%s",
				key.Line,
				key.Value,
				describeType(typ),
				suggestion(key.Value, fields),
			))

			continue
		}

		walkUnknownFields(value, field, problems)
	}
}

// yamlFields returns the YAML keys accepted by a struct with their Go types
func yamlFields(typ reflect.Type) map[string]reflect.Type {
	fields := make(map[string]reflect.Type)

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}

		name, inline := parseYAMLTag(field.Tag.Get("yaml"))
		if name == "-" {
			continue
		}

		if inline {
			inlined := field.Type
			for inlined.Kind() == reflect.Ptr {
				inlined = inlined.Elem()
			}

			if inlined.Kind() == reflect.Struct {
				for key, value := range yamlFields(inlined) {
					fields[key] = value
				}
			}

			continue
		}

		if name == "" {
			name = strings.ToLower(field.Name)
		}

		fields[name] = field.Type
	}

	return fields
}

func parseYAMLTag(tag string) (string, bool) {
	parts := strings.Split(tag, ",")

	inline := false
	for _, option := range parts[1:] {
		if option == "inline" {
			inline = true
		}
	}

	return parts[0], inline
}

func describeType(typ reflect.Type) string {
	if label, ok := typeLabels[typ.Name()]; ok {
		return label
	}

	return strings.ToLower(typ.Name())
}

// suggestion returns a "did you mean" hint when a known field is close to the
// given unknown one
func suggestion(unknown string, fields map[string]reflect.Type) string {
	best, bestDistance := "", maxSuggestionDistance+1

	for name := range fields {
		distance := editDistance(strings.ToLower(unknown), name)
		if distance < bestDistance || (distance == bestDistance && name < best) {
			best, bestDistance = name, distance
		}
	}

	if best == "" {
		return ""
	}

	return fmt.Sprintf(" (did you mean %q?)", best)
}

// editDistance returns the Levenshtein distance between two strings
func editDistance(a, b string) int {
	first, second := []rune(a), []rune(b)

	previous := make([]int, len(second)+1)
	current := make([]int, len(second)+1)

	for j := range previous {
		previous[j] = j
	}

	for i := 1; i <= len(first); i++ {
		current[0] = i

		for j := 1; j <= len(second); j++ {
			cost := 1
			if first[i-1] == second[j-1] {
				cost = 0
			}

			current[j] = min(previous[j]+1, current[j-1]+1, previous[j-1]+cost)
		}

		previous, current = current, previous
	}

	return previous[len(second)]
}

func resolveAlias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}

	return node
}
