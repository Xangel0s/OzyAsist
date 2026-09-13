package search

import (
	"net/url"
	"strings"
)

type SourceCitation struct {
	ID      int    `json:"id"`
	URL     string `json:"url"`
	Domain  string `json:"domain"`
	Title   string `json:"title"`
	Snippet string `json:"snippet"`
}

type DeepSearchResult struct {
	Query      string           `json:"query"`
	Answer     string           `json:"answer"`
	Sources    []SourceCitation `json:"sources"`
	SubQueries []string         `json:"sub_queries"`
	ReportFile string           `json:"report_file,omitempty"`
}

// ExtractDomain obtiene el host limpio a partir de una URL para la UI de fuentes
func ExtractDomain(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	host := parsed.Hostname()
	return strings.TrimPrefix(host, "www.")
}
