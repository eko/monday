package copy

import (
	"io"
	"os"

	"github.com/eko/monday/pkg/config"
	"github.com/eko/monday/pkg/ui"
)

const (
	// HandlerType declares the copy file writter handler type name
	HandlerType = "copy"
)

// Handle handles a given File object in order to write it
func Handle(view ui.View, file *config.File, applicationName string) {
	var from = file.GetFrom()
	var to = file.GetTo()

	fromFile, err := os.Open(from)
	if err != nil {
		view.Writef("❌  Error while opening '%s' application source file '%s': %v\n", applicationName, from, err)
		return
	}
	defer fromFile.Close()

	// Create new file
	toFile, err := os.Create(to)
	if err != nil {
		view.Writef("❌  Error while creating '%s' application destination file '%s': %v\n", applicationName, to, err)
		return
	}
	defer toFile.Close()

	_, err = io.Copy(toFile, fromFile)
	if err != nil {
		view.Writef("❌  Error while copying '%s' application file '%s' to '%s': %v\n", applicationName, from, to, err)
		return
	}

	view.Writef("🗂  File '%s' successfully copied for application '%s'\n", to, applicationName)
}
