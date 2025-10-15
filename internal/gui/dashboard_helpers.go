package gui

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

// formatRelativeTime formats a time relative to now
func formatRelativeTime(timeStr string) string {
	if timeStr == "" {
		return "Never"
	}

	t, err := time.Parse("2006-01-02 15:04:05", timeStr)
	if err != nil {
		return timeStr
	}

	if t.IsZero() {
		return "Never"
	}

	duration := time.Since(t)

	if duration < time.Minute {
		return "Just now"
	} else if duration < time.Hour {
		mins := int(duration.Minutes())
		return fmt.Sprintf("%d min%s ago", mins, pluralize(mins))
	} else if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%d hour%s ago", hours, pluralize(hours))
	} else {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%d day%s ago", days, pluralize(days))
	}
}

// formatAbsoluteTime formats a time string
func formatAbsoluteTime(timeStr string) string {
	if timeStr == "" {
		return "Never"
	}
	return timeStr
}

// formatPercentage formats a percentage value
func formatPercentage(p float64) string {
	return fmt.Sprintf("%.1f%%", p)
}

// formatHealth returns a colored health indicator
func formatHealth(p float64) string {
	if p >= 95.0 {
		return fmt.Sprintf("%.1f%% ●", p)
	} else if p >= 80.0 {
		return fmt.Sprintf("%.1f%% ●", p)
	} else {
		return fmt.Sprintf("%.1f%% ●", p)
	}
}

// getHealthColor returns a color based on health percentage
func getHealthColor(p float64) color.Color {
	if p >= 95.0 {
		return color.RGBA{R: 46, G: 204, B: 113, A: 255} // Green
	} else if p >= 80.0 {
		return color.RGBA{R: 241, G: 196, B: 15, A: 255} // Yellow
	} else {
		return color.RGBA{R: 231, G: 76, B: 60, A: 255} // Red
	}
}

// getStatusColor returns a color based on status string
func getStatusColor(status string) color.Color {
	switch status {
	case "Up":
		return color.RGBA{R: 46, G: 204, B: 113, A: 255} // Green
	case "Down":
		return color.RGBA{R: 231, G: 76, B: 60, A: 255} // Red
	default:
		return color.RGBA{R: 149, G: 165, B: 166, A: 255} // Gray
	}
}

// pluralize returns "s" if count != 1
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}

// formatDuration formats a duration string to be more readable
func formatDuration(durStr string) string {
	if durStr == "" || durStr == "0s" {
		return "None"
	}
	return durStr
}

// createSection creates a titled section with content
func createSection(title string, rows ...*fyne.Container) *fyne.Container {
	titleLabel := widget.NewLabelWithStyle(title, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	
	content := []fyne.CanvasObject{
		titleLabel,
		widget.NewSeparator(),
	}
	
	for _, row := range rows {
		content = append(content, row)
	}
	
	content = append(content, widget.NewLabel("")) // Spacing
	
	return container.NewVBox(content...)
}

// createStatRow creates a label-value row with proper spacing
func createStatRow(label, value string) *fyne.Container {
	labelWidget := widget.NewLabel(label)
	labelWidget.TextStyle.Bold = true
	
	valueWidget := widget.NewLabel(value)
	
	// Use a grid layout for proper alignment
	return container.NewBorder(
		nil, nil,
		labelWidget,
		valueWidget,
		nil,
	)
}

// createColoredStatRow creates a label-value row with colored value
func createColoredStatRow(label, value string, valueColor color.Color) *fyne.Container {
	labelWidget := widget.NewLabel(label)
	labelWidget.TextStyle.Bold = true
	
	valueText := canvas.NewText(value, valueColor)
	valueText.TextSize = 14
	
	return container.NewBorder(
		nil, nil,
		labelWidget,
		valueText,
		nil,
	)
}

// createStatusIndicator creates a colored status indicator
func createStatusIndicator(status string) *fyne.Container {
	statusColor := getStatusColor(status)
	
	dot := canvas.NewCircle(statusColor)
	dot.Resize(fyne.NewSize(10, 10))
	
	statusLabel := widget.NewLabel(status)
	statusLabel.TextStyle.Bold = true
	
	return container.NewHBox(
		container.NewMax(
			container.NewPadded(dot),
		),
		statusLabel,
	)
}

// createHealthIndicator creates a colored health percentage indicator
func createHealthIndicator(healthPercent float64) *fyne.Container {
	healthColor := getHealthColor(healthPercent)
	
	dot := canvas.NewCircle(healthColor)
	dot.Resize(fyne.NewSize(10, 10))
	
	healthLabel := widget.NewLabel(formatPercentage(healthPercent))
	healthLabel.TextStyle.Bold = true
	
	return container.NewHBox(
		container.NewMax(
			container.NewPadded(dot),
		),
		healthLabel,
	)
}

// createEmptyState creates an empty state message
func createEmptyState(message string) *fyne.Container {
	label := widget.NewLabel(message)
	label.Alignment = fyne.TextAlignCenter
	label.TextStyle.Italic = true
	
	return container.NewCenter(
		container.NewVBox(
			widget.NewLabel(""),
			widget.NewLabel(""),
			label,
		),
	)
}

