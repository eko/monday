package log

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

type Streamer struct {
	buf     *bytes.Buffer
	stdType string
	name    string

	view ui.View
}

func NewStreamer(stdType string, name string, view ui.View) *Streamer {
	streamer := &Streamer{
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
	l.Flush()
	l.buf = bytes.NewBuffer([]byte(""))
}

func (l *Streamer) Flush() error {
	var p []byte
	if _, err := l.buf.Read(p); err != nil {
		return err
	}

	l.out(string(p))
	return nil
}

func (l *Streamer) output() (err error) {
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

func (l *Streamer) out(str string) (err error) {
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
