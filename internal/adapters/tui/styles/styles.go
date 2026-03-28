package styles

import "github.com/charmbracelet/lipgloss"

// Color palette.
var (
	primary   = lipgloss.Color("#7C3AED") // purple
	secondary = lipgloss.Color("#06B6D4") // cyan
	success   = lipgloss.Color("#22C55E") // green
	warning   = lipgloss.Color("#EAB308") // yellow
	danger    = lipgloss.Color("#EF4444") // red
	muted     = lipgloss.Color("#6B7280") // gray
	bg        = lipgloss.Color("#1E1E2E") // dark background
	fg        = lipgloss.Color("#CDD6F4") // light foreground
)

// TabBar returns the style for the tab bar container.
func TabBar() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderBottom(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(muted).
		PaddingBottom(0)
}

// ActiveTab returns the style for the currently selected tab.
func ActiveTab() lipgloss.Style {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(primary).
		BorderBottom(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(primary).
		Padding(0, 2)
}

// InactiveTab returns the style for unselected tabs.
func InactiveTab() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(muted).
		Padding(0, 2)
}

// StatusBar returns the style for the bottom status bar.
func StatusBar() lipgloss.Style {
	return lipgloss.NewStyle().
		Foreground(fg).
		Background(lipgloss.Color("#313244")).
		Padding(0, 1)
}

// StatusCode returns a style colored by HTTP status code range.
func StatusCode(code int) lipgloss.Style {
	var color lipgloss.Color
	switch {
	case code >= 200 && code < 300:
		color = success
	case code >= 300 && code < 400:
		color = secondary
	case code >= 400 && code < 500:
		color = warning
	case code >= 500:
		color = danger
	default:
		color = muted
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

// MethodColor returns a style colored by HTTP method.
func MethodColor(method string) lipgloss.Style {
	var color lipgloss.Color
	switch method {
	case "GET":
		color = success
	case "POST":
		color = secondary
	case "PUT":
		color = warning
	case "PATCH":
		color = lipgloss.Color("#F97316") // orange
	case "DELETE":
		color = danger
	default:
		color = muted
	}
	return lipgloss.NewStyle().Bold(true).Foreground(color)
}

// Label returns a style for form labels.
func Label() lipgloss.Style {
	return lipgloss.NewStyle().Bold(true).Foreground(fg)
}

// Focused returns a style for focused input fields.
func Focused() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.ThickBorder()).
		BorderForeground(primary).
		PaddingLeft(1)
}

// Blurred returns a style for unfocused input fields.
func Blurred() lipgloss.Style {
	return lipgloss.NewStyle().
		BorderLeft(true).
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(muted).
		PaddingLeft(1)
}
