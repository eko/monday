package content

import (
	"os"
	"text/template"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/ui"
)

const (
	// HandlerType declares the content file writter handler type name
	HandlerType = "content"
)

// Handle handles a given File object in order to write it
func Handle(view ui.View, project *config.Project, file *config.File, applicationName string) {
	var to = file.GetTo()

	f, err := os.Create(to)
	if err != nil {
		view.Writef("❌  Error while creating '%s' application file '%s': %v\n", applicationName, to, err)
		return
	}
	defer f.Close()

	t := template.Must(template.New(to).Parse(file.Content))

	if err := t.Execute(f, project); err != nil {
		view.Writef("❌  Error while writting '%s' application file: %v\n", applicationName, err)
		return
	}

	view.Writef("🗂  File '%s' for application '%s' written\n", to, applicationName)
}
