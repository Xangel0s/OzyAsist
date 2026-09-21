package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	listeningWaveFrames = []string{
		"  ▂ ▃ ▅ ▃ ▂    ▂ ▃ ▅ ▃ ▂  ",
		" ▂ ▃ ▅ ▆ ▅ ▃  ▂ ▃ ▅ ▆ ▅ ▃ ",
		"▃ ▅ ▆ ▇ ▆ ▅ ▃ ▅ ▆ ▇ ▆ ▅ ▃ ",
		" ▂ ▃ ▅ ▆ ▅ ▃  ▂ ▃ ▅ ▆ ▅ ▃ ",
	}

	speakingWaveFrames = []string{
		"█ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █",
		"▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇",
		"▃ ▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃ ▂ ▃ ▅",
		"▂ ▃ ▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃ ▂ ▃",
		"▃ ▂ ▃ ▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃ ▂",
		"▅ ▃ ▂ ▃ ▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅ ▃",
		"▇ ▅ ▃ ▂ ▃ ▅ █ ▇ ▅ ▃ ▂ ▃ ▅ ▇ █ ▇ ▅",
	}

	VoiceHeaderStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#181e00")).
		Background(lipgloss.Color("#d1f107")).
		Bold(true).
		Padding(0, 1)

	VoiceLimeText = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#d1f107")).
		Bold(true)

	VoiceDimText = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888"))

	VoiceCardBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#333333")).
		Padding(1, 1)

	VoiceActiveCardBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#d1f107")).
		Padding(1, 1)
)

// renderVoiceCard renders the voice interaction panel to be displayed in the sidebar
func renderVoiceCard(state VoiceConsoleState, waveFrame int, activeUtterance string, width int) string {
	var statusLabel string
	var waveText string
	var subtitleText string
	cardStyle := VoiceCardBoxStyle.Width(width - 2)

	switch state {
	case VoiceStateSpeaking:
		cardStyle = VoiceActiveCardBoxStyle.Width(width - 2)
		statusLabel = VoiceLimeText.Render("[OZY HABLANDO]")
		idx := waveFrame % len(speakingWaveFrames)
		waveText = VoiceLimeText.Render(speakingWaveFrames[idx])
		if activeUtterance != "" {
			wrapped := wrapLine(activeUtterance, width-4)
			subtitleText = lipgloss.NewStyle().Italic(true).Render("\"" + strings.Join(wrapped, "\n") + "\"")
			subtitleText = fmt.Sprintf("\n%s\n", subtitleText)
		}

	case VoiceStateThinking:
		statusLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#f1c40f")).Bold(true).Render("[PROCESANDO VOZ]")
		spinnerFrames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		idx := waveFrame % len(spinnerFrames)
		waveText = fmt.Sprintf("%s Analizando acciones...", spinnerFrames[idx])

	case VoiceStateListening:
		statusLabel = VoiceDimText.Render("[ESCUCHANDO]")
		idx := waveFrame % len(listeningWaveFrames)
		waveText = VoiceDimText.Render(listeningWaveFrames[idx])
		subtitleText = "\n'Hey Ozy'\n"
	
	default: // Idle
		statusLabel = VoiceDimText.Render("[VOZ ACTIVA]")
		waveText = VoiceDimText.Render(listeningWaveFrames[0])
		subtitleText = "\n'Hey Ozy'\n"
	}

	cardContent := fmt.Sprintf("%s\n\n %s\n%s", statusLabel, waveText, subtitleText)
	return cardStyle.Render(cardContent)
}
