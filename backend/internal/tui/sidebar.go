package tui

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/ozyassist/backend/internal/db"
	"github.com/ozyassist/backend/internal/memory"
	"github.com/ozyassist/backend/internal/system"
)

func (m Model) estimateTokens() int {
	totalChars := 0
	for _, e := range m.entries {
		totalChars += len(e.Content) + len(e.Thinking) + len(e.ToolInput)
	}
	tokens := int(float64(totalChars) / 3.8)
	if tokens < 10 && len(m.entries) > 0 {
		tokens = 50
	}
	return tokens
}

func (m Model) getModelContextLimit() int {
	modelName := ""
	if m.chat != nil && m.chat.Model != "" {
		modelName = strings.ToLower(m.chat.Model)
	} else if m.provider != nil && len(m.provider.Models()) > 0 {
		modelName = strings.ToLower(m.provider.Models()[0])
	}

	if strings.Contains(modelName, "claude") {
		return 200000
	}
	if strings.Contains(modelName, "deepseek") {
		return 64000
	}
	return 128000
}

func (m Model) renderSidebar(width, height int) string {
	if width < 22 {
		width = 22
	}
	innerWidth := width - 4
	if innerWidth < 18 {
		innerWidth = 18
	}

	var sb strings.Builder

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	dimStyle := lipgloss.NewStyle().Foreground(ColorMuted)
	boxStyle := lipgloss.NewStyle().
		Background(ColorMessageBg).
		Border(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("#2a2a2a")).
		Padding(0, 1).
		Width(innerWidth)

	// Cabecera del Panel
	sb.WriteString(" " + titleStyle.Render("◈ CONTEXTO & TELEMETRÍA") + "\n")
	sb.WriteString(" " + dimStyle.Render("[Ctrl+B] Colapsar") + "\n\n")

	// 0. Voice Panel (if enabled)
	if m.voiceEnabled {
		voiceView := renderVoiceCard(m.voiceState, m.voiceWaveFrame, m.activeUtterance, innerWidth)
		if voiceView != "" {
			sb.WriteString(voiceView + "\n\n")
		}
	}

	// 1. Telemetría de Tokens y Context Window
	usedTokens := m.estimateTokens()
	maxTokens := m.getModelContextLimit()
	pct := float64(usedTokens) / float64(maxTokens) * 100.0
	if pct > 100 {
		pct = 100
	}

	barWidth := innerWidth - 8
	if barWidth > 12 {
		barWidth = 12
	}
	if barWidth < 4 {
		barWidth = 4
	}
	filledBars := int((pct / 100.0) * float64(barWidth))
	if filledBars == 0 && usedTokens > 0 {
		filledBars = 1
	}
	emptyBars := barWidth - filledBars
	if emptyBars < 0 {
		emptyBars = 0
	}
	progressBar := titleStyle.Render(strings.Repeat("█", filledBars)) + dimStyle.Render(strings.Repeat("░", emptyBars))

	var tokensSb strings.Builder
	tokensSb.WriteString(titleStyle.Render("◈ TOKENS & CONTEXTO") + "\n")
	tokensSb.WriteString(fmt.Sprintf("%s %s\n", progressBar, dimStyle.Render(fmt.Sprintf("%.1f%%", pct))))
	tokensSb.WriteString(labelStyle.Render(fmt.Sprintf("~%s / %sk", formatNumber(usedTokens), formatNumber(maxTokens/1000))) + "\n")
	tokensSb.WriteString(dimStyle.Render(fmt.Sprintf("Turnos: %d mensajes", len(m.entries))))
	sb.WriteString(boxStyle.Render(tokensSb.String()) + "\n\n")

	// 2. Memoria RAM y Caché Zero-Docker
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	allocMB := float64(ms.Alloc) / 1024 / 1024
	sysMB := float64(ms.Sys) / 1024 / 1024

	projectsCount := len(system.DefaultPathRegistry().ListProjects())
	pathsCount := len(system.DefaultPathRegistry().ListAll())

	var ramSb strings.Builder
	ramSb.WriteString(titleStyle.Render("◈ MEMORIA RAM (ZERO-DOCKER)") + "\n")
	ramSb.WriteString(labelStyle.Render(fmt.Sprintf("RAM en uso: %.1f MB", allocMB)) + "\n")
	ramSb.WriteString(dimStyle.Render(fmt.Sprintf("RAM sistema: %.1f MB", sysMB)) + "\n")
	ramSb.WriteString(dimStyle.Render(fmt.Sprintf("Proyectos en RAM: %d", projectsCount)) + "\n")
	ramSb.WriteString(dimStyle.Render(fmt.Sprintf("Rutas indexadas: %d", pathsCount)))
	sb.WriteString(boxStyle.Render(ramSb.String()) + "\n\n")

	// 3. Recuerdos Activos en SQLite
	var memFacts []memory.FactMemory
	if store := memory.DefaultStore(); store != nil && db.DB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		memFacts, _ = store.GetRecentFacts(ctx, db.DefaultUserID(), 3)
		cancel()
	}

	var memSb strings.Builder
	memSb.WriteString(titleStyle.Render("◈ MEMORIA CONTINUA") + "\n")
	if len(memFacts) == 0 {
		memSb.WriteString(dimStyle.Render("(Sin recuerdos registrados)\nUsa /memories para gestionar"))
	} else {
		for i, f := range memFacts {
			cat := strings.ToUpper(string(f.Category))
			content := f.Content
			if len(content) > innerWidth-8 {
				content = content[:innerWidth-11] + "..."
			}
			memSb.WriteString(fmt.Sprintf("• %s: %s\n", titleStyle.Render(cat), dimStyle.Render(content)))
			if i >= 1 {
				break
			}
		}
	}
	sb.WriteString(boxStyle.Render(memSb.String()) + "\n\n")

	// 4. Modelo Activo
	provName := "Local"
	if m.provider != nil {
		provName = m.provider.Name()
	}
	modelName := "auto"
	if m.chat != nil && m.chat.Model != "" {
		modelName = m.chat.Model
	}

	var modelSb strings.Builder
	modelSb.WriteString(titleStyle.Render("◈ MODELO & SESIÓN") + "\n")
	modelSb.WriteString(labelStyle.Render(fmt.Sprintf("%s:%s", provName, modelName)) + "\n")
	modelSb.WriteString(dimStyle.Render(fmt.Sprintf("Permisos: %s", m.permissionLevel)))
	sb.WriteString(boxStyle.Render(modelSb.String()))

	rendered := sb.String()
	return lipgloss.NewStyle().Width(width).Height(height).Render(rendered)
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	return fmt.Sprintf("%d,%03d", n/1000, n%1000)
}
