import os
import sys
from huggingface_hub import HfApi

def create_and_upload_voice_repo():
    repo_id = "Xangel0s/OzyAssist-Voice"
    local_gguf = r"C:\Users\User\Documents\ozyAsis\training\exports\OzyAssist-1.5B-v3-q4_k_m.gguf"
    
    if not os.path.exists(local_gguf):
        print(f"[ERROR] Archivo no encontrado: {local_gguf}")
        sys.exit(1)
        
    api = HfApi()
    
    print(f"=== Creando o verificando repositorio en Hugging Face: {repo_id} ===")
    repo_url = api.create_repo(
        repo_id=repo_id,
        repo_type="model",
        private=False,
        exist_ok=True
    )
    print(f"[OK] Repositorio listo: {repo_url}")
    
    # 1. Crear README.md completo y profesional
    readme_content = """---
language:
- es
- en
license: apache-2.0
base_model: Qwen/Qwen2.5-Coder-1.5B-Instruct
tags:
- ozyassist
- voice-assistant
- speech-to-speech
- ultra-low-latency
- windows-agent
- desktop-automation
- llama-cpp
- gguf
- streaming
pipeline_tag: text-generation
---

# 🎙️ OzyAssist Voice (1.5B Distilled GGUF)

**OzyAssist Voice** es un modelo de inteligencia artificial de ultra-baja latencia (**30+ tokens/segundo**, tiempo de respuesta **<330 ms**), destilado y afinado específicamente para **asistentes de voz interactivos**, **cowork en vivo**, **livestreaming** y **control de Windows**.

Diseñado bajo la arquitectura **Zero-Docker** de [OzyAssist](https://github.com/Xangel0s/OzyAssist), este modelo corre localmente consumiendo apenas **~1.37 GB de VRAM**, lo que permite operarlo en cualquier laptop o GPU mientras juegas o transmites en directo.

---

## ⚡ Especificaciones del Modelo

| Característica | Detalle |
| :--- | :--- |
| **Arquitectura Base** | Qwen2.5-Coder-1.5B-Instruct |
| **Técnica** | Destilación de Conocimiento + Fine-tuning LoRA 4-bit (Unsloth) |
| **Formato** | GGUF Cuantizado (**Q4_K_M**) |
| **Tamaño de Archivo** | **986 MB** |
| **VRAM Requerida** | **~1.37 GB** (Contexto 16k con FlashAttention y KV Q8_0) |
| **Velocidad de Inferencia** | **~30.0 - 32.5 tokens/segundo** (NVIDIA RTX) |
| **Latencia Primer Token / Frase** | **~330 ms** (Ideal para síntesis por voz fluida Piper / SAPI) |
| **Plantilla de Chat** | ChatML (`<|im_start|>`, `<|im_end|>`) |
| **Idioma Principal** | Español (con soporte nativo para comandos en inglés y código) |

---

## 📊 Benchmark: Comparativa de Latencia y Velocidad

Pruebas empíricas ejecutadas en el mismo entorno de hardware (NVIDIA Quadro RTX 4000 Max-Q):

| Tarea / Comando | OzyAssist-Voice (1.5B) | Modelo Estándar (7B) | Mejora |
| :--- | :---: | :---: | :---: |
| **Confirmación de Voz ("¿Estás listo?")** | **0.33s (333 ms)** | 1.75s | **5.3x más rápido** |
| **Reacción Livestream (Saludo en vivo)** | **0.72s** | 2.77s | **3.8x más rápido** |
| **Control de Apps ("Abre calculadora")** | **0.43s** | 1.74s | **4.0x más rápido** |
| **Cerrar Ventana ("Cierra notepad")** | **0.41s** | 1.75s | **4.3x más rápido** |
| **Ajuste de Audio ("Sube volumen 80%")** | **0.54s** | 1.08s | **2.0x más rápido** |
| **Consumo de VRAM** | **1.37 GB** | 4.90 GB | **3.6x menos VRAM** |

---

## 🚀 Cómo Usarlo

### 1. Con Llama.cpp / Llama-Server

Descarga el modelo GGUF y ejecútalo con aceleración GPU:

```bash
llama-server -m OzyAssist-Voice-1.5B-v3-q4_k_m.gguf -ngl 99 -c 16384 -fa on --port 8080
```

### 2. Con Ollama

Crea un archivo `Modelfile`:

```dockerfile
FROM ./OzyAssist-Voice-1.5B-v3-q4_k_m.gguf
PARAMETER temperature 0.2
PARAMETER stop "<|im_end|>"
PARAMETER stop "<|im_start|>"
SYSTEM "Eres OzyAssist Voice, un asistente de voz y escritorio para Windows de ultra-baja latencia."
```

Luego construye y corre el modelo:
```bash
ollama create ozy-voice -f Modelfile
ollama run ozy-voice
```

### 3. En OzyAssist (Modo Turbo / Voz)

En el proyecto **OzyAssist**, puedes iniciar directamente este modelo con el script integrado:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\run_advanced_brain.ps1 -Turbo
```

O iniciar la consola interactiva de voz con soporte de audio en tiempo real:

```powershell
cd backend
go run cmd/ozy/main.go voice
```

---

## 🎯 Capacidades y Casos de Uso

- **Asistentes de Voz en Tiempo Real:** Ideal para integración con motores STT (Whisper/Vosk) y TTS (Piper TTS/Windows SAPI) con *Barge-in* instantáneo.
- **Livestreaming y Bots de Chat:** Respuestas inmediatas al chat de Twitch/YouTube sin pausas que rompan el ritmo de la transmisión.
- **Automatización de Escritorio:** Comandos rápidos para abrir/cerrar ventanas, mutear o ajustar volumen, telemetría de GPU/CPU y atajos en Windows.
- **Decodificación Especulativa (Speculative Decoding):** Puede utilizarse como modelo borrador (*draft model*) para acelerar al modelo maestro de 7B (`Xangel0s/OzyAssist-7B`) al doble de velocidad.

---

## 📄 Licencia

Este modelo se distribuye bajo la licencia **Apache 2.0**.
"""

    readme_local_path = r"C:\Users\User\Documents\ozyAsis\training\exports\README_VOICE_HF.md"
    with open(readme_local_path, "w", encoding="utf-8") as f:
        f.write(readme_content)

    # 2. Crear Modelfile para Ollama
    modelfile_content = """FROM ./OzyAssist-Voice-1.5B-v3-q4_k_m.gguf

PARAMETER temperature 0.2
PARAMETER repeat_penalty 1.15
PARAMETER top_p 0.9
PARAMETER stop "<|im_end|>"
PARAMETER stop "<|im_start|>"
PARAMETER stop "</s>"

TEMPLATE \"\"\"<|im_start|>system
{{ .System }}<|im_end|>
<|im_start|>user
{{ .Prompt }}<|im_end|>
<|im_start|>assistant
{{ .Response }}<|im_end|>\"\"\"

SYSTEM \"\"\"Eres OzyAssist Voice, un asistente autónomo de voz y escritorio para Windows. Respondes de forma directa, cálida, sin markdown excesivo y optimizado para síntesis de voz en tiempo real.\"\"\"
"""
    modelfile_local_path = r"C:\Users\User\Documents\ozyAsis\training\exports\Modelfile_Voice"
    with open(modelfile_local_path, "w", encoding="utf-8") as f:
        f.write(modelfile_content)

    # 3. Subir README.md
    print("[1/3] Subiendo README.md...")
    api.upload_file(
        path_or_fileobj=readme_local_path,
        path_in_repo="README.md",
        repo_id=repo_id,
        commit_message="docs: initial model card and benchmarks for OzyAssist Voice"
    )
    print("[OK] README.md subido.")

    # 4. Subir Modelfile
    print("[2/3] Subiendo Modelfile...")
    api.upload_file(
        path_or_fileobj=modelfile_local_path,
        path_in_repo="Modelfile",
        repo_id=repo_id,
        commit_message="feat: add Ollama Modelfile"
    )
    print("[OK] Modelfile subido.")

    # 5. Subir binario GGUF
    print("[3/3] Subiendo binario GGUF (986 MB)... Esto tomará unos momentos.")
    api.upload_file(
        path_or_fileobj=local_gguf,
        path_in_repo="OzyAssist-Voice-1.5B-v3-q4_k_m.gguf",
        repo_id=repo_id,
        commit_message="feat: upload distilled OzyAssist-Voice-1.5B-v3 Q4_K_M GGUF"
    )
    print("[OK] Archivo GGUF subido exitosamente.")

    print("\n" + "=" * 70)
    print(f"🎉 ¡REPOSITORIO DE VOZ CREADO Y PUBLICADO CON ÉXITO!")
    print(f"URL: https://huggingface.co/{repo_id}")
    print("=" * 70)

if __name__ == "__main__":
    create_and_upload_voice_repo()
