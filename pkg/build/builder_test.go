package build

import (
	"testing"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/log"
	"github.com/eko/monday/pkg/ui"
	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewBuilder(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)

	project := getMockedProjectWithApplication()

	// When
	b := NewBuilder(view, project, &config.GlobalBuild{})

	// Then
	assert.IsType(t, new(builder), b)
	assert.Implements(t, new(Builder), b)

	assert.Equal(t, view, b.view)
	assert.Equal(t, project.Name, b.projectName)
	assert.Equal(t, project.Applications, b.applications)
}

func TestBuildAll(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("⚙️   Building application '%s' via %s...\n", "test-app", "command")
	view.EXPECT().Writef("👉  Running commands:\n%s\n", "echo 'ok it works'\necho yes it's ok")
	view.EXPECT().Write(log.ColorGreen + "test-app" + log.ColorWhite + " 'ok it works'\n")
	view.EXPECT().Write(log.ColorGreen + "test-app" + log.ColorWhite + " yes it's ok\n")
	view.EXPECT().Writef("\n✅  Build of application '%s' complete!\n\n", "test-app")

	project := getMockedProjectWithApplication()

	builder := NewBuilder(view, project, &config.GlobalBuild{})

	// When - Then
	builder.BuildAll()
}

func getMockedProjectWithApplication() *config.Project {
	return &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			{
				Name: "test-app",
				Path: "/",

				Build: &config.Build{
					Type: "command",
					Commands: []string{
						"echo 'ok it works'",
						"echo yes it's ok",
					},
				},
			},
		},
	}
}
