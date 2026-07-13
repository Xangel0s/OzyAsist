package memory

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/ozyassist/backend/internal/db"
)

var fileRefRe = regexp.MustCompile(`\b[\w/-]+\.(go|ts|tsx|js|jsx|py|rs|java|c|cpp|h)\b`)

func BuildGraphContext(projectID, userMessage string) string {
	refs := fileRefRe.FindAllString(userMessage, 3)
	if len(refs) == 0 {
		return ""
	}

	seen := map[string]bool{}
	var contextParts []string
	totalNeighbors := 0
	maxNeighbors := 5

	for _, ref := range refs {
		if totalNeighbors >= maxNeighbors {
			break
		}
		if seen[ref] {
			continue
		}
		seen[ref] = true

		edges, err := db.GetGraphNeighbors(projectID, ref)
		if err != nil || len(edges) == 0 {
			continue
		}

		for _, edge := range edges {
			if totalNeighbors >= maxNeighbors {
				break
			}
			neighbor := edge.ToSymbol
			if edge.FromSymbol != ref {
				neighbor = edge.FromSymbol
			}
			if seen[neighbor] {
				continue
			}
			seen[neighbor] = true

			project, err := db.GetProject(projectID)
			if err != nil || project.RootPath == "" {
				continue
			}
			fullPath := filepath.Join(project.RootPath, neighbor)
			content, err := readFileHead(fullPath, 50)
			if err != nil {
				continue
			}
			contextParts = append(contextParts, fmt.Sprintf("--- %s ---\n%s", neighbor, content))
			totalNeighbors++
		}
	}

	if len(contextParts) == 0 {
		return ""
	}
	return strings.Join(contextParts, "\n\n")
}

func readFileHead(path string, maxLines int) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var lines []string
	for i := 0; i < maxLines && scanner.Scan(); i++ {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return fmt.Sprintf("(archivo vacío: %s)", filepath.Base(path)), nil
	}
	if len(lines) >= maxLines {
		lines = append(lines, "...")
	}
	return strings.Join(lines, "\n"), nil
}
