package agent

import (
	"github.com/ozyassist/backend/internal/providers"
)

var FileTools = []providers.ToolDef{
	{
		Name:        "read_file",
		Description: "Lee el contenido completo de un archivo del proyecto. Úsalo para entender el código antes de modificarlo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta relativa al archivo desde la raíz del proyecto (ej: src/main.go)"
				}
			},
			"required": ["path"]
		}`),
	},
	{
		Name:        "write_file",
		Description: "Crea o sobreescribe completamente un archivo del proyecto. Incluye el contenido completo del archivo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta relativa al archivo desde la raíz del proyecto"
				},
				"content": {
					"type": "string",
					"description": "Contenido completo del archivo"
				}
			},
			"required": ["path", "content"]
		}`),
	},
	{
		Name:        "run_command",
		Description: "Ejecuta un comando en la consola de Windows (PowerShell / Git / CLI). Si el comando es sobre otro proyecto (ej: 'crmgeofal', 'cotizador'), especifica el nombre o ruta en 'cwd'.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"command": {
					"type": "string",
					"description": "Comando a ejecutar (ej: 'git log -n 5 --oneline', 'git status', 'dir')"
				},
				"cwd": {
					"type": "string",
					"description": "Directorio de trabajo o nombre del proyecto donde ejecutar el comando (ej: 'crmgeofal', 'C:\\Users\\User\\Documents\\crmgeofal')"
				}
			},
			"required": ["command"]
		}`),
	},
	{
		Name:        "list_files",
		Description: "Lista archivos del proyecto con un patrón glob. Útil para explorar la estructura del proyecto.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"pattern": {
					"type": "string",
					"description": "Patrón glob (ej: **/*.go, src/**/*.ts, *.json)"
				},
				"max_results": {
					"type": "integer",
					"description": "Máximo número de resultados a devolver (default: 50)"
				}
			},
			"required": ["pattern"]
		}`),
	},
	{
		Name:        "search_text",
		Description: "Busca texto o expresiones regulares dentro de los archivos del proyecto (similar a ripgrep). Devuelve archivo, línea y contenido.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Texto o regex a buscar"
				},
				"include": {
					"type": "string",
					"description": "Patrón glob para filtrar archivos (ej: *.go, *.ts)"
				},
				"case_sensitive": {
					"type": "boolean",
					"description": "Si la búsqueda es sensible a mayúsculas (default: false)"
				},
				"max_results": {
					"type": "integer",
					"description": "Máximo número de coincidencias (default: 30)"
				}
			},
			"required": ["query"]
		}`),
	},
	{
		Name:        "apply_diff",
		Description: "Aplica un diff unificado (formato unified diff) a un archivo. Úsalo para modificaciones quirúrgicas sin reescribir el archivo completo.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta relativa al archivo a parchear"
				},
				"diff": {
					"type": "string",
					"description": "Diff en formato unified diff estándar (--- a/ +++ b/ @@ ...)"
				}
			},
			"required": ["path", "diff"]
		}`),
	},
	{
		Name:        "os_explore",
		Description: "Navega y explora carpetas del sistema de archivos de forma nativa retornando metadatos estructurados (archivos, tamaños, fechas, subdirectorios).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta absoluta o relativa a explorar (ej: C:\\Users\\User\\Documents o .)"
				},
				"depth": {
					"type": "integer",
					"description": "Profundidad máxima de recursión (default: 1)"
				}
			}
		}`),
	},
	{
		Name:        "os_find_files",
		Description: "Busca archivos de forma nativa e instantánea por nombre o patrón en un directorio del sistema.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"root": {
					"type": "string",
					"description": "Directorio raíz donde buscar (ej: C:\\Users\\User o .)"
				},
				"pattern": {
					"type": "string",
					"description": "Nombre de archivo o patrón a buscar (ej: *.png, report, ozymetas)"
				},
				"max_results": {
					"type": "integer",
					"description": "Máximo de resultados (default: 30)"
				}
			}
		}`),
	},
	{
		Name:        "os_create_dir",
		Description: "Crea una carpeta o estructura de directorios en el sistema de archivos de forma nativa e instantánea.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta de la carpeta a crear (ej: C:\\Users\\User\\Documents\\MiProyecto o ./nueva_carpeta)"
				}
			},
			"required": ["path"]
		}`),
	},
	{
		Name:        "os_move_item",
		Description: "Mueve o renombra un archivo o carpeta a una nueva ubicación en el sistema.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"src": {
					"type": "string",
					"description": "Ruta de origen del archivo o carpeta"
				},
				"dst": {
					"type": "string",
					"description": "Ruta de destino del archivo o carpeta"
				}
			},
			"required": ["src", "dst"]
		}`),
	},
	{
		Name:        "os_copy_item",
		Description: "Copia un archivo o carpeta a una nueva ubicación en el sistema.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"src": {
					"type": "string",
					"description": "Ruta de origen del archivo o carpeta"
				},
				"dst": {
					"type": "string",
					"description": "Ruta de destino"
				}
			},
			"required": ["src", "dst"]
		}`),
	},
	{
		Name:        "os_delete_item",
		Description: "Elimina un archivo o carpeta. Por defecto lo envía de forma segura a la Papelera de reciclaje de Windows.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta del archivo o carpeta a eliminar"
				},
				"permanent": {
					"type": "boolean",
					"description": "true para eliminar permanentemente, false para enviar a la Papelera (default: false)"
				}
			},
			"required": ["path"]
		}`),
	},
	{
		Name:        "os_organize_folder",
		Description: "Organiza y clasifica automáticamente todos los archivos de una carpeta (como Escritorio o Descargas) agrupándolos en subcarpetas temáticas (Documentos, Imágenes, Videos, Música, Instaladores, Código, etc.).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta de la carpeta a organizar (ej: C:\\Users\\User\\Desktop o C:\\Users\\User\\Downloads)"
				}
			},
			"required": ["path"]
		}`),
	},
	{
		Name:        "os_read_document",
		Description: "Lee y extrae el texto completo de documentos locales en formatos PDF (.pdf), Excel (.xlsx, .xlsm), Word (.docx), CSV (.csv) y texto plano (.txt, .md, .json, .log). Úsalo para inspeccionar contratos, hojas de cálculo, reportes o cotizaciones.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta absoluta o relativa al documento a leer"
				},
				"max_length": {
					"type": "integer",
					"description": "Máximo de caracteres a extraer (default: 8000)"
				}
			},
			"required": ["path"]
		}`),
	},
	{
		Name:        "os_compress_zip",
		Description: "Comprime uno o más archivos o carpetas en un archivo .zip estándar de forma nativa e instantánea. Úsalo para empaquetar proyectos, documentos o backups.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"src_paths": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Lista de rutas de archivos o carpetas a incluir en el zip"
				},
				"src": {
					"type": "string",
					"description": "Ruta de un archivo o carpeta individual a comprimir (alternativa a src_paths)"
				},
				"dest_zip": {
					"type": "string",
					"description": "Ruta de destino del archivo .zip (ej: 'C:\\Users\\User\\Desktop\\backup.zip')"
				}
			},
			"required": ["dest_zip"]
		}`),
	},
	{
		Name:        "os_extract_zip",
		Description: "Descomprime un archivo .zip en una carpeta de destino de forma segura y con protección contra vulnerabilidades de path traversal.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"zip_path": {
					"type": "string",
					"description": "Ruta del archivo .zip a descomprimir"
				},
				"dest_dir": {
					"type": "string",
					"description": "Carpeta donde se extraerán los archivos (si se omite, se extrae en una subcarpeta junto al zip)"
				}
			},
			"required": ["zip_path"]
		}`),
	},
	{
		Name:        "os_search_content",
		Description: "Busca palabras clave, patrones de texto o expresiones regulares dentro del contenido de archivos en una carpeta o proyecto (Grep nativo ultrarrápido).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"query": {
					"type": "string",
					"description": "Término, palabra o patrón de texto a buscar dentro de los archivos"
				},
				"root": {
					"type": "string",
					"description": "Directorio raíz donde buscar (ej: '.', 'C:\\Users\\User\\Documents\\crmgeofal')"
				},
				"is_regex": {
					"type": "boolean",
					"description": "true si 'query' es una expresión regular, false para coincidencia literal (default: false)"
				},
				"extensions": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Filtro opcional de extensiones (ej: ['.go', '.env', '.json', '.md'])"
				},
				"max_results": {
					"type": "integer",
					"description": "Máximo número de coincidencias a retornar (default: 50)"
				}
			},
			"required": ["query"]
		}`),
	},
	{
		Name:        "os_download_file",
		Description: "Descarga un archivo, imagen, documento o instalador desde una URL web directamente al disco del usuario (por defecto en Downloads o Escritorio).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"url": {
					"type": "string",
					"description": "URL directa de descarga (http:// o https://)"
				},
				"path": {
					"type": "string",
					"description": "Ruta o carpeta donde guardar el archivo descargado (opcional)"
				}
			},
			"required": ["url"]
		}`),
	},
	{
		Name:        "os_smart_organizer",
		Description: "Identifica archivos duplicados por hash criptográfico SHA256 o instaladores huérfanos antiguos (>14 días) en Descargas o una carpeta específica para recuperar espacio en disco.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["duplicates", "clutter"],
					"description": "'duplicates' busca archivos idénticos por hash; 'clutter' busca instaladores y zips antiguos (default: 'duplicates')"
				},
				"target_dir": {
					"type": "string",
					"description": "Ruta de la carpeta a examinar (por defecto la carpeta de Descargas del usuario)"
				}
			}
		}`),
	},
	{
		Name:        "os_file_info",
		Description: "Obtiene información y metadatos detallados de un archivo o carpeta específico: tamaño, fecha y hora exacta de creación, fecha de última modificación y atributos de Windows. Úsalo para saber cuándo fue creado o modificado un archivo o informe.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta absoluta o relativa del archivo o carpeta a inspeccionar (ej: 'C:\\Users\\User\\Documents\\RobloxInfo.pdf')"
				}
			},
			"required": ["path"]
		}`),
	},
}

