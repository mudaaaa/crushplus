package styles

import (
	"charm.land/lipgloss/v2"
)

// NewMagmaTheme creates a new vibrant orange/red theme
func NewMagmaTheme() *Theme {
	t := &Theme{
		Name:   "magma",
		IsDark: true,

		// Vibrant Orange/Red Palette
		// Primary: Bright Orange
		Primary: lipgloss.Color("#FF5F1F"),
		// Secondary: Vermilion
		Secondary: lipgloss.Color("#E34234"),
		// Tertiary: Darker Red/Orange
		Tertiary: lipgloss.Color("#CC3300"),
		// Accent: Gold/Yellow for contrast
		Accent: lipgloss.Color("#FFD700"),

		// Backgrounds - Dark, warm grays
		BgBase:        lipgloss.Color("#1A1A1A"),
		BgBaseLighter: lipgloss.Color("#262626"),
		BgSubtle:      lipgloss.Color("#333333"),
		BgOverlay:     lipgloss.Color("#404040"),

		// Foregrounds
		FgBase:      lipgloss.Color("#E6E6E6"),
		FgMuted:     lipgloss.Color("#999999"),
		FgHalfMuted: lipgloss.Color("#666666"),
		FgSubtle:    lipgloss.Color("#4D4D4D"),
		FgSelected:  lipgloss.Color("#FFFFFF"),

		// Borders
		Border:      lipgloss.Color("#404040"),
		BorderFocus: lipgloss.Color("#FF5F1F"), // Primary

		// Status
		Success: lipgloss.Color("#32CD32"), // Lime Green
		Error:   lipgloss.Color("#FF0000"), // Red
		Warning: lipgloss.Color("#FFA500"), // Orange
		Info:    lipgloss.Color("#00BFFF"), // Deep Sky Blue

		// Colors for syntax highlighting etc.
		White: lipgloss.Color("#FFFFFF"),

		// Blues (Cool contrast)
		BlueLight: lipgloss.Color("#87CEFA"),
		BlueDark:  lipgloss.Color("#00008B"),
		Blue:      lipgloss.Color("#1E90FF"),

		// Yellows
		Yellow: lipgloss.Color("#FFD700"),
		Citron: lipgloss.Color("#DFFF00"),

		// Greens
		Green:      lipgloss.Color("#32CD32"),
		GreenDark:  lipgloss.Color("#006400"),
		GreenLight: lipgloss.Color("#90EE90"),

		// Reds
		Red:      lipgloss.Color("#FF0000"),
		RedDark:  lipgloss.Color("#8B0000"),
		RedLight: lipgloss.Color("#FF6347"),
		Cherry:   lipgloss.Color("#D2042D"),
	}

	// Text selection.
	t.TextSelection = lipgloss.NewStyle().Foreground(t.FgSelected).Background(t.Primary)

	// LSP and MCP status.
	t.ItemOfflineIcon = lipgloss.NewStyle().Foreground(t.FgMuted).SetString("●")
	t.ItemBusyIcon = t.ItemOfflineIcon.Foreground(t.Warning)
	t.ItemErrorIcon = t.ItemOfflineIcon.Foreground(t.Error)
	t.ItemOnlineIcon = t.ItemOfflineIcon.Foreground(t.Success)

	// Yolo Mode
	t.YoloIconFocused = lipgloss.NewStyle().Foreground(t.BgBase).Background(t.Warning).Bold(true).SetString(" ! ")
	t.YoloIconBlurred = t.YoloIconFocused.Foreground(t.BgBase).Background(t.FgMuted)
	t.YoloDotsFocused = lipgloss.NewStyle().Foreground(t.Accent).SetString(":::")
	t.YoloDotsBlurred = t.YoloDotsFocused.Foreground(t.FgMuted)

	return t
}
