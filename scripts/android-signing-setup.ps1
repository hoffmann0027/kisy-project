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

# The secret goes to gh as exact bytes on its stdin. Piping a string to a native
# program in Windows PowerShell 5.1 re-encodes it (UTF-8 with a BOM on this
# machine) and appends a newline: a BOM in ANDROID_KEYSTORE_B64 breaks
# `base64 -d` in CI, and in the password it leaves the keystore unopenable.
function Set-RepoSecret([string]$name, [string]$value) {
  $gh = (Get-Command gh -CommandType Application | Select-Object -First 1).Source
  $psi = New-Object System.Diagnostics.ProcessStartInfo
  $psi.FileName = $gh
  $psi.Arguments = "secret set $name --repo $Repo"
  $psi.UseShellExecute = $false
  $psi.RedirectStandardInput = $true
  $psi.RedirectStandardOutput = $true
  $psi.RedirectStandardError = $true
  # .NET Framework opens the child's stdin writer in Console.InputEncoding and
  # writes that encoding's preamble at once; UTF-8 without a BOM has none.
  $savedInput = [Console]::InputEncoding
  [Console]::InputEncoding = New-Object System.Text.UTF8Encoding $false
  try { $p = [System.Diagnostics.Process]::Start($psi) }
  finally { [Console]::InputEncoding = $savedInput }
  $stdout = $p.StandardOutput.ReadToEndAsync()
  $stderr = $p.StandardError.ReadToEndAsync()
  $bytes = (New-Object System.Text.UTF8Encoding $false).GetBytes($value)
  $p.StandardInput.BaseStream.Write($bytes, 0, $bytes.Length)
  $p.StandardInput.Close()
  $p.WaitForExit()
  if ($p.ExitCode -ne 0) {
    throw "gh secret set $name failed (exit $($p.ExitCode))$([Environment]::NewLine)$($stderr.Result)$($stdout.Result)"
  }
}

$keytool = Find-Keytool
try { Invoke-Native "gh auth status" { gh auth status } | Out-Null }
catch { throw "gh is not signed in: run 'gh auth login' first$([Environment]::NewLine)$($_.Exception.Message)" }

$keystore = Join-Path $OutDir "kisy-signing.jks"
if (Test-Path $keystore) {
  throw "$keystore already exists. It may already be THE key - do not replace it. Move it away deliberately if you really mean to start over."
}
New-Item -ItemType Directory -Force $OutDir | Out-Null

Write-Host "Password for the signing key: at least 16 characters. Save it in the password manager NOW."
$password = Read-Secret "Password"
if ($password.Length -lt 16) { throw "the password is shorter than 16 characters" }
# Printable ASCII only: the password crosses keytool's environment on Windows
# and Gradle's on the Linux runner; outside ASCII that depends on code pages.
if ($password -notmatch "^[!-~]+$") { throw "use printable ASCII only (letters, digits, punctuation; no spaces)" }
if ((Read-Secret "Repeat the password") -ne $password) { throw "the passwords differ" }

# keytool reads the password from the environment, never from argv.
$env:KISY_SIGNING_PASSWORD = $password
try {
  Invoke-Native "keytool -genkeypair" {
    & $keytool -genkeypair -v -storetype PKCS12 -keystore $keystore -alias $Alias `
      -keyalg RSA -keysize 4096 -validity 10000 `
      -dname "CN=KISY, O=KISY" `
      -storepass:env KISY_SIGNING_PASSWORD -keypass:env KISY_SIGNING_PASSWORD
  } -Stream | Out-Null

  $listing = Invoke-Native "keytool -list" { & $keytool -list -v -keystore $keystore -alias $Alias -storepass:env KISY_SIGNING_PASSWORD }
  $line = $listing | Select-String "SHA256:" | Select-Object -First 1
  if (-not $line) { throw "could not read the certificate fingerprint" }
  $sha256 = ($line.ToString() -replace ".*SHA256:\s*", "" -replace ":", "").Trim().ToLower()

  Set-RepoSecret "ANDROID_KEYSTORE_B64" ([Convert]::ToBase64String([IO.File]::ReadAllBytes($keystore)))
  Set-RepoSecret "ANDROID_KEYSTORE_PASSWORD" $password
  Set-RepoSecret "ANDROID_KEY_ALIAS" $Alias
  # PKCS12 has one password for the store and the key.
  Set-RepoSecret "ANDROID_KEY_PASSWORD" $password

  Invoke-Native "gh variable set ANDROID_SIGNING_CERT_SHA256" { gh variable set ANDROID_SIGNING_CERT_SHA256 --repo $Repo --body $sha256 } | Out-Null
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
