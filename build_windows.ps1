# build_windows.ps1
# Script to build GopherAutomate natively on Windows

$AppName = "GopherAutomate"
$AppID = "cc.pixp.GopherAutomate"
$IconSource = "assets/app_icon.png"

# Extract version from CHANGELOG.md (Scanning for ## [0.0.1])
$Version = "0.0.1"
if (Test-Path "CHANGELOG.md") {
    $content = Get-Content "CHANGELOG.md"
    foreach ($line in $content) {
        if ($line -match "^## \[([0-9]+\.[0-9]+\.[0-9]+)\]") {
            $Version = $matches[1]
            break
        }
    }
}
Write-Host "Detected version: $Version"

# Check for Fyne
if (-not (Get-Command "fyne" -ErrorAction SilentlyContinue)) {
    Write-Host "Installing Fyne CLI..."
    go install fyne.io/tools/cmd/fyne@latest
}

Write-Host "Packaging for Windows..."
# Note: On Windows, 'fyne package' defaults to windows/amd64
fyne package -os windows -icon $IconSource -name $AppName -app-id $AppID -app-version $Version

if ($LASTEXITCODE -eq 0) {
    Write-Host "Build success! ${AppName}.exe created."
} else {
    Write-Host "Build failed."
    exit 1
}
