# Meta del Sistema — Visión OzyAssist (Agente Autónomo de Escritorio)

La meta fundamental de **OzyAssist** es convertirse en un agente de escritorio local y autónomo de nivel **Manus**, diseñado para potenciar la productividad y el control del entorno operativo del usuario de forma segura y transparente.

---

## 1. Pilares Clave de la Visión

### 1.1 Control Seguro del Entorno y Dispositivo Local
- Capacidad para interactuar directamente con el sistema operativo local (gestión de archivos, carpetas, procesos y terminales).
- Arquitectura 100% local con base de datos SQLite y control estricto de permisos mediante perfiles de usuario y protección con PIN de 6 dígitos.

### 1.2 Búsqueda Global y Organización Inteligente de Directorios
- Indexación profunda del sistema de archivos para localizar cualquier archivo o fragmento de código instantáneamente.
- Clasificación, limpieza y organización estructurada de carpetas y proyectos mediante agentes en bucle continuo.

### 1.3 Asignación y Ejecución de Tareas en Segundo Plano
- Planificación y ejecución asíncrona de flujos de trabajo extensos (pruebas, compilaciones, auditorías y migraciones de código).
- Notificaciones no intrusivas en cápsula flotante con deduplicación automática y logs en tiempo real.

### 1.4 Interacción por Voz y Activación Contextual
- Detección de voz y comandos hablados para invocar al agente cuando el usuario se dirija a él.
- Asignación de tareas habladas con síntesis de respuesta fluida e intuitiva.

---

## 2. Roadmap de Desarrollo

1. **Fase 1 (Completada)**:
   - Arquitectura Go + React/Vite/TypeScript con WebSocket streaming.
   - Seguridad multiperfil con PIN bcrypt en SQLite.
   - Unificación de la experiencia en un chat inteligente con renderizado de artefactos, diffs y bloques de herramientas.
   - Sincronización dinámica de modelos LLM con proveedores reales (OpenRouter, Anthropic, OpenAI, DeepSeek, Ollama).

2. **Fase 2 (Próxima)**:
   - Indexación global de archivos con búsqueda semántica y de grafo.
   - Módulo de voz (Speech-to-Text / Wake Word / Text-to-Speech) local.
   - Motor de agentes en segundo plano con delegación de subtareas persistentes.
