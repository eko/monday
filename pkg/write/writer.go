package write

import (
	"sync"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/ui"
	"github.com/eko/monday/pkg/write/content"
	"github.com/eko/monday/pkg/write/copy"
)

// Writer represents a file, memory or something else writer
type Writer interface {
	WriteAll()
	Write(application *config.Application)
}

type writer struct {
	view    ui.View
	project *config.Project
}

// NewWriter instanciates a new writer instance
func NewWriter(view ui.View, project *config.Project) *writer {
	return &writer{
		view:    view,
		project: project,
	}
}

// WriteAll writes all memory, files or anything else that have to be written
func (w *writer) WriteAll() {
	var wg = sync.WaitGroup{}

	for _, application := range w.project.Applications {
		wg.Add(1)
		go func(application *config.Application) {
			defer wg.Done()
			w.Write(application)
		}(application)
	}

	wg.Wait()
}

// Write writes all the application-related objects
func (w *writer) Write(application *config.Application) {
	if len(application.Files) == 0 {
		return
	}

	for _, file := range application.Files {
		switch file.Type {
		case copy.HandlerType:
			copy.Handle(w.view, file, application.Name)

		case content.HandlerType:
			content.Handle(w.view, w.project, file, application.Name)

		default:
			w.view.Writef("❌  File type '%s' declared for application '%s' to does not exists\n", file.Type, application.Name)
			return
		}
	}
}
