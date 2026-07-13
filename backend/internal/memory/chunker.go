package memory

import (
	"strings"
	"unicode/utf8"
)

const (
	tokensPerChunk = 512
	overlapTokens  = 64
)

// roughTokenCount aproxima tokens (~4 chars por token en promedio para texto en español/inglés)
func roughTokenCount(s string) int {
	return utf8.RuneCountInString(s) / 4
}

type Chunk struct {
	Index   int    `json:"index"`
	Content string `json:"content"`
}

// ChunkText divide texto en chunks con overlap.
func ChunkText(text string) []Chunk {
	tokens := roughTokenCount(text)
	if tokens <= tokensPerChunk {
		return []Chunk{{Index: 0, Content: text}}
	}

	runes := []rune(text)
	chunkRunes := tokensPerChunk * 4
	overlapRunes := overlapTokens * 4

	var chunks []Chunk
	start := 0
	idx := 0

	for start < len(runes) {
		end := start + chunkRunes
		if end > len(runes) {
			end = len(runes)
		}

		// ajustar al límite de oración/párrafo si es posible
		content := strings.TrimSpace(string(runes[start:end]))
		if len(content) > 0 {
			chunks = append(chunks, Chunk{Index: idx, Content: content})
			idx++
		}

		if end >= len(runes) {
			break
		}
		start = end - overlapRunes
		if start < 0 {
			start = 0
		}
	}

	return chunks
}
