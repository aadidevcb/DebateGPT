package tui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	primaryColor   = lipgloss.Color("#7C3AED") // purple
	secondaryColor = lipgloss.Color("#10B981") // green
	accentColor    = lipgloss.Color("#3B82F6") // blue
	warningColor   = lipgloss.Color("#F59E0B") // amber
	errorColor     = lipgloss.Color("#EF4444") // red
	mutedColor     = lipgloss.Color("#6B7280") // gray
	bgColor        = lipgloss.Color("#1F2937") // dark bg

	// Agent colors
	AgentColors = map[int]lipgloss.Color{
		0: lipgloss.Color("#A855F7"), // purple
		1: lipgloss.Color("#22C55E"), // green
		2: lipgloss.Color("#3B82F6"), // blue
		3: lipgloss.Color("#F97316"), // orange
		4: lipgloss.Color("#EC4899"), // pink
	}

	// Agent icons
	AgentIcons = map[int]string{
		0: "🟣",
		1: "🟢",
		2: "🔵",
		3: "🟠",
		4: "🟡",
	}

	// Styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(primaryColor).
			Padding(0, 1)

	RoundHeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			Border(lipgloss.NormalBorder(), false, false, true, false).
			BorderForeground(mutedColor).
			MarginTop(1).
			MarginBottom(1)

	AgentNameStyle = lipgloss.NewStyle().
			Bold(true)

	StreamingStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Italic(true)

	CostStyle = lipgloss.NewStyle().
			Foreground(warningColor).
			Bold(true)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			Border(lipgloss.NormalBorder(), true, false, false, false).
			BorderForeground(mutedColor).
			MarginTop(1).
			Padding(0, 1)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(errorColor).
			Bold(true)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Bold(true)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(mutedColor).
			Padding(0, 1)
)

// GetAgentColor returns the color for an agent by index.
func GetAgentColor(index int) lipgloss.Color {
	if c, ok := AgentColors[index]; ok {
		return c
	}
	return lipgloss.Color("#FFFFFF")
}

// GetAgentIcon returns the icon for an agent by index.
func GetAgentIcon(index int) string {
	if icon, ok := AgentIcons[index]; ok {
		return icon
	}
	return "⚪"
}
