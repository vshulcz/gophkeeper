package tui

import "github.com/charmbracelet/lipgloss"

type uiStyles struct {
	Header       lipgloss.Style
	Subtle       lipgloss.Style
	Accent       lipgloss.Style
	Status       lipgloss.Style
	Error        lipgloss.Style
	Panel        lipgloss.Style
	PanelWide    lipgloss.Style
	Item         lipgloss.Style
	ItemDesc     lipgloss.Style
	Selected     lipgloss.Style
	SelectedDesc lipgloss.Style
}

func newStyles() uiStyles {
	return uiStyles{
		Header: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33")),
		Subtle: lipgloss.NewStyle().Foreground(lipgloss.Color("244")),
		Accent: lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true),
		Status: lipgloss.NewStyle().Foreground(lipgloss.Color("35")),
		Error:  lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Bold(true),
		Panel: lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")),
		PanelWide: lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("238")),
		Item:         lipgloss.NewStyle().Foreground(lipgloss.Color("251")),
		ItemDesc:     lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		Selected:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39")).Background(lipgloss.Color("236")),
		SelectedDesc: lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Background(lipgloss.Color("236")),
	}
}

var styles = newStyles()
