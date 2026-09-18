package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ozyassist/backend/internal/memory"
)

func (m Model) renderMemoryManagerView() string {
	convWidth := m.width
	if convWidth <= 0 {
		convWidth = 80
	}

	var sb strings.Builder

	// 1. Título superior
	title := TopHeaderTitleStyle.Render("◈ GESTOR INTERACTIVO DE MEMORIAS Y RECUERDOS")
	sub := TopHeaderMetaStyle.Render("Memoria continua persistida en SQLite con FTS5 BM25")
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(title))
	sb.WriteString("\n")
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(sub))
	sb.WriteString("\n\n")

	// 2. Barra de búsqueda / filtro
	searchBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(0, 1).
		Render(fmt.Sprintf(" %s %s", TagActiveStyle.Render("BUSCAR"), m.memorySearchInput.View()))
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(searchBox))
	sb.WriteString("\n\n")

	// 3. Si se está añadiendo un nuevo recuerdo
	if m.memoryAddingNew {
		addBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Background(ColorMessageBg).
			Padding(1, 2).
			Render(fmt.Sprintf("%s\n%s\n\n%s",
				lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("NUEVO RECUERDO / PREFERENCIA:"),
				m.memoryNewInput.View(),
				lipgloss.NewStyle().Foreground(ColorMuted).Render("[Enter] Guardar y Persistir  •  [Esc] Cancelar"),
			))
		sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(addBox))
		sb.WriteString("\n\n")
	}

	// 4. Lista de recuerdos filtrados
	query := strings.ToLower(strings.TrimSpace(m.memorySearchInput.Value()))
	var filtered []memory.FactMemory
	for _, f := range m.memoriesList {
		if query == "" || strings.Contains(strings.ToLower(f.Content), query) || strings.Contains(strings.ToLower(string(f.Category)), query) {
			filtered = append(filtered, f)
		}
	}

	listWidth := convWidth - 8
	if listWidth < 40 {
		listWidth = 40
	}
	if listWidth > 90 {
		listWidth = 90
	}

	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(ColorBorder).
		Padding(1, 2).
		Width(listWidth)

	var listSb strings.Builder
	if len(filtered) == 0 {
		if len(m.memoriesList) == 0 {
			listSb.WriteString(MutedStyle.Render("  Sin recuerdos registrados aún en la memoria persistente.\n  Presiona [a] para agregar tu primera preferencia o stack."))
		} else {
			listSb.WriteString(MutedStyle.Render(fmt.Sprintf("  No se encontraron recuerdos que coincidan con %q.", query)))
		}
		listSb.WriteString("\n")
	} else {
		for idx, f := range filtered {
			isSelected := idx == m.memoryIndex
			cat := strings.ToUpper(string(f.Category))
			dateStr := f.CreatedAt.Format("2006-01-02")
			if f.CreatedAt.IsZero() {
				dateStr = "persistido"
			}

			if isSelected {
				cursor := lipgloss.NewStyle().Bold(true).Foreground(ColorPrimary).Render("❯ ")
				badge := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#181e00")).Background(ColorPrimary).Padding(0, 1).Render(" " + cat + " ")
				content := lipgloss.NewStyle().Bold(true).Foreground(ColorText).Render(" " + f.Content)
				meta := MutedStyle.Render(fmt.Sprintf("  Relevancia: %.0f%% · Usos: %d · Fecha: %s · ID: %s", f.Confidence*100, f.AccessCount, dateStr, f.ID))

				listSb.WriteString(cursor + badge + content + "\n" + meta + "\n")

				if m.memoryConfirmDelete == f.ID {
					confirmBox := lipgloss.NewStyle().
						Bold(true).
						Foreground(lipgloss.Color("#181e00")).
						Background(ColorQueue).
						Padding(0, 1).
						Render(" CONFIRMAR ELIMINACIÓN: [s] Eliminar definitivamente  [n / Esc] Cancelar ")
					listSb.WriteString("    " + confirmBox + "\n")
				}
				listSb.WriteString("\n")
			} else {
				cursor := "  "
				badge := lipgloss.NewStyle().Foreground(ColorMuted).Background(ColorContainer).Padding(0, 1).Render(cat)
				content := MutedStyle.Render(" " + f.Content)
				listSb.WriteString(cursor + badge + content + "\n\n")
			}
		}
	}

	renderedList := border.Render(listSb.String())
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(renderedList))
	sb.WriteString("\n")

	// 5. Atajos de navegación
	var hintsStr string
	if m.memoryConfirmDelete != "" {
		hintsStr = " [s] Confirmar borrado  •  [n / Esc] Cancelar "
	} else if m.memoryAddingNew {
		hintsStr = " [Enter] Guardar  •  [Esc] Cancelar "
	} else {
		hintsStr = " [↑ / ↓] Navegar  •  [a] Nuevo recuerdo  •  [d] Eliminar  •  [Esc] Volver "
	}
	sb.WriteString(lipgloss.NewStyle().Width(convWidth).Align(lipgloss.Center).Render(MutedStyle.Render(hintsStr)))
	sb.WriteString("\n")

	return sb.String()
}
