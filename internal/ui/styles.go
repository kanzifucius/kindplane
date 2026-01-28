package ui

import (
	"github.com/charmbracelet/lipgloss"
)

// Theme colors - using adaptive colors for light/dark terminal support
var (
	// Primary colors
	ColorPrimary   = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"} // Purple
	ColorSecondary = lipgloss.AdaptiveColor{Light: "#0891B2", Dark: "#22D3EE"} // Cyan

	// Status colors
	ColorSuccess = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"} // Green
	ColorError   = lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"} // Red
	ColorWarning = lipgloss.AdaptiveColor{Light: "#D97706", Dark: "#FBBF24"} // Yellow/Amber
	ColorInfo    = lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#60A5FA"} // Blue

	// Neutral colors
	ColorMuted  = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"} // Gray
	ColorSubtle = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#6B7280"} // Lighter gray
	ColorText   = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#F9FAFB"} // Main text
	ColorDim    = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#374151"} // Dimmed
	ColorBorder = lipgloss.AdaptiveColor{Light: "#E5E7EB", Dark: "#374151"} // Border

	// Accent colors for different components
	ColorCrossplane = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"} // Purple (Crossplane brand)
	ColorKubernetes = lipgloss.AdaptiveColor{Light: "#326CE5", Dark: "#60A5FA"} // Blue (K8s brand)
	ColorHelm       = lipgloss.AdaptiveColor{Light: "#0F1689", Dark: "#818CF8"} // Navy/Indigo (Helm brand)
)

// Icon definitions for consistent visual language
var (
	IconSuccess   = "✓"
	IconError     = "✗"
	IconWarning   = "!"
	IconInfo      = "•"
	IconPending   = "○"
	IconRunning   = "◉"
	IconArrow     = "→"
	IconBullet    = "▸"
	IconCheck     = "✔"
	IconCross     = "✖"
	IconStar      = "★"
	IconDot       = "·"
	IconPipe      = "│"
	IconCorner    = "└"
	IconTee       = "├"
	IconDash      = "─"
	IconBox       = "■"
	IconBoxEmpty  = "□"
	IconSpinner   = "◐"
	IconCluster   = "⎈"
	IconPackage   = "📦"
	IconGear      = "⚙"
	IconCloud     = "☁"
	IconLock      = "🔒"
	IconUnlock    = "🔓"
	IconFolder    = "📁"
	IconFile      = "📄"
	IconRocket    = "🚀"
	IconWrench    = "🔧"
	IconMagnifier = "🔍"
)

// Base styles
var (
	// Text styles
	StyleBold = lipgloss.NewStyle().Bold(true)
	StyleDim  = lipgloss.NewStyle().Faint(true)
	StyleCode = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#1F2937"}).
			Foreground(lipgloss.AdaptiveColor{Light: "#DC2626", Dark: "#F87171"}).
			Padding(0, 1)

	// Status message styles
	StyleSuccess = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	StyleError = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	StyleWarning = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)

	StyleInfo = lipgloss.NewStyle().
			Foreground(ColorInfo)

	StyleMuted = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Header styles
	StyleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			MarginBottom(1)

	StyleSubtitle = lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Italic(true)

	StyleSectionHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorText).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(ColorBorder).
				MarginTop(1).
				MarginBottom(1)

	// Box styles for panels and containers
	StyleBox = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	StyleBoxSuccess = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSuccess).
			Padding(1, 2)

	StyleBoxError = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorError).
			Padding(1, 2)

	StyleBoxWarning = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorWarning).
			Padding(1, 2)

	StyleBoxInfo = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorInfo).
			Padding(1, 2)

	// Diagnostic styles
	StyleDiagnosticHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorError).
				Background(lipgloss.AdaptiveColor{Light: "#FEE2E2", Dark: "#7F1D1D"}).
				Padding(0, 2).
				MarginTop(1)

	StyleDiagnosticBox = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorError).
				Padding(1, 2).
				MarginBottom(1)

	// List styles
	StyleListItem = lipgloss.NewStyle().
			PaddingLeft(2)

	StyleListItemSelected = lipgloss.NewStyle().
				PaddingLeft(2).
				Foreground(ColorPrimary).
				Bold(true)

	// Table styles
	StyleTableHeader = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorSecondary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(ColorBorder)

	StyleTableRow = lipgloss.NewStyle().
			Foreground(ColorText)

	StyleTableRowAlt = lipgloss.NewStyle().
				Foreground(ColorText).
				Background(lipgloss.AdaptiveColor{Light: "#F9FAFB", Dark: "#111827"})

	// Step/Progress styles
	StyleStepPending = lipgloss.NewStyle().
				Foreground(ColorMuted)

	StyleStepActive = lipgloss.NewStyle().
			Foreground(ColorInfo).
			Bold(true)

	StyleStepComplete = lipgloss.NewStyle().
				Foreground(ColorSuccess)

	StyleStepFailed = lipgloss.NewStyle().
			Foreground(ColorError)

	// Log/Output styles
	StyleLogLine = lipgloss.NewStyle().
			Foreground(ColorMuted).
			PaddingLeft(4)

	StyleLogTimestamp = lipgloss.NewStyle().
				Foreground(ColorSubtle)

	StyleLogLevel = lipgloss.NewStyle().
			Bold(true)

	// Badge styles
	StyleBadge = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true)

	StyleBadgeSuccess = StyleBadge.
				Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}).
				Background(ColorSuccess)

	StyleBadgeError = StyleBadge.
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}).
			Background(ColorError)

	StyleBadgeWarning = StyleBadge.
				Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#000000"}).
				Background(ColorWarning)

	StyleBadgeInfo = StyleBadge.
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}).
			Background(ColorInfo)

	StyleBadgeMuted = StyleBadge.
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#000000"}).
			Background(ColorMuted)

	// Indent helper
	StyleIndent1 = lipgloss.NewStyle().PaddingLeft(2)
	StyleIndent2 = lipgloss.NewStyle().PaddingLeft(4)
	StyleIndent3 = lipgloss.NewStyle().PaddingLeft(6)
	StyleIndent4 = lipgloss.NewStyle().PaddingLeft(8)
)

// StatusStyle returns the appropriate style for a given status
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case "success", "ready", "healthy", "running", "deployed":
		return StyleSuccess
	case "error", "failed", "unhealthy":
		return StyleError
	case "warning", "degraded":
		return StyleWarning
	case "pending", "waiting", "installing":
		return StyleInfo
	default:
		return StyleMuted
	}
}

// StatusIcon returns the appropriate icon for a given status
func StatusIcon(status string) string {
	switch status {
	case "success", "ready", "healthy", "deployed":
		return IconSuccess
	case "error", "failed", "unhealthy":
		return IconError
	case "warning", "degraded":
		return IconWarning
	case "pending", "waiting":
		return IconPending
	case "running", "installing":
		return IconRunning
	case "info":
		return IconInfo
	default:
		return IconDot
	}
}

// StatusBadge returns a styled badge for a status
func StatusBadge(status string) string {
	var style lipgloss.Style
	switch status {
	case "success", "ready", "healthy", "running", "deployed":
		style = StyleBadgeSuccess
	case "error", "failed", "unhealthy":
		style = StyleBadgeError
	case "warning", "degraded":
		style = StyleBadgeWarning
	case "pending", "waiting", "installing":
		style = StyleBadgeInfo
	default:
		style = StyleBadgeMuted
	}
	return style.Render(status)
}

// -----------------------------------------------------------------------------
// Dashboard Styles (TUI Bootstrap Dashboard)
// -----------------------------------------------------------------------------

// Dashboard layout constants
const (
	DashboardMinWidth  = 80
	DashboardMaxWidth  = 120
	DashboardLogBuffer = 15
)

var (
	// Dashboard box styles (sharp borders per user preference)
	StyleDashboardBox = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorBorder).
				Padding(0, 1)

	StyleDashboardBoxActive = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// Dashboard header box
	StyleDashboardHeader = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorPrimary).
				Padding(0, 1)

	// Dashboard phase table styles
	StyleDashboardPhaseHeader = lipgloss.NewStyle().
					Bold(true).
					Foreground(ColorSecondary)

	StyleDashboardPhaseRow = lipgloss.NewStyle().
				Foreground(ColorText)

	StyleDashboardPhaseRowActive = lipgloss.NewStyle().
					Foreground(ColorPrimary).
					Bold(true)

	StyleDashboardPhaseRowComplete = lipgloss.NewStyle().
					Foreground(ColorSuccess)

	StyleDashboardPhaseRowSkipped = lipgloss.NewStyle().
					Foreground(ColorMuted)

	StyleDashboardPhaseRowFailed = lipgloss.NewStyle().
					Foreground(ColorError)

	// Dashboard current operation panel
	StyleDashboardOperationBox = lipgloss.NewStyle().
					Border(lipgloss.NormalBorder()).
					BorderForeground(ColorSecondary).
					Padding(0, 1)

	// Dashboard log panel (verbose mode)
	StyleDashboardLogBox = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorMuted).
				Padding(0, 1)

	StyleDashboardLogLine = lipgloss.NewStyle().
				Foreground(ColorMuted)

	// Dashboard footer/status bar
	StyleDashboardFooter = lipgloss.NewStyle().
				Foreground(ColorMuted)

	StyleDashboardHotkey = lipgloss.NewStyle().
				Foreground(ColorSecondary).
				Bold(true)

	// Dashboard timeout styles
	StyleDashboardTimeoutOk = lipgloss.NewStyle().
				Foreground(ColorMuted)

	StyleDashboardTimeoutWarning = lipgloss.NewStyle().
					Foreground(ColorWarning).
					Bold(true)

	StyleDashboardTimeoutCritical = lipgloss.NewStyle().
					Foreground(ColorError).
					Bold(true)

	// Dashboard key-value display
	StyleDashboardLabel = lipgloss.NewStyle().
				Foreground(ColorMuted)

	StyleDashboardValue = lipgloss.NewStyle().
				Foreground(ColorText).
				Bold(true)

	// Progress bar colors for gradient effect
	ColorProgressStart  = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#A78BFA"} // Purple
	ColorProgressMiddle = lipgloss.AdaptiveColor{Light: "#2563EB", Dark: "#60A5FA"} // Blue
	ColorProgressEnd    = lipgloss.AdaptiveColor{Light: "#059669", Dark: "#34D399"} // Green
)

// DashboardWidth returns the dashboard width clamped to min/max bounds
func DashboardWidth(termWidth int) int {
	if termWidth < DashboardMinWidth {
		return DashboardMinWidth
	}
	if termWidth > DashboardMaxWidth {
		return DashboardMaxWidth
	}
	return termWidth
}

// PhaseStatusStyle returns the appropriate style for a phase status
func PhaseStatusStyle(status PhaseStatus) lipgloss.Style {
	switch status {
	case PhaseRunning:
		return StyleDashboardPhaseRowActive
	case PhaseComplete:
		return StyleDashboardPhaseRowComplete
	case PhaseSkipped:
		return StyleDashboardPhaseRowSkipped
	case PhaseFailed:
		return StyleDashboardPhaseRowFailed
	default:
		return StyleDashboardPhaseRow
	}
}
