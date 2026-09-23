import os
from huggingface_hub import HfApi

def upload_1_5b_model():
    repo_id = "Xangel0s/OzyAssist-7B"
    local_file = r"C:\Users\User\Documents\ozyAsis\training\exports\OzyAssist-1.5B-v3-q4_k_m.gguf"
    
    if not os.path.exists(local_file):
        print(f"[ERROR] No se encuentra el archivo local: {local_file}")
        return
        
    file_size_gb = os.path.getsize(local_file) / (1024 ** 3)
    print(f"Iniciando subida de OzyAssist-1.5B ({file_size_gb:.2f} GB) hacia {repo_id}...")
    
    api = HfApi()
    
    api.upload_file(
        path_or_fileobj=local_file,
        path_in_repo="OzyAssist-1.5B-v3-q4_k_m.gguf",
        repo_id=repo_id,
        commit_message="feat: upload distilled OzyAssist-1.5B-v3 for ultra-low latency voice & stream (60+ tps)"
    )
    
    # Actualizar README
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

# OzyAssist Family (GGUF Quantized Models)

Modelos afinados (fine-tuned) para operar como el cerebro autónomo de **OzyAssist** en Windows.

## Modelos Disponibles

| Modelo | Tamaño GGUF | Arquitectura | Velocidad | Caso de Uso Principal |
| :--- | :---: | :---: | :---: | :--- |
| **`OzyAssist-7B-v3-q4_k_m.gguf`** | **4.68 GB** | Qwen2.5-Coder-7B | ~12-18 t/s | **Cerebro Principal:** Razonamiento complejo, cowork, programación y multitarea en Windows. |
| **`OzyAssist-1.5B-v3-q4_k_m.gguf`** | **986 MB** | Qwen2.5-Coder-1.5B | ~50-65 t/s | **Cerebro Turbo / Voice Mode:** Respuestas instantáneas (<300 ms) para comandos por voz, stream en vivo y PCs con poca VRAM. |

## Mejoras de la Versión 3 (v3)
- **Eliminación Total de Alucinaciones:** Cero bucles de saltos de línea (`\\n\\n\\n`) ni copias textuales del System Prompt.
- **Herramientas de Windows Nativas:** Control de ventanas, aplicaciones, audio, telemetría, salud SMART y comandos de sistema.
- **Doble Compatibilidad:** El modelo 1.5B puede actuar como modelo borrador de **Speculative Decoding** para acelerar al 7B hasta 2.5x.
"""
    readme_path = r"C:\Users\User\Documents\ozyAsis\training\exports\README_HF.md"
    with open(readme_path, "w", encoding="utf-8") as f:
        f.write(readme_content)
        
    api.upload_file(
        path_or_fileobj=readme_path,
        path_in_repo="README.md",
        repo_id=repo_id,
        commit_message="docs: add OzyAssist-1.5B specs and model comparison table to README"
    )
    
    print("\n[OK] ¡OzyAssist-1.5B subido exitosamente a Hugging Face!")
    print(f"URL: https://huggingface.co/{repo_id}")

if __name__ == "__main__":
    upload_1_5b_model()
