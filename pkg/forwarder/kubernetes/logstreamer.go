package kubernetes

import (
	"fmt"
	"strings"
)

type Logstreamer struct {
	podName string
}

func NewLogstreamer(podName string) *Logstreamer {
	return &Logstreamer{
		podName: podName,
	}
}

func (l *Logstreamer) Write(b []byte) (int, error) {
	line := string(b)
	strings.TrimSuffix(line, "\n")

	fmt.Printf("%s %s", l.podName, line)

	return 0, nil
}
