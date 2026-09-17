<#
.SYNOPSIS
  Build a tester APK on this machine, signed with the app key.

.DESCRIPTION
  The same build CI makes ("Android build"), for when the APK is needed before
  the code is pushed. It is signed with the one app key, so it installs over
  every later build - this is the last reinstall a tester has to do.

  The keystore password is asked for here, hidden, and handed to Gradle through
  this process's environment only. The signature is checked against the pinned
  fingerprint in the repository variable ANDROID_SIGNING_CERT_SHA256 before the
  APK is copied out.

  Requires: Node/npm, JDK 21, the Android SDK, `gh` signed in, and the key
  created by scripts/android-signing-setup.ps1.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts\android-build-signed.ps1
#>
param(
  [string]$Keystore = (Join-Path $env:USERPROFILE "Documents\kisy-signing\kisy-signing.jks"),
  [string]$Alias = "kisy",
  [string]$GoogleServices = (Join-Path $env:USERPROFILE "Downloads\google-services.json"),
  [string]$ApiOrigin = "https://kisy.onrender.com",
  [string]$OutDir = (Join-Path $env:USERPROFILE "Desktop"),
  [string]$Repo = "hoffmann0027/kisy-project"
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $PSScriptRoot
$frontend = Join-Path $root "frontend"
$android = Join-Path $frontend "android"

function Find-Jdk21 {
  $candidates = @()
  if ($env:JAVA_HOME) { $candidates += $env:JAVA_HOME }
  $candidates += Get-ChildItem "$env:LOCALAPPDATA\Programs\jdk-21*" -Directory -ErrorAction SilentlyContinue | ForEach-Object FullName
  $candidates += Get-ChildItem "$env:ProgramFiles\Eclipse Adoptium\jdk-21*" -Directory -ErrorAction SilentlyContinue | ForEach-Object FullName
  $candidates += "$env:ProgramFiles\Android\Android Studio\jbr"
  foreach ($c in $candidates) {
    $release = Join-Path $c "release"
    if ((Test-Path $release) -and ((Get-Content $release -Raw) -match 'JAVA_VERSION="2[1-9]')) { return $c }
  }
  throw "JDK 21 not found (Capacitor 8 needs it): install one or set JAVA_HOME"
}

function Read-Secret([string]$prompt) {
  $secure = Read-Host -Prompt $prompt -AsSecureString
  $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
  try { return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr) }
  finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr) }
}

# Windows PowerShell 5.1 turns every line a native program writes to stderr
# into an error record, and with ErrorActionPreference=Stop that aborts the
# script although the program succeeded: gh prints its status to stderr,
# keytool its progress, npm and Gradle their warnings. Native programs are
# therefore judged by their exit code alone. Returns stdout; with -Stream, every
# line is shown as it arrives instead.
function Invoke-Native([string]$what, [scriptblock]$command, [switch]$Stream) {
  $saved = $ErrorActionPreference
  $ErrorActionPreference = "Continue"
  $stdout = New-Object System.Collections.Generic.List[string]
  $all = New-Object System.Collections.Generic.List[string]
  try {
    & $command 2>&1 | ForEach-Object {
      $line = if ($_ -is [System.Management.Automation.ErrorRecord]) { $_.Exception.Message } else { [string]$_ }
      if ($_ -isnot [System.Management.Automation.ErrorRecord]) { $stdout.Add($line) }
      $all.Add($line)
      if ($Stream) { Write-Host $line }
    }
    $code = $LASTEXITCODE
  }
  finally { $ErrorActionPreference = $saved }
  if ($code -ne 0) {
    $tail = ($all | Select-Object -Last 20) -join [Environment]::NewLine
    throw "$what failed (exit $code)$([Environment]::NewLine)$tail"
  }
  return ,$stdout.ToArray()
}

function Invoke-Step([string]$title, [scriptblock]$body) {
  Write-Host "== $title"
  Invoke-Native $title $body -Stream | Out-Null
}

if (-not (Test-Path $Keystore)) { throw "keystore not found: $Keystore (run scripts\android-signing-setup.ps1 first)" }
if (-not (Test-Path $GoogleServices)) { throw "google-services.json not found: $GoogleServices (push notifications need it)" }

try { $want = (Invoke-Native "gh variable get" { gh variable get ANDROID_SIGNING_CERT_SHA256 --repo $Repo }) -join "" }
catch { throw "repository variable ANDROID_SIGNING_CERT_SHA256 is not readable (run scripts\android-signing-setup.ps1)$([Environment]::NewLine)$($_.Exception.Message)" }
$want = ($want -replace "[:\s]", "").ToLower()
if (-not $want) { throw "repository variable ANDROID_SIGNING_CERT_SHA256 is empty (run scripts\android-signing-setup.ps1)" }

$env:JAVA_HOME = Find-Jdk21
$sdk = if ($env:ANDROID_HOME) { $env:ANDROID_HOME } else { Join-Path $env:LOCALAPPDATA "Android\Sdk" }
if (-not (Test-Path $sdk)) { throw "Android SDK not found: set ANDROID_HOME" }
$env:ANDROID_HOME = $sdk
$env:PATH = "$env:JAVA_HOME\bin;$env:PATH"

$commit = (Invoke-Native "git rev-parse" { git -C $root rev-parse --short HEAD }) -join ""
$dirty = Invoke-Native "git status" { git -C $root status --porcelain -- frontend }
if ($dirty) { Write-Warning "frontend has uncommitted changes; the APK will include them" }

$googleTarget = Join-Path $android "app\google-services.json"
try {
  $env:KISY_KEYSTORE_PASSWORD = Read-Secret "Signing key password"
  $env:KISY_KEY_PASSWORD = $env:KISY_KEYSTORE_PASSWORD
  $env:KISY_KEYSTORE_FILE = $Keystore
  $env:KISY_KEY_ALIAS = $Alias
  $env:KISY_VERSION_CODE = [string][math]::Floor([DateTimeOffset]::UtcNow.ToUnixTimeSeconds() / 60)
  $env:KISY_VERSION_NAME = "dev-$commit"

  Push-Location $frontend
  try {
    $env:VITE_NATIVE_API_ORIGIN = $ApiOrigin
    Invoke-Step "web bundle" { npm run build }
    Copy-Item $GoogleServices $googleTarget -Force
    Invoke-Step "sync into the Android project" { npx cap sync android }
  } finally { Pop-Location }

  Push-Location $android
  try {
    Invoke-Step "native unit tests + signed debug APK" { .\gradlew.bat --no-daemon testDebugUnitTest assembleDebug }
  } finally { Pop-Location }
}
finally {
  Remove-Item Env:KISY_KEYSTORE_PASSWORD, Env:KISY_KEY_PASSWORD -ErrorAction SilentlyContinue
  Remove-Item $googleTarget -ErrorAction SilentlyContinue
}

$apk = Join-Path $android "app\build\outputs\apk\debug\app-debug.apk"
$apksigner = Get-ChildItem (Join-Path $sdk "build-tools\*\lib\apksigner.jar") | Sort-Object FullName | Select-Object -Last 1
$java = Join-Path $env:JAVA_HOME "bin\java.exe"
$certs = Invoke-Native "apksigner verify" { & $java -jar $apksigner.FullName verify --print-certs $apk }
$got = (($certs | Select-String "Signer #1 certificate SHA-256 digest:" | Select-Object -First 1).ToString() -replace ".*digest:\s*", "").Trim().ToLower()
if ($got -ne $want) { throw "the APK is signed with $got, not the app key $want - do NOT distribute it" }

$out = Join-Path $OutDir "kisy-$commit.apk"
Copy-Item $apk $out -Force
$hash = (Get-FileHash $out -Algorithm SHA256).Hash.ToLower()
Write-Host ""
Write-Host "Signed with the app key ($got)."
Write-Host "  APK:     $out"
Write-Host "  SHA-256: $hash"
