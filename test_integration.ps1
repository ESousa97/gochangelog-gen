# Automated Integration Test for gochangelog-gen

$repoUrl = "https://github.com/google/go-cmp.git"
$tempDir = "test_integration_repo"
$executable = "./gochangelog-gen.exe"

Write-Host "--- Starting Automated Integration Test ---" -ForegroundColor Cyan

# 1. Build the application
Write-Host "[1/4] Building application..." -ForegroundColor Yellow
go build -o $executable ./cmd/gochangelog-gen/main.go
if ($LASTEXITCODE -ne 0) {
    Write-Host "Build failed!" -ForegroundColor Red
    exit $LASTEXITCODE
}

# 2. Clone a real repository
Write-Host "[2/4] Cloning real repository ($repoUrl)..." -ForegroundColor Yellow
if (Test-Path $tempDir) { Remove-Item -Recurse -Force $tempDir }
git clone $repoUrl $tempDir
if ($LASTEXITCODE -ne 0) {
    Write-Host "Clone failed!" -ForegroundColor Red
    exit $LASTEXITCODE
}

# 3. Run the tool
Write-Host "[3/4] Running gochangelog-gen in the cloned repo..." -ForegroundColor Yellow
Push-Location $tempDir
& "..\$executable"
if (Test-Path "CHANGELOG_PENDING.md") {
    $output = Get-Content "CHANGELOG_PENDING.md" -Raw
} else {
    $output = ""
}
Pop-Location

# 4. Validate output
Write-Host "[4/4] Validating output..." -ForegroundColor Yellow

$hasVersionHeader = $output -match "# v"
$hasSections = $output -match "## "
$hasOthers = $output -match "## Others"

if ($hasVersionHeader -and $hasSections -and $hasOthers) {
    Write-Host "SUCCESS: Output format is valid!" -ForegroundColor Green
    Write-Host "`nSample of the generated output:" -ForegroundColor Gray
    $output[0..15] | ForEach-Object { Write-Host $_ }
    Write-Host "..."
} else {
    Write-Host "FAILED: Output does not meet expected criteria." -ForegroundColor Red
    Write-Host "Debug Output:"
    $output | ForEach-Object { Write-Host $_ }
    $exitCode = 1
}

# Cleanup
Write-Host "`nCleaning up..." -ForegroundColor Yellow
Remove-Item -Recurse -Force $tempDir
Remove-Item $executable

if ($exitCode -eq 1) { exit 1 }
Write-Host "--- Test Finished Successfully ---" -ForegroundColor Cyan
