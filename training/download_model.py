import os
import time

os.environ["HF_HUB_ENABLE_HF_TRANSFER"] = "0"
os.environ["HF_XET_HIGH_PERFORMANCE"] = "0"
os.environ["HF_HUB_DISABLE_FAST_DOWNLOAD"] = "1"

from huggingface_hub import snapshot_download

repo = "unsloth/Qwen2.5-Coder-7B-Instruct-bnb-4bit"
success = False
intentos = 0

print("="*60)
print("INICIANDO DESCARGA ROBUSTA DEL MODELO (A PRUEBA DE CORTES)")
print("="*60)

while not success:
    try:
        intentos += 1
        print(f"\n[Intento #{intentos}] Verificando/Descargando archivos...")
        snapshot_download(
            repo_id=repo, 
            resume_download=True, 
            max_workers=1 # Un solo hilo para no saturar tu conexión
        )
        success = True
        print("\n" + "="*60)
        print("¡DESCARGA COMPLETADA CON ÉXITO! El modelo está en caché.")
        print("="*60)
    except Exception as e:
        print(f"\n⚠️ Error de red detectado: {e}")
        print("🔌 Tu conexión a internet se interrumpió o inestabilizó.")
        print("⏳ No te preocupes, reanudando la descarga automáticamente en 10 segundos desde donde se quedó...")
        time.sleep(10)
