Write-Host "==========================================" -ForegroundColor Cyan
Write-Host "🚀 API Gateway & Rate Limiting Verification" -ForegroundColor Cyan
Write-Host "==========================================" -ForegroundColor Cyan

# 1. Run Go Unit Tests
Write-Host "`n[1/3] Running Go unit test suite..." -ForegroundColor Yellow
$testResult = go test -v ./gateway/internal/auth ./control-plane/internal/services
if ($LASTEXITCODE -ne 0) {
    Write-Host "❌ Unit tests failed!" -ForegroundColor Red
    exit 1
}
Write-Host "✅ All Go unit tests passed!" -ForegroundColor Green

# 2. Check Compiled Binaries
Write-Host "`n[2/3] Checking compiled Go binaries..." -ForegroundColor Yellow
$binaries = @("gateway/gateway.exe", "control-plane/control-plane.exe", "mock-backend/mock-backend.exe")
foreach ($bin in $binaries) {
    if (Test-Path $bin) {
        $size = (Get-Item $bin).Length / 1MB
        Write-Host "✅ $bin exists ($([math]::Round($size, 2)) MB)" -ForegroundColor Green
    } else {
        Write-Host "❌ Missing binary: $bin" -ForegroundColor Red
        exit 1
    }
}

# 3. Check Next.js Build
Write-Host "`n[3/3] Verifying Next.js Dashboard production build..." -ForegroundColor Yellow
if (Test-Path "dashboard/.next") {
    Write-Host "✅ Next.js .next production artifacts verified!" -ForegroundColor Green
} else {
    Write-Host "❌ Missing Next.js build artifacts" -ForegroundColor Red
    exit 1
}

Write-Host "`n==========================================" -ForegroundColor Cyan
Write-Host "🎉 ALL GATEWAY COMPONENTS VERIFIED SUCCESSFULLY!" -ForegroundColor Green
Write-Host "==========================================" -ForegroundColor Cyan

