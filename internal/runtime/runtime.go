package runtime

import (
	"fmt"
	"os"
	"os/exec"
	goRuntime "runtime"
	"strings"
)

var (
	// EditorCommand specified the editor that will be used to open configuration files
	EditorCommand string

	// EditorArgs specified the editor arguments that will be sent when opening editor
	EditorArgs []string

	// Private available editor with args
	editors = []string{"open", "gedit", "xed"}
	args    = map[string][]string{
		"open": []string{"-t"}, // By default, when using open, use the "-t" option to specify we want to use the default text editor
	}
)

// InitRuntimeEnvironment initializes runtime depending on user's environment
// such as OS and architecture
func InitRuntimeEnvironment() {
	initRuntimeEditor()
}

// initRuntimeEditor initializes the editor that will be used
// to manage configuration files
func initRuntimeEditor() {
	// In case environment variables are set, use the specified editor with args
	if customEditor := os.Getenv("MONDAY_EDITOR"); customEditor != "" {
		EditorCommand = customEditor

		if customEditorArgs := os.Getenv("MONDAY_EDITOR_ARGS"); customEditorArgs != "" {
			EditorArgs = strings.Split(customEditorArgs, ",")
		}

		return
	}

	// Don't have a custom editor, guess it depending on the environment
	switch goRuntime.GOOS {
	case "darwin", "linux":
		for _, editor := range editors {
			_, err := exec.LookPath(editor)
			if err != nil {
				continue
			}

			EditorCommand = editor

			if value, ok := args[editor]; ok {
				EditorArgs = value
			}

			break
		}

	default:
		panic(fmt.Sprintf("Your operating system (%s) does not seems compatible. Sorry about that", goRuntime.GOOS))
	}
}
