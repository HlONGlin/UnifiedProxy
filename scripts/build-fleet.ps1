param([string]$Version = 'dev')
$ErrorActionPreference = 'Stop'
Push-Location (Join-Path $PSScriptRoot '..')
try {
    New-Item -ItemType Directory -Force -Path dist | Out-Null
    foreach ($role in @('control', 'node')) {
        go build -trimpath -ldflags "-s -w -X main.version=$Version" -o "dist/$role.exe" "./cmd/$role"
        if ($LASTEXITCODE -ne 0) { throw "Build failed: $role" }
    }
    $checksums = foreach ($role in @('control', 'node')) {
        $hash = (Get-FileHash "dist/$role.exe" -Algorithm SHA256).Hash.ToLowerInvariant()
        "$hash  $role.exe"
    }
    $checksums | Set-Content -Encoding ascii dist/fleet-sha256sums.txt
} finally { Pop-Location }
