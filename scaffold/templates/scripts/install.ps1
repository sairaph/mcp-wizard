param(
  [string]$Owner = "${Owner}",
  [string]$Repo = "${BinaryName}",
  [string]$Bin = "${BinaryName}",
  [switch]$Daemon,
  [string]$ConfigureArgs = ""
)

$ErrorActionPreference = "Stop"

function Get-Arch {
  $arch = $env:PROCESSOR_ARCHITECTURE
  switch ($arch) {
    { $_ -in 'AMD64','x64' } { return 'amd64' }
    'ARM64'                  { return 'arm64' }
    default                  { Write-Host "  Unsupported architecture: $arch" -ForegroundColor Red; return }
  }
}

function Get-OS {
  return "windows"
}

$os = Get-OS
$arch = Get-Arch
$assetName = "$Bin-$os-$arch.exe"
$asset = "$Bin-$os-$arch.exe"
$url = "https://github.com/$Owner/$Repo/releases/latest/download/$asset"
$localAppData = if ($env:LOCALAPPDATA) { $env:LOCALAPPDATA } else { Join-Path $env:USERPROFILE "AppData\Local" }
$installDir = Join-Path $localAppData "$Repo\bin"
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

# Verify SHA256 checksum
$checksumUrl = "$url/../SHA256SUMS.txt"
try {
    $checksums = (Invoke-WebRequest -Uri $checksumUrl -UseBasicParsing).Content
    $pattern = ' ' + [regex]::Escape($assetName) + '$'
    $expectedHash = ($checksums -split "`n" | Where-Object { $_ -match $pattern }) -split ' ' | Select-Object -First 1
    if ($expectedHash) {
        $actualHash = (Get-FileHash -Path "$target.new" -Algorithm SHA256).Hash.ToLower()
        if ($expectedHash.ToLower() -ne $actualHash) {
            Write-Host "  SHA256 mismatch." -ForegroundColor Red
            Remove-Item "$target.new" -ErrorAction SilentlyContinue
            exit 1
        }
    }
} catch { }

# Swap the new binary into place using move-aside
$oldTarget = "$target.old-$([System.Guid]::NewGuid().ToString('N').Substring(0,8))"
if (Test-Path $target) {
    Move-Item $target $oldTarget -Force
}
try {
    Move-Item "$target.new" $target -Force
} catch {
    Write-Host "  Could not replace binary. Close any running processes and retry." -ForegroundColor Red
    if (Test-Path $oldTarget) { Move-Item $oldTarget $target -Force }
    Remove-Item "$target.new" -ErrorAction SilentlyContinue
    exit 1
}
Remove-Item $oldTarget -Force -ErrorAction SilentlyContinue

$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installDir*") {
  $newPath = "$installDir;$userPath"
  [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
  $env:Path = "$installDir;$env:Path"
  Write-Host "  Added $installDir to your PATH. Restart your terminal."
}

Write-Host "  Installed $Bin to $target"
