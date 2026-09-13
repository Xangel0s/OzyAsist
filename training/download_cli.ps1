$env:HF_HUB_ENABLE_HF_TRANSFER="0"
$env:HF_XET_HIGH_PERFORMANCE="0"

$success = $false
$attempt = 1

while (-not $success) {
    Write-Host "Iniciando intento #$attempt con huggingface-cli..."
    
    $process = Start-Process -FilePath ".\venv\Scripts\hf.exe" -ArgumentList "download","unsloth/Qwen2.5-Coder-7B-Instruct" -Wait -NoNewWindow -PassThru
    
    if ($process.ExitCode -eq 0) {
        $success = $true
        Write-Host "¡DESCARGA COMPLETADA CON ÉXITO!"
    } else {
        Write-Host "Error detectado o proceso colgado. Reintentando en 10 segundos..."
        Start-Sleep -Seconds 10
        $attempt++
    }
}
