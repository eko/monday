package build

import (
	"sync"

	"github.com/eko/monday/pkg/build/command"
	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/helper"
	"github.com/eko/monday/pkg/ui"
)

// Builder represents a local application builder
type Builder interface {
	BuildAll()
	Build(application *config.Application)
}

type builder struct {
	projectName  string
	applications []*config.Application
	view         ui.View
	conf         *config.GlobalBuild
}

// NewBuilder instanciates a new builder instance
func NewBuilder(view ui.View, project *config.Project, conf *config.GlobalBuild) *builder {
	return &builder{
		projectName:  project.Name,
		applications: project.Applications,
		view:         view,
		conf:         conf,
	}
}

// BuildAll builds all local applications in separated goroutines
func (b *builder) BuildAll() {
	var wg = sync.WaitGroup{}

	for _, application := range b.applications {
		wg.Add(1)
		go func(application *config.Application) {
			defer wg.Done()
			b.Build(application)
		}(application)
	}

	wg.Wait()
}

// Build builds the application
func (b *builder) Build(application *config.Application) {
	if application.Build == nil {
		return
	}

	if err := helper.CheckPathExists(application.GetPath()); err != nil {
		b.view.Writef("❌  %s\n", err.Error())
		return
	}

	var build = application.Build
	var err error

	b.view.Writef("⚙️   Building application '%s' via %s...\n", application.Name, build.Type)

	switch build.Type {
	case command.BuilderType:
		err = command.Build(application, b.view, b.conf)

	default:
		err = command.Build(application, b.view, b.conf)
	}

	if err != nil {
		b.view.Writef("❌  Error while building application '%s': %v\n", application.Name, err)
		return
	}

	b.view.Writef("\n✅  Build of application '%s' complete!\n\n", application.Name)
}
