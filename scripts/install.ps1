param(
    [string]$AppName = "vssh",
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\VeloSSH\bin",
    [string]$VersionLdflags = ""
)

$ErrorActionPreference = "Stop"

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "go is required to build VeloSSH"
}

$RootDir = Resolve-Path (Join-Path $PSScriptRoot "..")
$BinPath = Join-Path $InstallDir "$AppName.exe"
$VersionPath = Join-Path $RootDir "VERSION"
if ([string]::IsNullOrWhiteSpace($VersionLdflags) -and (Test-Path $VersionPath)) {
    $Version = (Get-Content $VersionPath -Raw).Trim()
    $VersionLdflags = "-X github.com/motoryang/velo-ssh/internal/version.Current=$Version"
}

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Write-Host "Building $AppName..."
go build -trimpath -ldflags "$VersionLdflags" -o $BinPath $RootDir

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if (($userPath -split ";") -notcontains $InstallDir) {
    [Environment]::SetEnvironmentVariable("Path", ($userPath.TrimEnd(";") + ";$InstallDir").TrimStart(";"), "User")
    Write-Host "Added $InstallDir to the user PATH. Open a new terminal to use $AppName."
}

Write-Host "Installed $AppName to $BinPath"
Write-Host "Run: $AppName"
