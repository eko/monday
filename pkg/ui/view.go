package ui

import (
	"fmt"

	"github.com/jroimartin/gocui"
)

type View interface {
	GetName() string
	Write(str string)
	Writef(str string, args ...interface{})
}

// view is the view structure
type view struct {
	name  string
	title string
	view  *gocui.View
}

// NewView returns a new instance of a view
func NewView(name, title string, v *gocui.View) *view {
	return &view{
		name:  name,
		title: title,
		view:  v,
	}
}

// NewEmptyView returns a new instance of an empty view
func NewEmptyView(name string) *view {
	return &view{
		name: name,
	}
}

// GetName returns the name of the view
func (v *view) GetName() string {
	return v.name
}

// GetTitle returns the title of the view
func (v *view) GetTitle() string {
	return v.title
}

// GetView returns the GoCUI view structure
func (v *view) GetView() *gocui.View {
	return v.view
}

// Write allows to write a string to the view
func (v *view) Write(str string) {
	if v.view == nil {
		fmt.Print(str)
		return
	}

	v.view.Write([]byte(str))
}

// Writef allows to write a string to the view with some given arguments
func (v *view) Writef(str string, args ...interface{}) {
	if v.view == nil {
		fmt.Print(fmt.Sprintf(str, args...))
		return
	}

	v.view.Write([]byte(fmt.Sprintf(str, args...)))
}
