# PulseFlow - Go Microservices & React Starter Script
Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "  🚀 Launching PulseFlow Microservices Architecture       " -ForegroundColor Cyan
Write-Host "==========================================================" -ForegroundColor Cyan

# 1. Ensure Go is in PATH
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    if (Test-Path "C:\Program Files\Go\bin") {
        $env:PATH += ";C:\Program Files\Go\bin"
        Write-Host "✔ Added Go to PATH (C:\Program Files\Go\bin)" -ForegroundColor Green
    } else {
        Write-Host "✖ Go compiler not found! Please check Go installation." -ForegroundColor Red
        exit 1
    }
}

$root = $PSScriptRoot

# 2. Start User & Auth Service (:8001)
Write-Host "Starting User Service on :8001..." -ForegroundColor Yellow
$procUser = Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\Backend\services\user-service'; go run cmd/main.go" -PassThru

# 3. Start Project Service (:8002)
Write-Host "Starting Project Service on :8002..." -ForegroundColor Yellow
$procProject = Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\Backend\services\project-service'; go run cmd/main.go" -PassThru

# 4. Start Task & Board Service (:8003)
Write-Host "Starting Task Service on :8003..." -ForegroundColor Yellow
$procTask = Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\Backend\services\task-service'; go run cmd/main.go" -PassThru

# Small delay to let services start before gateway
Start-Sleep -Seconds 2

# 5. Start API Gateway (:8000)
Write-Host "Starting API Gateway on :8000..." -ForegroundColor Green
$procGateway = Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\Backend\api-gateway'; go run main.go" -PassThru

# 6. Start React Frontend (:5173)
Write-Host "Starting React Frontend on :5173..." -ForegroundColor Cyan
$procFrontend = Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$root\Frontend'; npm run dev" -PassThru

Write-Host ""
Write-Host "✔ All microservices and frontend launched successfully!" -ForegroundColor Green
Write-Host "  - React Web UI:  http://localhost:5173" -ForegroundColor White
Write-Host "  - API Gateway:   http://localhost:8000" -ForegroundColor White
Write-Host "  - User Service:  http://localhost:8001" -ForegroundColor White
Write-Host "  - Project Service: http://localhost:8002" -ForegroundColor White
Write-Host "  - Task Service:  http://localhost:8003" -ForegroundColor White
Write-Host ""
