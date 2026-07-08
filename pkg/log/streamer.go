package log

import (
	"bytes"
	"hash/fnv"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/eko/monday/pkg/ui"
)

const (
	StdOut = "stdout"
	StdErr = "stderr"
)

var (
	// namePalette holds the colors assigned to application/pod names, each name
	// keeps a stable color so it can be identified at a glance in the logs
	namePalette = []lipgloss.Color{
		"#9D86F9", // violet
		"#56B6C2", // cyan
		"#98C379", // green
		"#E5C07B", // yellow
		"#61AFEF", // blue
		"#C678DD", // magenta
		"#D19A66", // orange
		"#5FD7A7", // mint
	}

	stderrNameStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#E06C75"))
)

type Streamer struct {
	buf     *bytes.Buffer
	stdType string
	name    string
	prefix  string

	view ui.View
}

func NewStreamer(stdType string, name string, view ui.View) *Streamer {
	return &Streamer{
		buf:     bytes.NewBuffer([]byte("")),
		stdType: stdType,
		name:    name,
		prefix:  Prefix(stdType, name),
		view:    view,
	}
}

// Prefix returns the rendered log prefix: a stable color per name for standard
// output, red for standard error
func Prefix(stdType, name string) string {
	switch stdType {
	case StdOut:
		return nameStyle(name).Render(name) + " "
	case StdErr:
		return stderrNameStyle.Render(name) + " "
	default:
		return stdType
	}
}

// nameStyle returns the style assigned to the given name, stable across calls
func nameStyle(name string) lipgloss.Style {
	hash := fnv.New32a()
	hash.Write([]byte(name))

	color := namePalette[int(hash.Sum32())%len(namePalette)]

	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

func (l *Streamer) Write(p []byte) (n int, err error) {
	if n, err = l.buf.Write(p); err != nil {
		return
	}

	err = l.output()
	if err != nil {
		panic(err)
	}
	return
}

func (l *Streamer) Close() {
	_ = l.Flush()
	l.buf = nil
}

func (l *Streamer) Flush() error {
	if l.buf == nil {
		return nil
	}

	var p []byte
	if _, err := l.buf.Read(p); err != nil {
		return err
	}

	l.out(string(p))
	return nil
}

func (l *Streamer) output() (err error) {
	for l.buf != nil {
		line, err := l.buf.ReadString('\n')
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		l.out(line)
	}

	return nil
}

func (l *Streamer) out(str string) {
	l.view.Write(l.prefix + str)
}
