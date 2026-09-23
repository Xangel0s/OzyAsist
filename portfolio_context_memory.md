# Memoria de Contexto OzyAssist (Para Portafolio)

**Destino:** Portafolio (`C:\Users\User\Documents\Portfolio`)
**Objetivo:** Agregar una sección que destaque el proceso de OzyAssist, con énfasis en el entrenamiento del modelo.

## 🧠 Arquitectura y Entrenamiento (Puntos Clave a Destacar)

1. **Modelo Base:** Se utilizó `Qwen2.5-Coder-7B-Instruct` como base.
2. **Entrenamiento y Fine-Tuning:** 
   - Se aplicó Fine-Tuning con LoRA (4-bit) a través de la librería `Unsloth`.
   - Se usó un dataset sintético (`generate_dataset.py`) diseñado específicamente para "romper" la barrera típica de un asistente virtual. Se le enseñó a pensar (`<thought>`) e invocar herramientas del SO directamente (`<tool_call>`) para tener autonomía a nivel de kernel/Win32.
3. **Exportación y Optimización:**
   - El resultado fue exportado en formato GGUF con cuantización `Q4_K_M` (tamaño ~4.7GB, `OzyAssist-7B-q4_k_m.gguf`).
   - Este formato fue elegido como el "sweet spot" ideal: máxima velocidad de inferencia sin sacrificar potencial o inteligencia.
   - El modelo fue respaldado en Hugging Face bajo `Xangel0s/OzyAssist-7B`.
4. **Infraestructura Zero-Docker:** Todo corre en local utilizando Go puro y `modernc.org/sqlite` para bases de datos sin dependencias externas.
5. **Sistema de Memorias (Ozygram):** Motor híbrido de memoria continua (SQLite + RAM con indexación FTS5 y búsqueda semántica). El agente extrae hechos en segundo plano (`FactExtractor`) tras cada turno para tener contexto persistente.

> **Instrucción para el próximo Agente:** Lee este archivo y utilízalo como contexto base para diseñar y escribir los componentes de Astro (`CommercialStack.astro`, `PixelMosaicBanner.astro` o cualquier nueva sección) en el proyecto del Portafolio del usuario.
