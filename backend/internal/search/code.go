package search

// CodeIndex indexa símbolos del proyecto (AST) para búsqueda estructural.
// Por ahora es un placeholder — se implementará cuando el agente lo necesite.
// Estrategia: usar ctags como subproceso, no un parser AST propio.

type CodeSymbol struct {
	Name     string `json:"name"`
	Kind     string `json:"kind"` // "function", "type", "variable", etc.
	FilePath string `json:"filePath"`
	Line     int    `json:"line"`
}

func SearchCode(query string, projectRoot string) ([]CodeSymbol, error) {
	return nil, nil
}
