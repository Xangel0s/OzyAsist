import os
import sys
from huggingface_hub import HfApi

def upload_3b():
    local_file = r"C:\Users\User\Documents\ozyAsis\training\exports\OzyAssist-3B-v6-q4_k_m.gguf"
    if not os.path.exists(local_file):
        print(f"[ERROR] No se encuentra el archivo local: {local_file}")
        return

    file_size_gb = os.path.getsize(local_file) / (1024 ** 3)
    print(f"Iniciando subida de OzyAssist-3B-v6 ({file_size_gb:.2f} GB)...")

    api = HfApi()

    # 1. Subir al repositorio principal de la familia OzyAssist (Xangel0s/OzyAssist-7B) en la rama main
    main_repo = "Xangel0s/OzyAssist-7B"
    print(f"\n[1/2] Subiendo OzyAssist-3B-v6-q4_k_m.gguf a {main_repo} (rama main)...")
    api.upload_file(
        path_or_fileobj=local_file,
        path_in_repo="OzyAssist-3B-v6-q4_k_m.gguf",
        repo_id=main_repo,
        commit_message="feat: upload OzyAssist-3B-v6 GGUF (Office suite 10-20p Word, multi-sheet Excel formulas, sub-100ms search, P2P collaboration)",
        repo_type="model"
    )

    # Actualizar README en Xangel0s/OzyAssist-7B
    readme_hub = """---
language:
- es
- en
license: apache-2.0
base_model: Qwen/Qwen2.5-Coder-7B-Instruct
tags:
- ozyassist
- windows-agent
- desktop-automation
- tool-calling
- voice-assistant
- llama-cpp
- gguf
- reasoning
- cot
- office-automation
pipeline_tag: text-generation
---

# ⚡ OzyAssist Model Family (GGUF Quantized Models)

Modelos oficiales afinados y destilados para operar como el cerebro autónomo de **OzyAssist** en Windows.

## 🏆 Comparativa de la Familia OzyAssist

| Modelo | Tamaño GGUF | Arquitectura | Velocidad | Consumo VRAM | Caso de Uso Recomendado |
| :--- | :---: | :---: | :---: | :---: | :--- |
| **`OzyAssist-3B-v6-q4_k_m.gguf`** | **1.80 GB** | Qwen2.5-Coder-3B + CoT + Office P2P | **~20-28 t/s** | **~2.3 GB** | 🌟 **RECOMENDADO (v6 Flagship):** Manejo completo de Suite Office (Word 10-20 páginas con saltos nativos OpenXML, Excel multi-hoja con fórmulas nativas =SUM/=AVG, PDFs ejecutivos), búsqueda sub-100ms y trato colaborativo P2P. |
| **`OzyAssist-7B-v3-q4_k_m.gguf`** | **4.68 GB** | Qwen2.5-Coder-7B | ~8-14 t/s | ~4.3 GB | **Cerebro Pesado (Maestro):** Tareas complejas de programación, redacción extensa y análisis profundo. |
| **`OzyAssist-1.5B-v3-q4_k_m.gguf`** | **986 MB** | Qwen2.5-Coder-1.5B | ~50-65 t/s | ~1.3 GB | **Ultra-Ligero Turbo / Voice:** Diseñado para comandos rápidos por voz y equipos con recursos muy limitados. |

---

## 🚀 Mejoras Clave de OzyAssist-3B-v6
- **Suite Office Avanzada:** Generación nativa de documentos Word (.docx) de 10 a 20 páginas con capítulos, formateo OpenXML y saltos de página genuinos; libros Excel (.xlsx) de múltiples hojas con fórmulas nativas (`=SUM`, `=AVERAGE`, etc.); y generación de informes ejecutivos en PDF.
- **Interacción Conversacional Persona a Persona (P2P):** Ante peticiones abiertas o ambiguas, actúa proactivamente como un colega técnico ofreciendo preguntas de seguimiento, recomendaciones y próximos pasos.
- **Búsqueda Ultrarrápida Sub-100ms:** Indexación y navegación instantánea por el sistema de archivos de Windows con telemetría de latencia.
- **Destilación CoT (Chain-of-Thought):** Razonamiento antes de invocar herramientas para validar precondiciones del sistema Windows.
- **Ultra-Bajo Consumo de VRAM:** Requiere solo ~2.3 GB de VRAM, permitiendo operar en conjunto con tareas de desarrollo, streaming o juegos.
"""
    readme_path = r"C:\Users\User\Documents\ozyAsis\training\exports\README_HF.md"
    with open(readme_path, "w", encoding="utf-8") as f:
        f.write(readme_hub)

    print(f"Actualizando README en {main_repo}...")
    api.upload_file(
        path_or_fileobj=readme_path,
        path_in_repo="README.md",
        repo_id=main_repo,
        commit_message="docs: update README with OzyAssist-3B-v6 specifications"
    )

    # 2. Crear y subir también al repositorio dedicado Xangel0s/OzyAssist-3B
    dedicated_repo = "Xangel0s/OzyAssist-3B"
    print(f"\n[2/2] Creando/verificando repositorio dedicado {dedicated_repo}...")
    try:
        api.create_repo(repo_id=dedicated_repo, repo_type="model", exist_ok=True)
        print(f"Subiendo modelo a {dedicated_repo}...")
        api.upload_file(
            path_or_fileobj=local_file,
            path_in_repo="OzyAssist-3B-v6-q4_k_m.gguf",
            repo_id=dedicated_repo,
            commit_message="feat: upload OzyAssist-3B-v6 GGUF Q4_K_M"
        )
        api.upload_file(
            path_or_fileobj=readme_path,
            path_in_repo="README.md",
            repo_id=dedicated_repo,
            commit_message="docs: update model card to v6"
        )
        print(f"[OK] Repositorio dedicado listo: https://huggingface.co/{dedicated_repo}")
    except Exception as e:
        print(f"[AVISO] Error opcional en repo dedicado: {e}")

    print("\n[OK] ¡Subida completada con éxito a Hugging Face!")
    print(f"Repositorio Principal: https://huggingface.co/{main_repo}")

if __name__ == "__main__":
    upload_3b()
