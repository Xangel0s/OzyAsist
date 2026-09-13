package audio

import (
	"regexp"
	"strings"
)

// WakeWordMatch representa el resultado del análisis de activación por voz
type WakeWordMatch struct {
	Matched     bool    `json:"matched"`
	Keyword     string  `json:"keyword"`
	Confidence  float64 `json:"confidence"`
	CleanPhrase string  `json:"clean_phrase"`
}

var (
	wakeRegex = regexp.MustCompile(`(?i)\b(hey|oye|hola|ey|jei|ay|hay|ok|okay|oe|e|el|abre|abrir)?\s*(ozy|ozi|osi|osy|ozzy|osie|ossy|ocy|oz)\b`)
	
	exactKeywords = []string{
		"hey ozy", "oye ozy", "hola ozy", "ok ozy", "okey ozy",
		"hey osi", "oye osi", "hey osy", "hey ozi", "oye ozi",
		"ey ozy", "ey ozi", "ey osi", "jei ozy", "jei ozi",
		"jei osi", "hay ozy", "hay ozi", "ay ozy", "ay ozi",
		"el ozy", "el ozi", "oe ozy", "oe ozi", "ozy assist",
		"ozy asist", "ozi assist", "ozi asist", "abrir ozy", "iniciar ozy",
	}
)

// DetectWakeWord evalúa si una frase transcrita contiene el llamado de activación de Ozy
func DetectWakeWord(text string) WakeWordMatch {
	clean := strings.ToLower(strings.TrimSpace(text))
	if clean == "" {
		return WakeWordMatch{Matched: false, Confidence: 0.0}
	}

	// 1. Coincidencia directa exacta
	for _, kw := range exactKeywords {
		if strings.Contains(clean, kw) {
			return WakeWordMatch{
				Matched:     true,
				Keyword:     kw,
				Confidence:  0.98,
				CleanPhrase: clean,
			}
		}
	}

	// 2. Coincidencia por regex fonético
	if loc := wakeRegex.FindString(clean); loc != "" {
		return WakeWordMatch{
			Matched:     true,
			Keyword:     loc,
			Confidence:  0.92,
			CleanPhrase: clean,
		}
	}

	// 3. Palabra aislada "ozy" o "ozi" en frases cortas (< 25 caracteres)
	if len(clean) < 25 && (strings.Contains(clean, "ozy") || strings.Contains(clean, "ozi") || strings.Contains(clean, "ozzy")) {
		return WakeWordMatch{
			Matched:     true,
			Keyword:     "ozy",
			Confidence:  0.88,
			CleanPhrase: clean,
		}
	}

	return WakeWordMatch{
		Matched:     false,
		Keyword:     "",
		Confidence:  0.0,
		CleanPhrase: clean,
	}
}

// IsVoiceActive determina si un nivel de energía RMS supera el umbral de actividad de voz (VAD)
func IsVoiceActive(energy float64, threshold float64) bool {
	if threshold <= 0 {
		threshold = 0.05
	}
	return energy >= threshold
}
