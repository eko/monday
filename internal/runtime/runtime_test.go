package runtime

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitRuntimeEditorWhenMethodNotCalled(t *testing.T) {
	// Given
	os.Setenv("MONDAY_EDITOR", "")

	// Then
	assert.Equal(t, "", EditorCommand)
}

func TestInitRuntimeEditorWhenGuessByEnvironment(t *testing.T) {
	// Then
	os.Setenv("MONDAY_EDITOR", "")

	// When
	initRuntimeEditor()

	// Then
	var editorIsFound = false
	if EditorCommand == "open" || EditorCommand == "gedit" || EditorCommand == "xed" {
		editorIsFound = true
	}

	assert.True(t, editorIsFound)
}

func TestInitRuntimeEditorWhenEnvironmentVariableIsSet(t *testing.T) {
	// Given
	os.Setenv("MONDAY_EDITOR", "code")

	// When
	initRuntimeEditor()

	// Then
	assert.Equal(t, "code", EditorCommand)
}
