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

func TestInitRuntimeEditorWhenEnvironmentVariableIsSet(t *testing.T) {
	// Given
	os.Setenv("MONDAY_EDITOR", "code")

	// When
	initRuntimeEditor()

	// Then
	assert.Equal(t, "code", EditorCommand)
}
