package search

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/ozyassist/backend/internal/providers"
)

type ProviderSearchFunc func(ctx context.Context, query string) ([]SourceCitation, error)

type DeepSearchEngine struct {
	llmProvider providers.Provider
	httpClient  *http.Client
	searchFunc  ProviderSearchFunc
}

func NewDeepSearchEngine(llmProvider providers.Provider, searchFunc ProviderSearchFunc) *DeepSearchEngine {
	if searchFunc == nil {
		searchFunc = DefaultDuckDuckGoSearch
	}
	return &DeepSearchEngine{
		llmProvider: llmProvider,
		httpClient:  &http.Client{Timeout: 8 * time.Second},
		searchFunc:  searchFunc,
	}
}

// Execute realiza la investigación completa y devuelve la síntesis con fuentes
func (d *DeepSearchEngine) Execute(ctx context.Context, originalQuery string) (*DeepSearchResult, error) {
	sourcesMap := make(map[string]SourceCitation)
	var mu sync.Mutex

	// 0. Sondeo directo de dominio/URL si se menciona en la consulta (ej: geofal.com.pe, baseten.co)
	if directSrc := probeDomainOrURL(ctx, originalQuery); directSrc != nil {
		sourcesMap[directSrc.URL] = *directSrc
	}

	// 0.5. Consulta rápida a DuckDuckGo Instant Answer API
	if instantSrc := DuckDuckGoInstantAnswer(ctx, originalQuery); instantSrc != nil {
		if _, exists := sourcesMap[instantSrc.URL]; !exists {
			sourcesMap[instantSrc.URL] = *instantSrc
		}
	}

	// 1. Descomposición de Sub-queries
	subQueries := d.generateSubQueries(ctx, originalQuery)

	// 2. Fetcher Concurrente de Búsqueda
	var wg sync.WaitGroup
	for _, sq := range subQueries {
		wg.Add(1)
		go func(query string) {
			defer wg.Done()
			results, err := d.searchFunc(ctx, query)
			// Si el buscador primario no devolvió resultados, intentar fallback a Wikipedia
			if err != nil || len(results) == 0 {
				results, _ = FallbackWikipediaSearch(ctx, query)
			}
			mu.Lock()
			for _, r := range results {
				if _, exists := sourcesMap[r.URL]; !exists && r.URL != "" {
					sourcesMap[r.URL] = r
				}
			}
			mu.Unlock()
		}(sq)
	}
	wg.Wait()

	// 3. Selección de las mejores fuentes
	var rankedSources []SourceCitation
	idx := 1
	for _, src := range sourcesMap {
		src.ID = idx
		src.Domain = ExtractDomain(src.URL)
		rankedSources = append(rankedSources, src)
		idx++
		if idx > 6 {
			break
		}
	}

	if len(rankedSources) == 0 {
		return &DeepSearchResult{
			Query:      originalQuery,
			Answer:     "No se encontraron fuentes relevantes disponibles para la consulta.",
			Sources:    []SourceCitation{},
			SubQueries: subQueries,
		}, nil
	}

	// 3.5. Descarga concurrente del contenido real de las páginas web (Full-Page Scraping hasta 10.000 chars)
	var scrapeWg sync.WaitGroup
	for i := range rankedSources {
		scrapeWg.Add(1)
		go func(src *SourceCitation) {
			defer scrapeWg.Done()
			fullText := d.fetchPageSnippet(ctx, src.URL)
			if len(fullText) > len(src.Snippet) {
				src.Snippet = fullText
			}
		}(&rankedSources[i])
	}
	scrapeWg.Wait()

	// 4. Síntesis Grounded con Referencias [N]
	answer, err := d.synthesizeGroundedAnswer(ctx, originalQuery, rankedSources)
	if err != nil {
		return nil, fmt.Errorf("error generando síntesis grounded: %w", err)
	}

	res := &DeepSearchResult{
		Query:      originalQuery,
		Answer:     answer,
		Sources:    rankedSources,
		SubQueries: subQueries,
	}

	// 5. Guardado automático de Reporte de Investigación Markdown
	if reportPath, err := SaveResearchReport(res, "data/research"); err == nil {
		res.ReportFile = reportPath
	}

	return res, nil
}

func (d *DeepSearchEngine) queryLLM(ctx context.Context, prompt string, temp float64, maxTokens int) (string, error) {
	if d.llmProvider == nil {
		return "", fmt.Errorf("llmProvider no configurado")
	}

	messages := []providers.Message{
		{Role: "user", Content: prompt},
	}

	chunkCh, err := d.llmProvider.StreamCompletion(ctx, messages, providers.CompletionOptions{
		Temperature: temp,
		MaxTokens:   maxTokens,
		Stream:      false,
	})
	if err != nil {
		return "", err
	}

	var fullText string
	for chunk := range chunkCh {
		if chunk.Type == "text" {
			fullText += chunk.Content
		}
	}
	return strings.TrimSpace(fullText), nil
}

func (d *DeepSearchEngine) generateSubQueries(ctx context.Context, query string) []string {
	if d.llmProvider == nil {
		return []string{query}
	}

	prompt := fmt.Sprintf(`Eres un asistente de investigación DeepSearch.
Divide la siguiente consulta de usuario en 2 o 3 sub-búsquedas atómicas y complementarias para recopilar información técnica precisa.

Consulta: "%s"

Responde estrictamente con un arreglo JSON de cadenas, por ejemplo:
["sub-query 1", "sub-query 2"]`, query)

	text, err := d.queryLLM(ctx, prompt, 0.1, 200)
	if err != nil {
		return []string{query}
	}

	var subQueries []string
	cleanText := strings.TrimSpace(text)
	if start := strings.Index(cleanText, "["); start != -1 {
		if end := strings.LastIndex(cleanText, "]"); end != -1 && end > start {
			_ = json.Unmarshal([]byte(cleanText[start:end+1]), &subQueries)
		}
	}

	if len(subQueries) == 0 {
		subQueries = []string{query}
	}
	return subQueries
}

func (d *DeepSearchEngine) synthesizeGroundedAnswer(ctx context.Context, query string, sources []SourceCitation) (string, error) {
	if d.llmProvider == nil {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Resultados recopilados para \"%s\" según %d fuentes:\n\n", query, len(sources)))
		for _, s := range sources {
			sb.WriteString(fmt.Sprintf("• [%d] **%s** (%s):\n  %s\n\n", s.ID, s.Title, s.Domain, s.Snippet))
		}
		return sb.String(), nil
	}

	var sourcesContext strings.Builder
	for _, s := range sources {
		sourcesContext.WriteString(fmt.Sprintf("[Fuente %d | URL: %s | Título: %s]\n%s\n\n",
			s.ID, s.URL, s.Title, s.Snippet))
	}

	prompt := fmt.Sprintf(`Eres un motor de investigación profunda (DeepSearch).
Responde a la consulta del usuario basándote en las siguientes fuentes numeradas.
Sintetiza la información en español de forma estructurada, clara y objetiva.
Cada afirmación, dato clave, número o hecho DEBE citar su fuente directamente con corchetes [1], [2], etc.
Incluye al final una sección de "Puntos Clave" si es pertinente.

=== FUENTES DISPONIBLES ===
%s

=== CONSULTA DEL USUARIO ===
%s

=== RESPUESTA GROUNDED ===`, sourcesContext.String(), query)

	text, err := d.queryLLM(ctx, prompt, 0.2, 1500)
	if err != nil {
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Resultados recopilados para \"%s\" (%d fuentes):\n\n", query, len(sources)))
		for _, s := range sources {
			sb.WriteString(fmt.Sprintf("• [%d] **%s** (%s):\n  %s\n\n", s.ID, s.Title, s.Domain, s.Snippet))
		}
		return sb.String(), nil
	}
	return text, nil
}

func (d *DeepSearchEngine) fetchPageSnippet(ctx context.Context, pageURL string) string {
	if strings.HasSuffix(pageURL, ".pdf") || strings.HasSuffix(pageURL, ".zip") {
		return ""
	}
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", pageURL, nil)
	if err != nil {
		return ""
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain")

	resp, err := d.httpClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return ""
	}

	return cleanHTMLToText(string(body), 10000)
}

func cleanHTMLToText(rawHTML string, maxLen int) string {
	if maxLen <= 0 {
		maxLen = 10000
	}
	cleaned := rawHTML
	// Eliminar bloques de código, scripts, estilos y menús
	cleaned = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`(?is)<nav[^>]*>.*?</nav>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`(?is)<header[^>]*>.*?</header>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`(?is)<footer[^>]*>.*?</footer>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`(?is)<aside[^>]*>.*?</aside>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`(?is)<iframe[^>]*>.*?</iframe>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`(?is)<svg[^>]*>.*?</svg>`).ReplaceAllString(cleaned, " ")
	cleaned = regexp.MustCompile(`(?is)<form[^>]*>.*?</form>`).ReplaceAllString(cleaned, " ")

	// Estructurar encabezados, párrafos y listas
	cleaned = regexp.MustCompile(`(?i)<h[1-6][^>]*>([\s\S]*?)</h[1-6]>`).ReplaceAllString(cleaned, "\n\n### $1\n")
	cleaned = regexp.MustCompile(`(?i)<p[^>]*>([\s\S]*?)</p>`).ReplaceAllString(cleaned, "\n$1\n")
	cleaned = regexp.MustCompile(`(?i)<li[^>]*>([\s\S]*?)</li>`).ReplaceAllString(cleaned, "\n• $1")
	cleaned = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(cleaned, "\n")

	// Remover etiquetas restantes
	reTags := regexp.MustCompile(`(?s)<[^>]+>`)
	cleaned = reTags.ReplaceAllString(cleaned, " ")

	cleaned = html.UnescapeString(cleaned)

	// Normalizar espacios y saltos de línea repetidos
	reSpaces := regexp.MustCompile(`[ \t]+`)
	cleaned = reSpaces.ReplaceAllString(cleaned, " ")
	reNewlines := regexp.MustCompile(`\n\s*\n\s*\n+`)
	cleaned = strings.TrimSpace(reNewlines.ReplaceAllString(cleaned, "\n\n"))

	if len([]rune(cleaned)) > maxLen {
		runes := []rune(cleaned)
		cleaned = string(runes[:maxLen]) + "..."
	}
	return cleaned
}

// probeDomainOrURL detecta si la consulta contiene un dominio o URL directa y extrae contenido real
func probeDomainOrURL(ctx context.Context, query string) *SourceCitation {
	domainPattern := regexp.MustCompile(`(?i)\b(?:https?://)?([a-zA-Z0-9][-a-zA-Z0-9]*\.(?:com|pe|org|net|io|co|ai|dev|app|edu|gov|cc|me)(?:\.[a-zA-Z]{2})?(?:/[^\s]*)?)\b`)
	match := domainPattern.FindStringSubmatch(query)
	if len(match) < 2 {
		return nil
	}

	rawTarget := match[1]
	targetURL := rawTarget
	if !strings.HasPrefix(targetURL, "http://") && !strings.HasPrefix(targetURL, "https://") {
		targetURL = "https://" + targetURL
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", targetURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil || len(body) == 0 {
		return nil
	}

	content := string(body)
	title := ""
	titleMatch := regexp.MustCompile(`(?i)<title[^>]*>([\s\S]*?)</title>`).FindStringSubmatch(content)
	if len(titleMatch) > 1 {
		title = strings.TrimSpace(html.UnescapeString(titleMatch[1]))
	}
	if title == "" {
		title = ExtractDomain(targetURL)
	}

	snippet := cleanHTMLToText(content, 8000)
	if len(snippet) < 30 {
		return nil
	}

	return &SourceCitation{
		URL:     targetURL,
		Domain:  ExtractDomain(targetURL),
		Title:   title,
		Snippet: snippet,
	}
}

// DuckDuckGoInstantAnswer consulta la API de respuestas directas de DuckDuckGo
func DuckDuckGoInstantAnswer(ctx context.Context, query string) *SourceCitation {
	cleanQuery := strings.TrimSpace(query)
	if cleanQuery == "" {
		return nil
	}

	endpoint := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1", url.QueryEscape(cleanQuery))
	reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "GET", endpoint, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "OzyAssist/2.0")

	client := &http.Client{Timeout: 4 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()

	var ddgResp struct {
		Heading     string `json:"Heading"`
		Abstract    string `json:"Abstract"`
		AbstractURL string `json:"AbstractURL"`
		Answer      string `json:"Answer"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&ddgResp); err != nil {
		return nil
	}

	text := strings.TrimSpace(ddgResp.Abstract)
	if text == "" {
		text = strings.TrimSpace(ddgResp.Answer)
	}
	if text == "" {
		return nil
	}

	srcURL := ddgResp.AbstractURL
	if srcURL == "" {
		srcURL = fmt.Sprintf("https://duckduckgo.com/?q=%s", url.QueryEscape(cleanQuery))
	}
	title := ddgResp.Heading
	if title == "" {
		title = cleanQuery
	}

	return &SourceCitation{
		URL:     srcURL,
		Domain:  ExtractDomain(srcURL),
		Title:   title,
		Snippet: text,
	}
}

// SaveResearchReport exporta un reporte estructurado de investigación en formato Markdown
func SaveResearchReport(res *DeepSearchResult, baseDir string) (string, error) {
	if baseDir == "" {
		baseDir = "data/research"
	}
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", err
	}

	slug := regexp.MustCompile(`[^a-zA-Z0-9_-]+`).ReplaceAllString(strings.ToLower(res.Query), "_")
	slug = strings.Trim(slug, "_")
	if len(slug) > 30 {
		slug = slug[:30]
	}
	if slug == "" {
		slug = "investigacion"
	}

	filename := fmt.Sprintf("research_%s_%s.md", slug, time.Now().Format("20060102_150405"))
	fullPath := filepath.Join(baseDir, filename)

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Reporte de Investigación: %s\n\n", res.Query))
	sb.WriteString(fmt.Sprintf("*Generado por OzyAssist DeepSearch 2.0 | Fecha: %s*\n\n", time.Now().Format("02/01/2006 15:04:05")))
	sb.WriteString("## Resumen y Hallazgos Clave\n\n")
	sb.WriteString(res.Answer)
	sb.WriteString("\n\n---\n\n## Fuentes y Referencias Consultadas\n\n")
	for _, s := range res.Sources {
		sb.WriteString(fmt.Sprintf("- **[%d] [%s](%s)** (`%s`)\n", s.ID, s.Title, s.URL, s.Domain))
		if len(s.Snippet) > 0 {
			short := s.Snippet
			if len([]rune(short)) > 200 {
				short = string([]rune(short)[:200]) + "..."
			}
			sb.WriteString(fmt.Sprintf("  > %s\n\n", strings.ReplaceAll(short, "\n", " ")))
		}
	}

	if err := os.WriteFile(fullPath, []byte(sb.String()), 0644); err != nil {
		return "", err
	}
	return fullPath, nil
}

// FallbackWikipediaSearch busca en Wikipedia en español si el buscador primario no tiene resultados
func FallbackWikipediaSearch(ctx context.Context, query string) ([]SourceCitation, error) {
	endpoint := fmt.Sprintf("https://es.wikipedia.org/w/api.php?action=opensearch&search=%s&limit=3&namespace=0&format=json", url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "OzyAssist/2.0")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var raw []interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil || len(raw) < 4 {
		return nil, fmt.Errorf("respuesta inválida de wikipedia")
	}

	titles, _ := raw[1].([]interface{})
	snippets, _ := raw[2].([]interface{})
	urls, _ := raw[3].([]interface{})

	var results []SourceCitation
	for i := 0; i < len(urls); i++ {
		u, _ := urls[i].(string)
		t, _ := titles[i].(string)
		s, _ := snippets[i].(string)
		if u != "" {
			results = append(results, SourceCitation{
				URL:     u,
				Title:   t,
				Snippet: s,
				Domain:  "es.wikipedia.org",
			})
		}
	}
	return results, nil
}

// cleanDuckDuckGoURL limpia y decodifica las URLs devueltas por DuckDuckGo
func cleanDuckDuckGoURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if strings.Contains(rawURL, "uddg=") {
		parts := strings.Split(rawURL, "uddg=")
		if len(parts) > 1 {
			target := parts[1]
			if ampIdx := strings.Index(target, "&"); ampIdx != -1 {
				target = target[:ampIdx]
			}
			if unescaped, err := url.QueryUnescape(target); err == nil && unescaped != "" {
				return unescaped
			}
		}
	}
	if strings.HasPrefix(rawURL, "//") {
		return "https:" + rawURL
	}
	if !strings.HasPrefix(rawURL, "http") {
		return "https://" + rawURL
	}
	return rawURL
}

// DefaultDuckDuckGoSearch es el proveedor base sin necesidad de API Keys (HTML Lite Scraper via POST)
func DefaultDuckDuckGoSearch(ctx context.Context, query string) ([]SourceCitation, error) {
	formData := url.Values{}
	formData.Set("q", query)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://html.duckduckgo.com/html/", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	htmlContent := string(body)

	var results []SourceCitation
	// Regex para títulos y enlaces en DuckDuckGo HTML Lite: <a rel="nofollow" class="result__a" href="...">Título</a>
	linkRegex := regexp.MustCompile(`(?i)<a[^>]+class="[^"]*result__a[^"]*"[^>]+href="([^"]+)"[^>]*>([\s\S]*?)</a>`)
	snippetRegex := regexp.MustCompile(`(?i)<a[^>]+class="[^"]*result__snippet[^"]*"[^>]*>([\s\S]*?)</a>`)
	tagStripRegex := regexp.MustCompile(`<[^>]+>`)

	links := linkRegex.FindAllStringSubmatch(htmlContent, 8)
	snippets := snippetRegex.FindAllStringSubmatch(htmlContent, 8)

	for i, match := range links {
		cleanURL := cleanDuckDuckGoURL(match[1])
		// Ignorar enlaces internos de DuckDuckGo o publicidad vacía
		if strings.Contains(cleanURL, "duckduckgo.com") && !strings.Contains(cleanURL, "uddg=") {
			continue
		}

		rawTitle := tagStripRegex.ReplaceAllString(match[2], "")
		title := strings.TrimSpace(html.UnescapeString(rawTitle))

		snippet := ""
		if i < len(snippets) {
			rawSnippet := tagStripRegex.ReplaceAllString(snippets[i][1], "")
			snippet = strings.TrimSpace(html.UnescapeString(rawSnippet))
		}

		if cleanURL != "" && title != "" {
			results = append(results, SourceCitation{
				URL:     cleanURL,
				Title:   title,
				Snippet: snippet,
				Domain:  ExtractDomain(cleanURL),
			})
		}
		if len(results) >= 5 {
			break
		}
	}

	return results, nil
}
