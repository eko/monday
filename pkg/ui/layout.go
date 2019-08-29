package ui

import (
	"fmt"
	"time"

	"github.com/jroimartin/gocui"
)

// Layout is the structure of the gui layout
type Layout struct {
	uiEnabled      bool
	gui            *gocui.Gui
	highlighted    *View
	statusView     *View
	fullscreenView *View
	logsView       *View
	forwardsView   *View
	proxyView      *View
	viewsOrder     map[string]*View
}

// NewLayout returns a new layout instance
func NewLayout(uiEnabled bool) *Layout {
	layout := &Layout{
		uiEnabled: uiEnabled,
	}

	if uiEnabled {
		gui, err := gocui.NewGui(gocui.OutputNormal)
		if err != nil {
			panic(err)
		}

		layout.gui = gui
	}

	return layout
}

// Init initializes the gui layout
func (l *Layout) Init() {
	if !l.uiEnabled {
		l.statusView = NewEmptyView("status")
		l.fullscreenView = NewEmptyView("fullscreen")
		l.logsView = NewEmptyView("logs")
		l.forwardsView = NewEmptyView("forwards")
		l.proxyView = NewEmptyView("proxy")

		return
	}

	maxX, maxY := l.gui.Size()

	statusView, err := l.setStatusView("status", 0, 0, maxX-1, 2)
	if err != nil {
		panic(err)
	}
	l.statusView = statusView

	fullscreenView, err := l.setView("fullscreen", "", -1, -1, maxX, maxY)
	if err != nil {
		panic(err)
	}
	l.fullscreenView = fullscreenView
	l.fullscreenView.GetView().Frame = false
	l.gui.SetViewOnBottom("fullscreen")

	logsView, err := l.setView("logs", " Logs ", 0, 3, maxX-1, (maxY/2)+9)
	if err != nil {
		panic(err)
	}
	l.logsView = logsView
	l.logsView.GetView().Title = fmt.Sprintf("%s (Current)", l.logsView.GetTitle())

	forwardsView, err := l.setView("forwards", " Forwards ", 0, (maxY/2)+10, (maxX/2)-1, maxY-1)
	if err != nil {
		panic(err)
	}
	l.forwardsView = forwardsView

	proxyView, err := l.setView("proxy", " Proxy ", maxX/2, (maxY/2)+10, maxX-1, maxY-1)
	if err != nil {
		panic(err)
	}
	l.proxyView = proxyView

	l.viewsOrder = map[string]*View{
		logsView.GetName():     forwardsView,
		forwardsView.GetName(): proxyView,
		proxyView.GetName():    logsView,
	}

	l.highlighted = logsView
	l.setKeyBindings()

	go func() {
		for {
			l.gui.Update(func(g *gocui.Gui) error {
				return nil
			})

			time.Sleep(time.Duration(50 * time.Millisecond))
		}
	}()
}

// GetGui returns the GoCUI GUI structure
func (l *Layout) GetGui() *gocui.Gui {
	return l.gui
}

// GetStatusView returns the top status bar view structure
func (l *Layout) GetStatusView() *View {
	return l.statusView
}

// GetLogsView returns the logs view structure
func (l *Layout) GetLogsView() *View {
	return l.logsView
}

// GetForwardsView returns the forward view structure
func (l *Layout) GetForwardsView() *View {
	return l.forwardsView
}

// GetProxyView returns the proxy view structure
func (l *Layout) GetProxyView() *View {
	return l.proxyView
}

func (l *Layout) setView(name, title string, xx, xy, yx, yy int) (*View, error) {
	view, err := l.gui.SetView(name, xx, xy, yx, yy)
	if err != nil && err != gocui.ErrUnknownView {
		return nil, err
	}

	view.Title = title
	view.Autoscroll = true
	view.Wrap = true
	view.SelFgColor = gocui.ColorGreen

	return NewView(name, title, view), nil
}

func (l *Layout) setStatusView(name string, xx, xy, yx, yy int) (*View, error) {
	view, err := l.gui.SetView(name, xx, xy, yx, yy)
	if err != nil && err != gocui.ErrUnknownView {
		return nil, err
	}

	view.FgColor = gocui.ColorGreen
	view.Frame = false

	return NewView(name, "", view), nil
}

func (l *Layout) setKeyBindings() {
	// Scroll up
	if err := l.gui.SetKeybinding("", gocui.KeyArrowUp, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		view := l.highlighted.GetView()

		if l.gui.CurrentView() == l.fullscreenView.GetView() {
			view = l.fullscreenView.GetView()
		}

		ox, oy := view.Origin()
		view.Autoscroll = false
		view.SetOrigin(ox, oy-6)

		return nil
	}); err != nil {
		panic(err)
	}

	// Scroll down
	if err := l.gui.SetKeybinding("", gocui.KeyArrowDown, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		view := l.highlighted.GetView()

		if l.gui.CurrentView() == l.fullscreenView.GetView() {
			view = l.fullscreenView.GetView()
		}

		ox, oy := view.Origin()

		view.Autoscroll = false
		view.SetOrigin(ox, oy+6)

		return nil
	}); err != nil {
		panic(err)
	}

	// Focus pane (highlight) - arrow left
	if err := l.gui.SetKeybinding("", gocui.KeyArrowLeft, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		var reversedViewsOrder = map[string]*View{}

		for name, view := range l.viewsOrder {
			view.GetView().Highlight = false
			view.GetView().Title = view.GetTitle()

			for _, loopView := range l.viewsOrder {
				if loopView.GetName() == name {
					reversedViewsOrder[view.GetName()] = loopView
					break
				}
			}
		}

		nextView := reversedViewsOrder[l.highlighted.GetName()]
		l.highlighted = nextView

		l.highlighted.GetView().Highlight = true
		l.highlighted.GetView().Title = fmt.Sprintf(" %s (Current)", l.highlighted.GetTitle())

		return nil
	}); err != nil {
		panic(err)
	}

	// Focus pane (highlight) - arrow right
	if err := l.gui.SetKeybinding("", gocui.KeyArrowRight, gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		for _, view := range l.viewsOrder {
			view.GetView().Highlight = false
			view.GetView().Title = view.GetTitle()
		}

		nextView := l.viewsOrder[l.highlighted.GetName()]
		l.highlighted = nextView

		l.highlighted.GetView().Highlight = true
		l.highlighted.GetView().Title = fmt.Sprintf(" %s (Current)", l.highlighted.GetTitle())

		return nil
	}); err != nil {
		panic(err)
	}

	// Enable autoscroll on highlighted view
	if err := l.gui.SetKeybinding("", 'a', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		view := l.highlighted.GetView()
		if l.gui.CurrentView() == l.fullscreenView.GetView() {
			view = l.fullscreenView.GetView()
		}

		view.Autoscroll = !view.Autoscroll

		return nil
	}); err != nil {
		panic(err)
	}

	// Enable autoscroll on highlighted view
	if err := l.gui.SetKeybinding("", 'f', gocui.ModNone, func(g *gocui.Gui, v *gocui.View) error {
		l.fullscreenView.GetView().Clear()

		if l.gui.CurrentView() == l.fullscreenView.GetView() {
			l.gui.SetViewOnBottom("fullscreen")
			l.gui.SetCurrentView(l.highlighted.GetName())
		} else {
			l.fullscreenView.GetView().Title = l.highlighted.GetTitle()

			l.gui.SetViewOnTop("fullscreen")
			l.gui.SetCurrentView("fullscreen")

			for _, line := range l.highlighted.GetView().BufferLines() {
				l.fullscreenView.Writef("%s\n", line)
			}
		}

		return nil
	}); err != nil {
		panic(err)
	}
}
