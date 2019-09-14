package runtime

import (
	"fmt"
	"os"
	"os/exec"
	goRuntime "runtime"
)

var (
	// EditorCommand specified the editor that will be used to open configuration files
	EditorCommand string
)

// InitRuntimeEnvironment initializes runtime depending on user's environment
// such as OS and architecture
func InitRuntimeEnvironment() {
	initRuntimeEditor()
}

// initRuntimeEditor initializes the editor that will be used
// to manage configuration files
func initRuntimeEditor() {
	if value := os.Getenv("MONDAY_EDITOR"); value != "" {
		// In case environment variable is set, do not guess an editor
		EditorCommand = value
		return
	}

	switch goRuntime.GOOS {
	case "darwin":
		EditorCommand = "open"

	case "linux":
		editors := []string{"gedit", "xed"}

		for _, editor := range editors {
			_, err := exec.LookPath(editor)
			if err == nil {
				EditorCommand = editor
				break
			}
		}

	default:
		panic(fmt.Sprintf("Your operating (%s) system does not seems compatible. Sorry about that", goRuntime.GOOS))
	}
}
