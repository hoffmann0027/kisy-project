<#
.SYNOPSIS
  One-time: create the KISY app signing key and hand it to CI.

.DESCRIPTION
  Every APK and AAB is signed with this one key. Android updates an installed
  app only when the new build carries the same signature, and Google Play ties
  the listing to it. Lose the key and the app can never be updated again -
  not on phones, not in Play. Keep the .jks file and its password in a
  password manager; this script never writes them into the repository.

  What it does, all on this machine:
    1. asks for a password (twice, hidden);
    2. creates a 4096-bit RSA keystore outside the repository;
    3. stores it in the repository secrets ANDROID_KEYSTORE_B64,
       ANDROID_KEYSTORE_PASSWORD, ANDROID_KEY_ALIAS, ANDROID_KEY_PASSWORD
       (through `gh secret set`, read from stdin - nothing on the command line);
    4. pins the certificate's SHA-256 in the repository variable
       ANDROID_SIGNING_CERT_SHA256, which CI checks every build against.

  Requires: a JDK (keytool) and `gh` signed in to the repository.

.EXAMPLE
  powershell -ExecutionPolicy Bypass -File scripts\android-signing-setup.ps1
#>
param(
  [string]$OutDir = (Join-Path $env:USERPROFILE "Documents\kisy-signing"),
  [string]$Alias = "kisy",
  [string]$Repo = "hoffmann0027/kisy-project"
)

$ErrorActionPreference = "Stop"

function Find-Keytool {
  $cmd = Get-Command keytool -ErrorAction SilentlyContinue
  if ($cmd) { return $cmd.Source }
  $candidates = @()
  if ($env:JAVA_HOME) { $candidates += (Join-Path $env:JAVA_HOME "bin\keytool.exe") }
  $candidates += Get-ChildItem "$env:LOCALAPPDATA\Programs\jdk-*\bin\keytool.exe" -ErrorAction SilentlyContinue | ForEach-Object FullName
  $candidates += Get-ChildItem "$env:ProgramFiles\*\jdk*\bin\keytool.exe" -ErrorAction SilentlyContinue | ForEach-Object FullName
  foreach ($c in $candidates) { if (Test-Path $c) { return $c } }
  throw "keytool not found: install a JDK or set JAVA_HOME"
}

function Read-Secret([string]$prompt) {
  $secure = Read-Host -Prompt $prompt -AsSecureString
  $bstr = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secure)
  try { return [Runtime.InteropServices.Marshal]::PtrToStringBSTR($bstr) }
  finally { [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($bstr) }
}

function Set-RepoSecret([string]$name, [string]$value) {
  $value | gh secret set $name --repo $Repo
  if ($LASTEXITCODE -ne 0) { throw "gh secret set $name failed" }
}

$keytool = Find-Keytool
gh auth status 1>$null 2>$null
if ($LASTEXITCODE -ne 0) { throw "gh is not signed in: run 'gh auth login' first" }

$keystore = Join-Path $OutDir "kisy-signing.jks"
if (Test-Path $keystore) {
  throw "$keystore already exists. It may already be THE key - do not replace it. Move it away deliberately if you really mean to start over."
}
New-Item -ItemType Directory -Force $OutDir | Out-Null

Write-Host "Password for the signing key: at least 16 characters. Save it in the password manager NOW."
$password = Read-Secret "Password"
if ($password.Length -lt 16) { throw "the password is shorter than 16 characters" }
if ((Read-Secret "Repeat the password") -ne $password) { throw "the passwords differ" }

# keytool reads the password from the environment, never from argv.
$env:KISY_SIGNING_PASSWORD = $password
try {
  & $keytool -genkeypair -v -storetype PKCS12 -keystore $keystore -alias $Alias `
    -keyalg RSA -keysize 4096 -validity 10000 `
    -dname "CN=KISY, O=KISY" `
    -storepass:env KISY_SIGNING_PASSWORD -keypass:env KISY_SIGNING_PASSWORD
  if ($LASTEXITCODE -ne 0) { throw "keytool failed" }

  $listing = & $keytool -list -v -keystore $keystore -alias $Alias -storepass:env KISY_SIGNING_PASSWORD
  $line = $listing | Select-String "SHA256:" | Select-Object -First 1
  if (-not $line) { throw "could not read the certificate fingerprint" }
  $sha256 = ($line.ToString() -replace ".*SHA256:\s*", "" -replace ":", "").Trim().ToLower()

  Set-RepoSecret "ANDROID_KEYSTORE_B64" ([Convert]::ToBase64String([IO.File]::ReadAllBytes($keystore)))
  Set-RepoSecret "ANDROID_KEYSTORE_PASSWORD" $password
  Set-RepoSecret "ANDROID_KEY_ALIAS" $Alias
  # PKCS12 has one password for the store and the key.
  Set-RepoSecret "ANDROID_KEY_PASSWORD" $password

  gh variable set ANDROID_SIGNING_CERT_SHA256 --repo $Repo --body $sha256
  if ($LASTEXITCODE -ne 0) { throw "gh variable set failed" }
}
finally {
  Remove-Item Env:KISY_SIGNING_PASSWORD -ErrorAction SilentlyContinue
  $password = $null
}

Write-Host ""
Write-Host "Done."
Write-Host "  Keystore:    $keystore"
Write-Host "  Alias:       $Alias"
Write-Host "  SHA-256:     $sha256"
Write-Host ""
Write-Host "Now, before anything else: put the .jks FILE and the password into the password manager"
Write-Host "(and a second copy somewhere offline). Without them the app can never be updated again."
