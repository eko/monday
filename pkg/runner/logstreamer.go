package runner

import (
	"bytes"
	"io"
	"os"
	"regexp"

	"github.com/eko/monday/pkg/ui"
)

const (
	StdOut = "stdout"
	StdErr = "stderr"
)

type Logstreamer struct {
	buf     *bytes.Buffer
	stdType string
	name    string

	colorOkay  string
	colorFail  string
	colorReset string

	view ui.ViewInterface
}

func NewLogstreamer(stdType string, name string, view ui.ViewInterface) *Logstreamer {
	streamer := &Logstreamer{
		buf:        bytes.NewBuffer([]byte("")),
		stdType:    stdType,
		name:       name,
		colorOkay:  "",
		colorFail:  "",
		colorReset: "",
		view:       view,
	}

	hasColors := regexp.MustCompile(`^(xterm|screen)`)
	if hasColors.MatchString(os.Getenv("TERM")) {
		streamer.colorOkay = "\x1b[32m"
		streamer.colorFail = "\x1b[31m"
		streamer.colorReset = "\x1b[0m"
	}

	return streamer
}

func (l *Logstreamer) Write(p []byte) (n int, err error) {
	if n, err = l.buf.Write(p); err != nil {
		return
	}

	err = l.output()
	return
}

func (l *Logstreamer) Close() {
	l.Flush()
	l.buf = bytes.NewBuffer([]byte(""))
}

func (l *Logstreamer) Flush() error {
	var p []byte
	if _, err := l.buf.Read(p); err != nil {
		return err
	}

	l.out(string(p))
	return nil
}

func (l *Logstreamer) output() (err error) {
	for {
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

func (l *Logstreamer) out(str string) (err error) {
	switch l.stdType {
	case StdOut:
		str = l.colorOkay + l.name + l.colorReset + " " + str

	case StdErr:
		str = l.colorFail + l.name + l.colorReset + " " + str

	default:
		str = l.stdType + str
	}

	l.view.Write(str)

	return nil
}
