param(
    [string]$QdrantVersion = "v1.18.2"
)

$QdrantDir = Join-Path $PSScriptRoot "..\data\qdrant"
$QdrantExe = Join-Path $QdrantDir "qdrant.exe"
$QdrantData = Join-Path $QdrantDir "storage"
$DownloadUrl = "https://github.com/qdrant/qdrant/releases/download/$QdrantVersion/qdrant-x86_64-pc-windows-msvc.zip"

# Create directories
New-Item -ItemType Directory -Path $QdrantDir -Force | Out-Null
New-Item -ItemType Directory -Path $QdrantData -Force | Out-Null

# Download Qdrant if not present
if (-not (Test-Path $QdrantExe)) {
    Write-Host "Downloading Qdrant $QdrantVersion..." -ForegroundColor Cyan
    $zipPath = Join-Path $env:TEMP "qdrant.zip"
    try {
        Invoke-WebRequest -Uri $DownloadUrl -OutFile $zipPath -ErrorAction Stop
        Expand-Archive -Path $zipPath -DestinationPath $QdrantDir -Force
        Remove-Item $zipPath -Force
        Write-Host "Qdrant downloaded to $QdrantDir" -ForegroundColor Green
    } catch {
        Write-Host "Failed to download Qdrant: $_" -ForegroundColor Red
        Write-Host "Download manually from: $DownloadUrl" -ForegroundColor Yellow
        exit 1
    }
}

# Create config
$ConfigPath = Join-Path $QdrantDir "config.yaml"
if (-not (Test-Path $ConfigPath)) {
    @"
storage:
  storage_path: "$($QdrantData -replace '\\', '/')"
service:
  host: 0.0.0.0
  http_port: 6333
  grpc_port: 6334
"@ | Set-Content -Path $ConfigPath
}

Write-Host "Starting Qdrant on http://localhost:6333 ..." -ForegroundColor Cyan
Write-Host "Storage: $QdrantData" -ForegroundColor DarkGray

& $QdrantExe --config-path $ConfigPath
