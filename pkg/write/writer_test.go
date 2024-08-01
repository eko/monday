package write

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/ui"
	"github.com/eko/monday/pkg/write/content"
	"github.com/eko/monday/pkg/write/copy"
	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewWriter(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	project := getProjectMock()

	// When
	w := NewWriter(view, project)

	// Then
	assert.IsType(t, new(writer), w)
	assert.Implements(t, new(Writer), w)

	assert.Equal(t, view, w.view)
	assert.Equal(t, project, w.project)
}

func TestWriteWhenTypeCopy(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileToCopy := &config.File{
		Type: copy.HandlerType,
		From: getTestsPath() + "write/files/file1.txt",
		To:   getTestsPath() + "write/file1-output.txt",
	}

	defer os.Remove(fileToCopy.GetTo())

	project := getProjectMock()
	project.Applications[0].Files = []*config.File{fileToCopy}

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef(
		"🗂  File '%s' successfully copied for application '%s'\n",
		fileToCopy.GetTo(),
		"test-app",
	)

	writer := NewWriter(view, project)

	// When
	writer.WriteAll()

	// Then
	assertFileContent(t, getTestsPath()+"write/file1-output.txt", "This is my file1")
}

func TestWriteWhenTypeContent(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	fileContent := &config.File{
		Type: content.HandlerType,
		To:   getTestsPath() + "write/file2-output.txt",
		Content: `
This is my test file content and here are the project applications:
{{- range $app := .Applications }}
Name: {{ $app.Name }}
{{- end }}
`,
	}

	defer os.Remove(fileContent.GetTo())

	project := getProjectMock()
	project.Applications[0].Files = []*config.File{fileContent}

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef(
		"🗂  File '%s' for application '%s' written\n",
		fileContent.GetTo(),
		"test-app",
	)

	writer := NewWriter(view, project)

	// When
	writer.WriteAll()

	// Then
	assertFileContent(t, getTestsPath()+"write/file2-output.txt", `
This is my test file content and here are the project applications:
Name: test-app
`)
}

func assertFileContent(t *testing.T, filepath, expected string) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		t.Fatal(fmt.Sprintf("Cannot read file '%s': %v", filepath, err))
	}

	assert.Equal(t, expected, string(data))
}

func getProjectMock() *config.Project {
	path := getTestsPath()

	return &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			{
				Name:  "test-app",
				Path:  path,
				Watch: true,
			},
		},
	}
}

func getTestsPath() string {
	dir, _ := os.Getwd()
	return dir + "/../../internal/test/"
}
