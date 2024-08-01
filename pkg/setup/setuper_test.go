package setup

import (
	"testing"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/log"
	"github.com/eko/monday/pkg/ui"
	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestNewSetuper(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)

	project := getMockedProjectWithApplication()

	// When
	s := NewSetuper(view, project, &config.GlobalSetup{})

	// Then
	assert.IsType(t, new(setuper), s)
	assert.Implements(t, new(Setuper), s)

	assert.Equal(t, project.Name, s.projectName)
	assert.Equal(t, project.Applications, s.applications)
}

func TestSetupAll(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("⚙️  Setuping application '%s'...\n", "test-app")
	view.EXPECT().Writef("👉  Running commands:\n%s\n\n", "echo Starting test command setup...\necho ...and a second setup command to confirm it works")
	view.EXPECT().Write(log.ColorGreen + "test-app" + log.ColorWhite + " Starting test command setup...\n")
	view.EXPECT().Write(log.ColorGreen + "test-app" + log.ColorWhite + " ...and a second setup command to confirm it works\n")
	view.EXPECT().Write("\n✅  Setup of application complete!\n\n")

	project := &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			{
				Name: "test-app",
				Path: "/unkown/directory",
				Setup: &config.Setup{
					Commands: []string{
						"echo Starting test command setup...",
						"echo ...and a second setup command to confirm it works",
					},
				},
			},
		},
	}

	setuper := NewSetuper(view, project, &config.GlobalSetup{})

	// When - Then
	setuper.SetupAll()
}

func getMockedProjectWithApplication() *config.Project {
	return &config.Project{
		Name: "My project name",
		Applications: []*config.Application{
			{
				Name: "test-app",
				Path: "/",
				Run: &config.Run{
					Command: "echo OK Arguments Seems -to=work",
				},
			},
		},
	}
}
