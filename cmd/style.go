package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

var (
	styleAccent = lipgloss.AdaptiveColor{Light: "#5A3FD6", Dark: "#9D86F9"}
	styleMuted  = lipgloss.AdaptiveColor{Light: "#8A87A0", Dark: "#6E6A86"}

	bannerBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(styleAccent).
				Padding(0, 1)

	bannerTaglineStyle = lipgloss.NewStyle().
				Foreground(styleMuted).
				Padding(0, 1)

	projectNameStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(styleAccent)

	projectDetailStyle = lipgloss.NewStyle().
				Foreground(styleMuted)
)

// printBanner displays the Monday header banner on CLI startup
func printBanner() {
	badge := bannerBadgeStyle.Render("⚡ Monday")
	tagline := bannerTaglineStyle.Render("your local development companion" + versionSuffix())

	fmt.Println(lipgloss.JoinHorizontal(lipgloss.Center, badge, tagline))
	fmt.Println()
}

func versionSuffix() string {
	if Version == "" {
		return ""
	}

	return " · " + Version
}
