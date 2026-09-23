package agent

import (
	"github.com/ozyassist/backend/internal/providers"
)

var WebTools = []providers.ToolDef{
	{
		Name:        "web_search",
		Description: "Busca en la web información técnica actualizada, paquetes, documentación, noticias o soluciones a errores utilizando fuentes confiables de internet.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Término de búsqueda en internet"
				}
			},
			"required": ["query"]
		}`),
	},
	{
		Name:        "deep_search",
		Description: "Realiza una investigación profunda en internet dividiendo la consulta en sub-búsquedas, extrayendo fuentes confiables y generando síntesis fundamentada con citas.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Pregunta o tema complejo para investigación exhaustiva en internet"
				}
			},
			"required": ["query"]
		}`),
	},
	{
		Name:        "web_fetch",
		Description: "Descarga e inspecciona directamente el contenido de cualquier página web o URL en vivo (HTTP/HTTPS). Extrae título, metadatos, encabezados, OpenGraph y texto Markdown legible. Úsalo SIEMPRE que el usuario mencione un dominio (ej: peruflack.com), sitio web, artículo o para comprobar si un sitio está activo y qué contiene.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "URL o nombre de dominio a inspeccionar (ej: 'https://peruflack.com' o 'peruflack.com')"
				},
				"max_length": {
					"type": "integer",
					"description": "Límite opcional de caracteres de contenido a extraer (default: 4000)"
				}
			},
			"required": ["url"]
		}`),
	},
	{
		Name:        "web_dns_lookup",
		Description: "Consulta registros DNS (A, AAAA, CNAME, MX, TXT) y resuelve las direcciones IP de cualquier dominio en internet para verificar si existe, a qué servidor apunta (Vercel, AWS, Cloudflare, etc.) y si está activo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"domain": {
					"type": "string",
					"description": "Nombre de dominio a consultar (ej: 'peruflack.com')"
				}
			},
			"required": ["domain"]
		}`),
	},
	{
		Name:        "browser_list_profiles",
		Description: "Detecta e inspecciona todos los perfiles de navegador instalados en el sistema (Google Chrome, Microsoft Edge, Brave) junto con sus cuentas asociadas (correos de Google/Microsoft, nombres y directorios de perfil) para abrir sesiones y correos autenticados sin pedir contraseñas.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "browser_open_groq",
		Description: "Abre la consola de API Keys de Groq (https://console.groq.com/keys) en Google Chrome con el perfil autenticado del usuario para obtener la clave con 1 solo clic.",
		InputSchema: mustJSON(`{"type":"object","properties":{}}`),
	},
	{
		Name:        "os_setup_groq_key",
		Description: "Valida y guarda automáticamente una API Key de Groq (gsk_...) leída del portapapeles o parámetro en el archivo .env, activando el motor de voz y Hey Ozy inmediatamente sin reiniciar.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"api_key": {
					"type": "string",
					"description": "Clave opcional de Groq (si se omite, Ozy la detectará automáticamente de lo que tengas copiado en el portapapeles)"
				}
			}
		}`),
	},
}
