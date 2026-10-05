package memory

import (
	"math"
	"testing"
)

func TestPureGoFallbackEmbedding(t *testing.T) {
	// 1. Vector generado y dimensión
	vec1 := PureGoFallbackEmbedding("captura de camara con rust nokhwa", 768)
	if len(vec1) != 768 {
		t.Fatalf("esperaba dimensión 768, obtuve %d", len(vec1))
	}

	// 2. Normalización L2 (la norma debe ser 1.0)
	var norm float64
	for _, v := range vec1 {
		norm += v * v
	}
	if math.Abs(norm-1.0) > 1e-4 {
		t.Fatalf("esperaba norma L2 ~1.0, obtuve %f", norm)
	}

	// 3. Similitud semántica entre textos relacionados
	vecRelated := PureGoFallbackEmbedding("camara web en rust con nokhwa", 768)
	vecUnrelated := PureGoFallbackEmbedding("receta de cocina de fideos y tomate", 768)

	simRelated := cosineSim(vec1, vecRelated)
	simUnrelated := cosineSim(vec1, vecUnrelated)

	if simRelated <= simUnrelated {
		t.Fatalf("esperaba mayor similitud para textos relacionados (rel=%.3f, unrel=%.3f)", simRelated, simUnrelated)
	}
}

func cosineSim(a, b []float64) float64 {
	var dot float64
	for i := range a {
		dot += a[i] * b[i]
	}
	return dot
}
