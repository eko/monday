package kubernetes

import (
	"strings"

	"github.com/eko/monday/pkg/ui"
)

type Logstreamer struct {
	podName string
	view    ui.View
}

func NewLogstreamer(view ui.View, podName string) *Logstreamer {
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
