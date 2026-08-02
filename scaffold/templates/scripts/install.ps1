param(
  [string]$Owner = "${Owner}",
  [string]$Repo = "${BinaryName}",
  [string]$Bin = "${BinaryName}",
  [switch]$Daemon,
  [string]$ConfigureArgs = ""
)

$ErrorActionPreference = "Stop"

function Get-Arch {
  $arch = if ([Environment]::Is64BitOperatingSystem) { "amd64" } else { "386" }
  return $arch
}

function Get-OS {
  return "windows"
}

$os = Get-OS
$arch = Get-Arch
$asset = "$Bin-$os-$arch.exe"
$url = "https://github.com/$Owner/$Repo/releases/latest/download/$asset"
$installDir = "$env:LOCALAPPDATA\$Repo\bin"
$target = "$installDir\$Bin.exe"

New-Item -ItemType Directory -Force -Path $installDir | Out-Null

Write-Host "  $Bin installer"
Write-Host "  Downloading $asset..."

try {
  [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
  Invoke-WebRequest -Uri $url -OutFile "$target.new" -UseBasicParsing -ErrorAction Stop
} catch {
  Write-Host "  Download failed: $_" -ForegroundColor Red
  exit 1
}

if (-not (Test-Path "$target.new")) {
  Write-Host "  Download did not complete." -ForegroundColor Red
  exit 1
}

Move-Item -Force "$target.new" $target

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
  $newPath = "$installDir;$userPath"
  [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
  $env:Path = "$installDir;$env:Path"
  Write-Host "  Added $installDir to your PATH. Restart your terminal."
}

Write-Host "  Installed $Bin to $target"
