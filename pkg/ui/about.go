package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	// aboutLogo is the monday ASCII-art logo displayed in the about overlay
	aboutLogo = []string{
		"                          _             ",
		" _ __ ___   ___  _ __   __| | __ _ _   _ ",
		"| '_ ` _ \\ / _ \\| '_ \\ / _` |/ _` | | | |",
		"| | | | | | (_) | | | | (_| | (_| | |_| |",
		"|_| |_| |_|\\___/|_| |_|\\__,_|\\__,_|\\__, |",
		"                                   |___/ ",
	}

	// aboutLogoGradient holds the violet shades applied to the logo, one per line
	aboutLogoGradient = []lipgloss.Color{
		"#6C4EE6",
		"#7B5BEC",
		"#8A6BF1",
		"#9879F5",
		"#A78AF8",
		"#B69BFB",
	}

	aboutBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 5)

	aboutTaglineStyle = lipgloss.NewStyle().
				Italic(true).
				Foreground(colorMuted)

	aboutAuthorStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorTitle)

	aboutLinkStyle = lipgloss.NewStyle().
			Underline(true).
			Foreground(colorAccent)

	aboutHintStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
)

// aboutView renders the about overlay: the monday logo, its creator and the
// project repository, centered on the screen
func (m *model) aboutView() string {
	logo := make([]string, 0, len(aboutLogo))
	for i, line := range aboutLogo {
		logo = append(logo, lipgloss.NewStyle().Foreground(aboutLogoGradient[i]).Render(line))
	}

	version := "⚡ your local development companion"
	if m.version != "" {
		version += " · " + m.version
	}

	sections := []string{
		strings.Join(logo, "\n"),
		"",
		aboutTaglineStyle.Render(version),
		"",
		statusSeparatorStyle.Render(strings.Repeat("─", lipgloss.Width(aboutLogo[1]))),
		"",
		aboutAuthorStyle.Render("Created by Vincent Composieux ") + aboutHintStyle.Render("<monday@composieux.fr>"),
		aboutLinkStyle.Render("https://github.com/eko/monday"),
		"",
		aboutHintStyle.Render("press any key to close"),
	}

	box := aboutBoxStyle.Render(
		lipgloss.JoinVertical(lipgloss.Center, sections...),
	)

	height := m.height - m.headerHeight() - m.footerHeight()
	if height < 1 {
		height = 1
	}

	return lipgloss.Place(m.width, height, lipgloss.Center, lipgloss.Center, box)
}
