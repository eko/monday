package run

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

	ColorGreen = "\x1b[32m"
	ColorRed   = "\x1b[31m"
	ColorWhite = "\x1b[0m"
)

var (
	ColorOkay  = ""
	ColorFail  = ""
	ColorReset = ""
)

type Logstreamer struct {
	buf     *bytes.Buffer
	stdType string
	name    string

	view ui.View
}

func NewLogstreamer(stdType string, name string, view ui.View) *Logstreamer {
	streamer := &Logstreamer{
		buf:     bytes.NewBuffer([]byte("")),
		stdType: stdType,
		name:    name,
		view:    view,
	}

	if hasColors := regexp.MustCompile(`^(xterm|screen)`); hasColors.MatchString(os.Getenv("TERM")) {
		ColorOkay = ColorGreen
		ColorFail = ColorRed
		ColorReset = ColorWhite
	}

	return streamer
}

func (l *Logstreamer) Write(p []byte) (n int, err error) {
	if n, err = l.buf.Write(p); err != nil {
		return
	}

	err = l.output()
	if err != nil {
		panic(err)
	}
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
		str = ColorOkay + l.name + ColorReset + " " + str

	case StdErr:
		str = ColorFail + l.name + ColorReset + " " + str

	default:
		str = l.stdType + str
	}

	l.view.Write(str)

	return nil
}
