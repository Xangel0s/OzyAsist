package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/ozyassist/backend/internal/providers"
	"github.com/ozyassist/backend/internal/search"
)

func execWebSearch(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	results, err := search.DefaultDuckDuckGoSearch(ctx, params.Query)
	if err != nil {
		return fmt.Sprintf("error en búsqueda web: %v", err), false
	}
	if len(results) == 0 {
		return "No se encontraron resultados en la web para la consulta.", true
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Resultados de búsqueda web para \"%s\":\n\n", params.Query))
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("[%d] %s\n    URL: %s\n    %s\n\n", i+1, r.Title, r.URL, r.Snippet))
	}
	return sb.String(), true
}

func execDeepSearch(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Query string `json:"query"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}
	prov := providers.GetDefaultOrFirstProvider()
	engine := search.NewDeepSearchEngine(prov, search.DefaultDuckDuckGoSearch)
	result, err := engine.Execute(ctx, params.Query)
	if err != nil {
		return fmt.Sprintf("error en deep search: %v", err), false
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== SÍNTESIS DEEP SEARCH: %s ===\n\n", result.Query))
	sb.WriteString(result.Answer)
	sb.WriteString("\n\n=== FUENTES CONSULTADAS ===\n")
	for _, s := range result.Sources {
		sb.WriteString(fmt.Sprintf("[%d] %s (%s)\n", s.ID, s.Title, s.URL))
	}
	if result.ReportFile != "" {
		sb.WriteString(fmt.Sprintf("\n📄 Reporte completo guardado en: %s\n", result.ReportFile))
	}
	return sb.String(), true
}

func execWebFetch(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		URL       string `json:"url"`
		MaxLength int    `json:"max_length"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}

	target := strings.TrimSpace(params.URL)
	if target == "" {
		return "URL vacía", false
	}

	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		target = "https://" + target
	}

	maxLen := params.MaxLength
	if maxLen <= 0 {
		maxLen = 4000
	}

	req, err := http.NewRequestWithContext(ctx, "GET", target, nil)
	if err != nil {
		return fmt.Sprintf("Error creando petición para %s: %v", target, err), false
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "es-ES,es;q=0.9,en;q=0.8")

	client := &http.Client{
		Timeout: 12 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("demasiadas redirecciones")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error conectando a %s: %v", target, err), false
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return fmt.Sprintf("Error leyendo respuesta de %s: %v", target, err), false
	}

	bodyStr := string(bodyBytes)

	// Extraer metadatos
	titleRegex := regexp.MustCompile(`(?i)<title[^>]*>([\s\S]*?)</title>`)
	descRegex := regexp.MustCompile(`(?i)<meta[^>]+name=["']description["'][^>]+content=["']([\s\S]*?)["']`)
	ogTitleRegex := regexp.MustCompile(`(?i)<meta[^>]+property=["']og:title["'][^>]+content=["']([\s\S]*?)["']`)
	ogDescRegex := regexp.MustCompile(`(?i)<meta[^>]+property=["']og:description["'][^>]+content=["']([\s\S]*?)["']`)
	ogSiteRegex := regexp.MustCompile(`(?i)<meta[^>]+property=["']og:site_name["'][^>]+content=["']([\s\S]*?)["']`)

	title := ""
	if m := titleRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
		title = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	desc := ""
	if m := descRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
		desc = strings.TrimSpace(html.UnescapeString(m[1]))
	} else if m := ogDescRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
		desc = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	ogTitle := ""
	if m := ogTitleRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
		ogTitle = strings.TrimSpace(html.UnescapeString(m[1]))
	}
	ogSite := ""
	if m := ogSiteRegex.FindStringSubmatch(bodyStr); len(m) > 1 {
		ogSite = strings.TrimSpace(html.UnescapeString(m[1]))
	}

	// Limpieza de HTML para texto legible
	textClean := bodyStr
	textClean = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(textClean, "")
	textClean = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(textClean, "")
	textClean = regexp.MustCompile(`(?is)<noscript[^>]*>.*?</noscript>`).ReplaceAllString(textClean, "")
	textClean = regexp.MustCompile(`(?is)<svg[^>]*>.*?</svg>`).ReplaceAllString(textClean, "")

	blockTagsRegex := regexp.MustCompile(`(?i)</?(p|div|h[1-6]|li|br|section|article|header|footer|tr|td)[^>]*>`)
	textClean = blockTagsRegex.ReplaceAllString(textClean, "\n")

	anyTagRegex := regexp.MustCompile(`<[^>]+>`)
	textClean = anyTagRegex.ReplaceAllString(textClean, " ")

	textClean = html.UnescapeString(textClean)

	multiSpaces := regexp.MustCompile(`[ \t]+`)
	multiNewlines := regexp.MustCompile(`\n{3,}`)
	textClean = multiSpaces.ReplaceAllString(textClean, " ")
	textClean = multiNewlines.ReplaceAllString(textClean, "\n\n")
	textClean = strings.TrimSpace(textClean)

	if len(textClean) > maxLen {
		textClean = textClean[:maxLen] + fmt.Sprintf("\n\n[...contenido truncado a %d caracteres...]", maxLen)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== INSPECCIÓN WEB EN VIVO: %s ===\n", resp.Request.URL.String()))
	sb.WriteString(fmt.Sprintf("- Estado HTTP: %d %s\n", resp.StatusCode, http.StatusText(resp.StatusCode)))
	if srv := resp.Header.Get("Server"); srv != "" {
		sb.WriteString(fmt.Sprintf("- Servidor / Hosting: %s\n", srv))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" {
		sb.WriteString(fmt.Sprintf("- Content-Type: %s\n", ct))
	}
	if title != "" {
		sb.WriteString(fmt.Sprintf("- Título: %s\n", title))
	}
	if ogSite != "" {
		sb.WriteString(fmt.Sprintf("- Sitio: %s\n", ogSite))
	}
	if ogTitle != "" && ogTitle != title {
		sb.WriteString(fmt.Sprintf("- Título OG: %s\n", ogTitle))
	}
	if desc != "" {
		sb.WriteString(fmt.Sprintf("- Descripción: %s\n", desc))
	}

	sb.WriteString("\n=== CONTENIDO PRINCIPAL EXTRAÍDO ===\n")
	if textClean != "" {
		sb.WriteString(textClean)
	} else {
		sb.WriteString("(No se pudo extraer texto visible en el cuerpo HTML)")
	}

	return sb.String(), resp.StatusCode >= 200 && resp.StatusCode < 400
}

func execWebDNSLookup(ctx context.Context, tc providers.ToolCall) (string, bool) {
	var params struct {
		Domain string `json:"domain"`
	}
	if err := json.Unmarshal(tc.Input, &params); err != nil {
		return "parámetros inválidos: " + err.Error(), false
	}

	rawDomain := strings.TrimSpace(params.Domain)
	rawDomain = strings.TrimPrefix(rawDomain, "http://")
	rawDomain = strings.TrimPrefix(rawDomain, "https://")
	if slashIdx := strings.Index(rawDomain, "/"); slashIdx != -1 {
		rawDomain = rawDomain[:slashIdx]
	}
	if colonIdx := strings.Index(rawDomain, ":"); colonIdx != -1 {
		rawDomain = rawDomain[:colonIdx]
	}

	if rawDomain == "" {
		return "Dominio vacío", false
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("=== REGISTROS DNS PARA: %s ===\n", rawDomain))

	// 1. Direcciones IP (A / AAAA)
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", rawDomain)
	if err != nil {
		return fmt.Sprintf("Error resolviendo DNS para %s: %v (El dominio no existe o no tiene resolución DNS activa)", rawDomain, err), false
	}

	sb.WriteString(fmt.Sprintf("✓ Direcciones IP encontradas (%d):\n", len(ips)))
	for _, ip := range ips {
		version := "IPv4"
		if ip.To4() == nil {
			version = "IPv6"
		}
		sb.WriteString(fmt.Sprintf("  • [%s] %s\n", version, ip.String()))
	}

	// 2. CNAME
	if cname, err := net.DefaultResolver.LookupCNAME(ctx, rawDomain); err == nil && cname != "" && cname != rawDomain+"." {
		sb.WriteString(fmt.Sprintf("\n- CNAME Alias: %s\n", cname))
	}

	// 3. Registros MX
	if mxs, err := net.DefaultResolver.LookupMX(ctx, rawDomain); err == nil && len(mxs) > 0 {
		sb.WriteString("\n- Servidores de Correo (MX):\n")
		for _, mx := range mxs {
			sb.WriteString(fmt.Sprintf("  • %s (Pref: %d)\n", mx.Host, mx.Pref))
		}
	}

	// 4. Registros TXT (SPF, Verificaciones)
	if txts, err := net.DefaultResolver.LookupTXT(ctx, rawDomain); err == nil && len(txts) > 0 {
		sb.WriteString("\n- Registros TXT:\n")
		for _, txt := range txts {
			if len(txt) > 120 {
				txt = txt[:117] + "..."
			}
			sb.WriteString(fmt.Sprintf("  • %s\n", txt))
		}
	}

	return sb.String(), true
}
