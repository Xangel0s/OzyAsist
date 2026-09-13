Write-Host "Iniciando instalación del entorno virtual de OzyAssist Fine-Tuning..." -ForegroundColor Cyan

# 1. Crear entorno virtual si no existe
if (-not (Test-Path "venv")) {
    Write-Host "Creando entorno virtual (venv)..."
    python -m venv venv
}

# 2. Activar entorno virtual
$env:Path = "$PSScriptRoot\venv\Scripts;" + $env:Path

# 3. Actualizar pip
Write-Host "Actualizando pip..."
python -m pip install --upgrade pip

# 4. Instalar PyTorch con soporte CUDA 12.1 (Recomendado para Unsloth en Windows)
Write-Host "Instalando PyTorch (Esto puede tomar varios minutos debido al tamaño de 3GB+)..." -ForegroundColor Yellow
pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cu121

# 5. Instalar dependencias de Unsloth y HuggingFace
Write-Host "Instalando Unsloth y dependencias..." -ForegroundColor Yellow
pip install -r requirements.txt

Write-Host "¡Instalación completada!" -ForegroundColor Green
Write-Host "Para entrenar el modelo, ejecuta: python finetune_ozy.py dentro del venv." -ForegroundColor Cyan
