param(
    [string]$Repo = "motoryang/velo-ssh",
    [string]$AppName = "vssh",
    [string]$InstallDir = "$env:LOCALAPPDATA\Programs\VeloSSH\bin",
    [string]$Version = "latest"
)

$ErrorActionPreference = "Stop"

switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { $Arch = "amd64" }
    "ARM64" { $Arch = "arm64" }
    default { throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE" }
}

$Asset = "velossh-windows-$Arch.zip"
$BinName = "velossh-windows-$Arch.exe"
if ($Version -eq "latest") {
    $Url = "https://github.com/$Repo/releases/latest/download/$Asset"
} else {
    $Url = "https://github.com/$Repo/releases/download/$Version/$Asset"
}

$TempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("velossh-install-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Force -Path $TempDir | Out-Null

try {
    $ArchivePath = Join-Path $TempDir $Asset
    Write-Host "Downloading $AppName $Version for windows/$Arch..."
    Invoke-WebRequest -Uri $Url -OutFile $ArchivePath

    Expand-Archive -Path $ArchivePath -DestinationPath $TempDir -Force
    $SourcePath = Join-Path $TempDir $BinName
    if (-not (Test-Path $SourcePath)) {
        throw "Release archive did not contain $BinName"
    }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $BinPath = Join-Path $InstallDir "$AppName.exe"
    Copy-Item -Force -Path $SourcePath -Destination $BinPath

    $UserPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if (($UserPath -split ";") -notcontains $InstallDir) {
        [Environment]::SetEnvironmentVariable("Path", ($UserPath.TrimEnd(";") + ";$InstallDir").TrimStart(";"), "User")
        Write-Host "Added $InstallDir to the user PATH. Open a new terminal to use $AppName."
    }

    Write-Host "Installed $AppName to $BinPath"
    Write-Host "Run: $AppName"
} finally {
    Remove-Item -Recurse -Force -ErrorAction SilentlyContinue $TempDir
}
