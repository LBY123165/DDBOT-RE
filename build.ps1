param(
    [switch]$All,
    [string]$Target = ""
)
$ErrorActionPreference = "Stop"
$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ProjectRoot
$MainPkg = "./cmd/main.go"
$OutputDir = "dist"
$LDFlags = "-s -w"
$Targets = @(
    [pscustomobject]@{ OS="windows"; Arch="amd64"; Out="ddbot-windows-amd64.exe" },
    [pscustomobject]@{ OS="windows"; Arch="arm64"; Out="ddbot-windows-arm64.exe" },
    [pscustomobject]@{ OS="linux";   Arch="amd64"; Out="ddbot-linux-amd64" },
    [pscustomobject]@{ OS="linux";   Arch="arm64"; Out="ddbot-linux-arm64" },
    [pscustomobject]@{ OS="darwin";  Arch="amd64"; Out="ddbot-darwin-amd64" },
    [pscustomobject]@{ OS="darwin";  Arch="arm64"; Out="ddbot-darwin-arm64" }
)
Write-Host "=== DDBOT-WSa Build ===" -ForegroundColor Cyan
Write-Host ""

# ── Step 1: 同步前端 dist 到 embed 目录 ────────────────────────────────────
$FrontendDist = Join-Path $ProjectRoot "DDBOT-WebUI\dist"
$EmbedDist    = Join-Path $ProjectRoot "internal\assets\dist"
if (Test-Path $FrontendDist) {
    Write-Host "Syncing frontend dist -> internal/assets/dist ..." -ForegroundColor Cyan
    if (Test-Path $EmbedDist) { Remove-Item $EmbedDist -Recurse -Force }
    Copy-Item -Recurse $FrontendDist $EmbedDist
    Write-Host "  Sync OK" -ForegroundColor Green
} else {
    Write-Host "  WARNING: DDBOT-WebUI/dist not found, skipping sync" -ForegroundColor Yellow
    Write-Host "  Run: cd DDBOT-WebUI && npm run build" -ForegroundColor Yellow
}
Write-Host ""

# Clean
if (Test-Path $OutputDir) { Remove-Item "$OutputDir\*" -Recurse -Force } else { New-Item -ItemType Directory -Path $OutputDir | Out-Null }
if (Test-Path "ddbot.exe") { Remove-Item "ddbot.exe" -Force }
# Select targets
if ($All) {
    $BuildTargets = $Targets
    Write-Host "Mode: all platforms ($($Targets.Count) targets)" -ForegroundColor Green
} elseif ($Target -ne "") {
    $parts = $Target -split "-"
    $matched = $Targets | Where-Object { $_.OS -eq $parts[0] -and $_.Arch -eq $parts[1] }
    if ($null -eq $matched) { Write-Host "Unknown target: $Target" -ForegroundColor Red; exit 1 }
    $BuildTargets = @($matched)
    Write-Host "Mode: single -> $Target" -ForegroundColor Green
} else {
    $BuildTargets = @($Targets[0])
    Write-Host "Mode: default (windows/amd64). Use -All or -Target os-arch for more." -ForegroundColor Gray
}
Write-Host ""
$Success = @(); $Failed = @()
foreach ($t in $BuildTargets) {
    $OutFile = Join-Path $OutputDir $t.Out
    Write-Host "Building $($t.OS)/$($t.Arch) -> $($t.Out)" -ForegroundColor Cyan
    $env:CGO_ENABLED = "0"; $env:GOOS = $t.OS; $env:GOARCH = $t.Arch
    go build -trimpath -ldflags $LDFlags -o $OutFile $MainPkg 2>&1 | ForEach-Object { Write-Host "  $_" }
    if ($LASTEXITCODE -eq 0) {
        $sz = [math]::Round((Get-Item $OutFile).Length / 1MB, 1)
        Write-Host "  OK ($sz MB)" -ForegroundColor Green
        $Success += "$($t.OS)-$($t.Arch)"
    } else {
        Write-Host "  FAILED" -ForegroundColor Red
        $Failed += "$($t.OS)-$($t.Arch)"
    }
}
Remove-Item Env:GOOS -ErrorAction SilentlyContinue
Remove-Item Env:GOARCH -ErrorAction SilentlyContinue
Write-Host ""
Write-Host "=== Summary ===" -ForegroundColor Cyan
if ($Success.Count -gt 0) { Write-Host "OK  : $($Success -join ', ')" -ForegroundColor Green }
if ($Failed.Count -gt 0)  { Write-Host "FAIL: $($Failed -join ', ')" -ForegroundColor Red }
Write-Host ""
Write-Host "Output: $ProjectRoot\$OutputDir\" -ForegroundColor White
if ($BuildTargets.Count -eq 1 -and $BuildTargets[0].OS -eq "windows" -and -not $All) {
    Copy-Item (Join-Path $OutputDir $BuildTargets[0].Out) "ddbot.exe" -Force
    Write-Host "Shortcut: $ProjectRoot\ddbot.exe" -ForegroundColor White
}
Write-Host ""
if ($Failed.Count -gt 0) { exit 1 }