package command

import (
	"testing"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/log"
	"github.com/eko/monday/pkg/ui"
	"go.uber.org/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestBuild(t *testing.T) {
	// Given
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	view := ui.NewMockView(ctrl)
	view.EXPECT().Writef("👉  Running commands:\n%s\n", "echo 'ok it works'\necho yes it's ok")
	view.EXPECT().Write(log.ColorGreen + "test-app" + log.ColorWhite + " 'ok it works'\n")
	view.EXPECT().Write(log.ColorGreen + "test-app" + log.ColorWhite + " yes it's ok\n")

	application := getMockedApplication()

	// When
	err := Build(application, view, &config.GlobalBuild{})

	// Then
	assert := assert.New(t)
	assert.Nil(err)
}

func getMockedApplication() *config.Application {
	return &config.Application{
		Name: "test-app",
		Path: "/",

		Build: &config.Build{
			Type: "command",
			Commands: []string{
				"echo 'ok it works'",
				"echo yes it's ok",
			},
		},
	}
}
