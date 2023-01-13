package models

import "github.com/charmbracelet/lipgloss"

const maxFormWidth = 30

var (
	itemStyle = lipgloss.NewStyle().
			MarginTop(1).
			AlignHorizontal(lipgloss.Center).
			AlignVertical(lipgloss.Center)

	infoMessageStyle = itemStyle.Copy()

	errorMessageStyle = itemStyle.Copy().
				Foreground(lipgloss.Color("7")).
				Background(lipgloss.Color("9"))

	blurredButtonStyle = itemStyle.Copy().
				Foreground(lipgloss.Color("7")).
				Background(lipgloss.Color("8"))

	focusedButtonStyle = itemStyle.Copy().
				Foreground(lipgloss.Color("7")).
				Background(lipgloss.Color("13")).
				Underline(true)
)

func buttonView(text string, width int, focused bool) string {
	var st lipgloss.Style
	if focused {
		st = focusedButtonStyle.Copy()
	} else {
		st = blurredButtonStyle.Copy()
	}

	return st.Width(width).Height(3).Render(text)
}

func infoView(text string, width, height int, isError bool) string {
	var st lipgloss.Style
	if isError {
		st = errorMessageStyle.Copy()
	} else {
		st = infoMessageStyle.Copy()
	}

	return st.Width(width).Height(height).Render(text)
}
