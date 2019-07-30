package kubernetes

import (
	"strings"

	"github.com/eko/monday/internal/ui"
)

type Logstreamer struct {
	podName string
	view    ui.ViewInterface
}

func NewLogstreamer(view ui.ViewInterface, podName string) *Logstreamer {
	return &Logstreamer{
		podName: podName,
		view:    view,
	}
}

func (l *Logstreamer) Write(b []byte) (int, error) {
	line := string(b)
	strings.TrimSuffix(line, "\n")

	l.view.Writef("%s %s", l.podName, line)

	return 0, nil
}
