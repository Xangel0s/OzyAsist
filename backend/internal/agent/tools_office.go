package agent

import (
	"github.com/ozyassist/backend/internal/providers"
)

var OfficeTools = []providers.ToolDef{
	{
		Name:        "os_draft_email",
		Description: "Redacta un correo electrónico formal o informal y abre automáticamente la ventana del cliente de correo (Outlook, Thunderbird, Windows Mail, o Gmail/Outlook Web en el navegador) con el destinatario, asunto y cuerpo pre-cargados para revisión y envío.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"to": {
					"type": "string",
					"description": "Dirección de correo del destinatario principal (ej: 'cliente@empresa.com')"
				},
				"subject": {
					"type": "string",
					"description": "Asunto del correo electrónico"
				},
				"body": {
					"type": "string",
					"description": "Cuerpo o contenido completo del correo electrónico redactado"
				},
				"cc": {
					"type": "string",
					"description": "Dirección o direcciones con copia (CC) opcional"
				},
				"client": {
					"type": "string",
					"enum": ["auto", "desktop_app", "gmail", "outlook_web"],
					"description": "Cliente a usar: 'auto' (cliente predeterminado del SO), 'gmail' (pestaña de redacción en Gmail Web), o 'outlook_web'"
				},
				"from_account": {
					"type": "string",
					"description": "Correo o nombre del perfil del navegador (ej: 'zastuto5@gmail.com', 'Tu Chrome', 'Default') para usar esa sesión autenticada directamente sin pedir contraseña"
				},
				"auto_open": {
					"type": "boolean",
					"description": "Si es true, abre la ventana de composición inmediatamente en pantalla (default: true)"
				}
			},
			"required": ["to", "subject", "body"]
		}`),
	},
	{
		Name:        "os_draft_whatsapp",
		Description: "Redacta un mensaje para WhatsApp y abre la conversación (en WhatsApp Web o en la app de escritorio de Windows) con el número del destinatario y el texto pre-cargado listo para enviar con 1 clic.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"phone": {
					"type": "string",
					"description": "Número de teléfono con código de país (ej: '+51999888777' o '51999888777')"
				},
				"text": {
					"type": "string",
					"description": "Mensaje completo a enviar"
				},
				"client": {
					"type": "string",
					"enum": ["auto", "web", "desktop"],
					"description": "Cliente: 'auto' (wa.me universal), 'web' (WhatsApp Web), 'desktop' (app de Windows)"
				},
				"auto_open": {
					"type": "boolean",
					"description": "Si es true, abre la ventana inmediatamente (default: true)"
				}
			},
			"required": ["text"]
		}`),
	},
	{
		Name:        "os_draft_telegram",
		Description: "Redacta un mensaje para Telegram y abre el chat (Telegram Desktop o Telegram Web) con el usuario o canal y el texto pre-cargado listo para enviar con 1 clic.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"recipient": {
					"type": "string",
					"description": "Nombre de usuario (@usuario) o número telefónico del destinatario"
				},
				"text": {
					"type": "string",
					"description": "Mensaje completo a enviar"
				},
				"client": {
					"type": "string",
					"enum": ["auto", "desktop", "web"],
					"description": "Cliente: 'auto' (t.me universal), 'desktop' (app de Windows tg://), 'web' (Telegram Web)"
				},
				"auto_open": {
					"type": "boolean",
					"description": "Si es true, abre la app inmediatamente (default: true)"
				}
			},
			"required": ["text"]
		}`),
	},
	{
		Name:        "telegram_send_message",
		Description: "Envía un mensaje de Telegram 100% en segundo plano sin abrir ninguna ventana de navegador o app, utilizando la API oficial de Telegram Bot (TELEGRAM_BOT_TOKEN y TELEGRAM_CHAT_ID).",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"text": {
					"type": "string",
					"description": "Texto del mensaje a enviar vía Telegram Bot API"
				},
				"chat_id": {
					"type": "string",
					"description": "ID de chat opcional (si se omite, usa el del archivo .env)"
				}
			},
			"required": ["text"]
		}`),
	},
	{
		Name:        "os_create_excel",
		Description: "Crea o genera un archivo de hoja de cálculo nativo de Excel (.xlsx) con tablas, múltiples hojas y cabeceras estilizadas en verde neón (#D1F107). Úsalo para reportes, consolidados, presupuestos o exportación de datos.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta destino del archivo Excel (ej: 'Desktop/reporte_ventas.xlsx', 'crm_data.xlsx')"
				},
				"title": {
					"type": "string",
					"description": "Título de la hoja principal (ej: 'Ventas 2026', 'Inventario')"
				},
				"headers": {
					"type": "array",
					"items": {"type": "string"},
					"description": "Lista de nombres de columnas (ej: ['ID', 'Cliente', 'Total', 'Fecha'])"
				},
				"rows": {
					"type": "array",
					"items": {
						"type": "array",
						"items": {"type": "string"}
					},
					"description": "Matriz bidimensional con los valores de cada fila"
				}
			},
			"required": ["path", "headers", "rows"]
		}`),
	},
	{
		Name:        "os_query_db",
		Description: "Ejecuta consultas SQL de solo lectura (SELECT, PRAGMA) en bases de datos SQLite locales (.db, .sqlite). Bloquea sentencias destructivas. Úsalo para auditar datos, clientes, tareas o métricas de proyectos.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"db_path": {
					"type": "string",
					"description": "Ruta a la base de datos (ej: 'backend/data/ozyassist.db', 'crmgeofal/app.db')"
				},
				"query": {
					"type": "string",
					"description": "Consulta SQL SELECT o PRAGMA a ejecutar"
				},
				"max_rows": {
					"type": "integer",
					"description": "Máximo de filas a devolver (default: 50)"
				}
			},
			"required": ["db_path", "query"]
		}`),
	},
	{
		Name:        "os_create_pdf",
		Description: "Crea un documento PDF profesional nativo con formato estético de alta calidad (cabecera, barra de acento, metadatos, secciones con títulos y viñetas, y tablas opcionales). NUNCA renombres un archivo .xlsx, .txt o .docx a .pdf; usa esta herramienta para generar PDFs válidos.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta completa donde se guardará el archivo PDF (debe terminar en .pdf)"
				},
				"title": {
					"type": "string",
					"description": "Título principal del documento PDF"
				},
				"subtitle": {
					"type": "string",
					"description": "Subtítulo explicativo opcional"
				},
				"author": {
					"type": "string",
					"description": "Autor o entidad que genera el informe (ej: 'OzyAssist')"
				},
				"theme_color": {
					"type": "string",
					"enum": ["lime", "dark", "slate"],
					"description": "Tema visual del documento (default: 'lime')"
				},
				"sections": {
					"type": "array",
					"description": "Lista de secciones del documento con títulos, contenido y viñetas",
					"items": {
						"type": "object",
						"properties": {
							"title": {"type": "string", "description": "Título de la sección"},
							"content": {"type": "string", "description": "Párrafo o contenido explicativo"},
							"bullets": {
								"type": "array",
								"items": {"type": "string"},
								"description": "Puntos clave o viñetas opcionales"
							}
						},
						"required": ["title"]
					}
				},
				"table": {
					"type": "object",
					"description": "Tabla de datos opcional a incluir en el informe",
					"properties": {
						"headers": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Nombres de las columnas"
						},
						"rows": {
							"type": "array",
							"items": {
								"type": "array",
								"items": {"type": "string"}
							},
							"description": "Filas de datos de la tabla"
						}
					}
				}
			},
			"required": ["path", "title", "sections"]
		}`),
	},
	{
		Name:        "os_convert_to_pdf",
		Description: "Convierte un archivo existente (.txt, .md, .csv, .json) a un documento PDF formal con diseño profesional. Úsalo para compilar informes desde datos existentes sin recurrir a renombrar extensiones.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"src": {
					"type": "string",
					"description": "Ruta del archivo fuente (.txt, .md, .csv, .json)"
				},
				"dst": {
					"type": "string",
					"description": "Ruta del archivo PDF destino (ej: 'C:\\Users\\...\\documento.pdf')"
				}
			},
			"required": ["src"]
		}`),
	},
	{
		Name:        "os_create_docx",
		Description: "Crea un documento de Microsoft Word (.docx) formal y editable con formato profesional nativo OpenXML (título, subtítulo, autor, secciones con viñetas y tablas estructuradas). NUNCA renombres un .txt a .docx; usa esta herramienta para generar documentos Word válidos.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "Ruta completa donde se guardará el archivo Word (debe terminar en .docx)"
				},
				"title": {
					"type": "string",
					"description": "Título principal del documento"
				},
				"subtitle": {
					"type": "string",
					"description": "Subtítulo opcional"
				},
				"author": {
					"type": "string",
					"description": "Nombre del autor o empresa emisora"
				},
				"target_pages": {
					"type": "integer",
					"description": "Número objetivo de páginas deseadas (ej: 10 o 20 páginas con saltos de página y capítulos estructurados)"
				},
				"sections": {
					"type": "array",
					"description": "Lista de secciones con títulos, párrafos y viñetas",
					"items": {
						"type": "object",
						"properties": {
							"title": {"type": "string"},
							"content": {"type": "string"},
							"bullets": {
								"type": "array",
								"items": {"type": "string"}
							}
						},
						"required": ["title"]
					}
				},
				"table": {
					"type": "object",
					"description": "Tabla estructurada opcional",
					"properties": {
						"headers": {
							"type": "array",
							"items": {"type": "string"}
						},
						"rows": {
							"type": "array",
							"items": {
								"type": "array",
								"items": {"type": "string"}
							}
						}
					}
				}
			},
			"required": ["path", "title", "sections"]
		}`),
	},
	{
		Name:        "os_speak_text",
		Description: "Sintetiza y pronuncia texto hablado en tiempo real a través de los altavoces de la máquina utilizando la voz en español local de Windows SAPI sin conexión externa ni Docker.",
		InputSchema: mustJSON(`{
			"type": "object",
			"properties": {
				"text": {
					"type": "string",
					"description": "Texto a pronunciar por los altavoces"
				},
				"voice": {
					"type": "string",
					"description": "Voz local específica opcional (ej: 'Microsoft Helena Desktop')"
				}
			},
			"required": ["text"]
		}`),
	},
}
