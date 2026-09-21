import os
from huggingface_hub import HfApi

def upload_model():
    repo_id = "Xangel0s/OzyAssist-7B"
    local_file = r"C:\Users\User\Documents\ozyAsis\training\exports\OzyAssist-7B-v3-q4_k_m.gguf"
    
    if not os.path.exists(local_file):
        print(f"[ERROR] No se encuentra el archivo local: {local_file}")
        return
        
    file_size_gb = os.path.getsize(local_file) / (1024 ** 3)
    print(f"Iniciando subida de {local_file} ({file_size_gb:.2f} GB) hacia {repo_id}...")
    
    api = HfApi()
    
    # 1. Subir archivo con nombre v3
    print("Subiendo OzyAssist-7B-v3-q4_k_m.gguf...")
    api.upload_file(
        path_or_fileobj=local_file,
        path_in_repo="OzyAssist-7B-v3-q4_k_m.gguf",
        repo_id=repo_id,
        commit_message="feat: upload OzyAssist-7B-v3 (multi-turn tools, voice mode, no hallucinations)"
    )
    
    # 2. Actualizar también OzyAssist-7B-q4_k_m.gguf para que apunte a la versión reparada
    print("Actualizando archivo principal OzyAssist-7B-q4_k_m.gguf...")
    api.upload_file(
        path_or_fileobj=local_file,
        path_in_repo="OzyAssist-7B-q4_k_m.gguf",
        repo_id=repo_id,
        commit_message="fix: replace broken v2 with stable v3 (fixed token loops and tool execution)"
    )
    
    # 3. Subir README actualizado
    readme_content = """---
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
pipeline_tag: text-generation
---

# OzyAssist-7B-v3 (GGUF Q4_K_M)

**OzyAssist-7B-v3** es un modelo afinado (fine-tuned) sobre `Qwen2.5-Coder-7B-Instruct` con Unsloth 4-bit LoRA, optimizado específicamente como el cerebro autónomo de **OzyAssist** para Windows.

## Mejoras en la Versión 3 (v3)
- **Eliminación Total de Alucinaciones:** Se eliminó el bucle de tokens de repetición (`\\n\\n\\n`) de la versión anterior.
- **Herramientas de Windows Nativas (Multi-Turno):** Entrenado con ejemplos reales de invocación de herramientas (`os_launch_app`, `os_close_window`, `os_audio_device`, `os_hardware_inspector`, `os_tile_windows`, `os_media_control`, etc.).
- **Modo Voz Ultra-Rápido:** Respuestas directas, concisas y sin formato markdown pesado para síntesis de voz fluida en tiempo real con Piper TTS / Windows SAPI.
- **Interacción en Livestream:** Personalidad adaptada para streaming en vivo y respuesta a audiencia.

## Especificaciones Técnicas
- **Formato:** GGUF Cuantizado en **Q4_K_M**
- **Tamaño:** 4.68 GB
- **Arquitectura Base:** Qwen2.5-Coder-7B
- **Optimizaciones Soportadas:**
  - FlashAttention 2 / FlashAttention en llama.cpp (`-fa on`)
  - Speculative Decoding (con borrador `qwen2.5-0.5b-instruct-q4_k_m.gguf` para ~2x de velocidad)
  - YaRN RoPE Scaling para contexto de hasta 64k tokens
  - KV Cache Cuantizado (`q8_0`)

## Archivos Disponibles
- `OzyAssist-7B-v3-q4_k_m.gguf`: Versión v3 recomendada.
- `OzyAssist-7B-q4_k_m.gguf`: Actualizado con la misma compilación v3 estable.
"""
    readme_path = r"C:\Users\User\Documents\ozyAsis\training\exports\README_HF.md"
    with open(readme_path, "w", encoding="utf-8") as f:
        f.write(readme_content)
        
    print("Actualizando README.md del repositorio...")
    api.upload_file(
        path_or_fileobj=readme_path,
        path_in_repo="README.md",
        repo_id=repo_id,
        commit_message="docs: update README with v3 model card and specifications"
    )
    
    print("\n[OK] ¡Subida completada exitosamente a Hugging Face!")
    print(f"URL: https://huggingface.co/{repo_id}")

if __name__ == "__main__":
    upload_model()
