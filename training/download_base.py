import os
import time
import socket
socket.setdefaulttimeout(15)

os.environ["HF_HUB_ENABLE_HF_TRANSFER"] = "0"
os.environ["HF_XET_HIGH_PERFORMANCE"] = "0"
os.environ["HF_HUB_DISABLE_FAST_DOWNLOAD"] = "1"

from huggingface_hub import snapshot_download

# ESTE ES EL MODELO BASE DE 15 GB QUE UNSLOTH NECESITA PARA EXPORTAR EL GGUF
repo = "unsloth/Qwen2.5-Coder-7B-Instruct" 
success = False
intentos = 0

print("="*60)
print("INICIANDO DESCARGA DEL MODELO BASE GIGANTE (15 GB)")
print("Este proceso tomará varias horas, pero es resistente a caídas.")
print("="*60)

while not success:
    try:
        intentos += 1
        print(f"\n[Intento #{intentos}] Verificando/Descargando archivos...")
        snapshot_download(
            repo_id=repo, 
            resume_download=True, 
            max_workers=1
        )
        success = True
        print("\n" + "="*60)
        print("¡DESCARGA BASE DE 15GB COMPLETADA CON ÉXITO!")
        print("="*60)
    except Exception as e:
        print(f"\nError de red detectado: {e}")
        print("Reanudando la descarga automaticamente en 10 segundos desde donde se quedo...")
        time.sleep(10)
