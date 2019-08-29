package ui

import (
	"fmt"

	"github.com/jroimartin/gocui"
)

type ViewInterface interface {
	GetName() string
	Write(str string)
	Writef(str string, args ...interface{})
}

// View is the view structure
type View struct {
	name  string
	title string
	view  *gocui.View
}

// NewView returns a new instance of a view
func NewView(name, title string, view *gocui.View) *View {
	return &View{
		name:  name,
		title: title,
		view:  view,
	}
}

// NewEmptyView returns a new instance of an empty view
func NewEmptyView(name string) *View {
	return &View{
		name: name,
	}
}

// GetName returns the name of the view
func (v *View) GetName() string {
	return v.name
}

// GetTitle returns the title of the view
func (v *View) GetTitle() string {
	return v.title
}

// GetView returns the GoCUI view structure
func (v *View) GetView() *gocui.View {
	return v.view
}

// Write allows to write a string to the view
func (v *View) Write(str string) {
	if v.view == nil {
		fmt.Print(str)
		return
	}

	v.view.Write([]byte(str))
}

// Writef allows to write a string to the view with some given arguments
func (v *View) Writef(str string, args ...interface{}) {
	if v.view == nil {
		fmt.Print(fmt.Sprintf(str, args...))
		return
	}

	v.view.Write([]byte(fmt.Sprintf(str, args...)))
}
