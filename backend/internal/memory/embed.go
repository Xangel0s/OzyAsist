package memory

// EmbeddingProvider genera vectores para Caps2.
type EmbeddingProvider interface {
	Embed(text string) ([]float32, error)
	Dimension() int
}
