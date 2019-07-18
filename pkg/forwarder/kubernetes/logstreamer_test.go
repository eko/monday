package kubernetes

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewLogstreamer(t *testing.T) {
	// Given
	podName := "my-test-pod"

	// When
	streamer := NewLogstreamer(podName)

	// Then
	assert.IsType(t, new(Logstreamer), streamer)

	assert.Equal(t, podName, streamer.podName)
}

func TestWrite(t *testing.T) {
	// When
	streamer := NewLogstreamer("my-test-pod")

	// Then
	output := captureStdout(func() {
		streamer.Write([]byte("This is a sample log from my unit test"))
	})

	// Tten
	assert.Equal(t, "my-test-pod This is a sample log from my unit test", output)
}

// This function allows to capture the stdout with all fmt.Print() lines
// to a single string
func captureStdout(f func()) string {
	old := os.Stdout
	reader, writer, _ := os.Pipe()

	os.Stdout = writer

	f()

	writer.Close()
	os.Stdout = old

	var buffer bytes.Buffer
	io.Copy(&buffer, reader)

	return buffer.String()
}
