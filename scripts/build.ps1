$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$env:GO111MODULE = 'on'
$env:GOCACHE = Join-Path $projectRoot '.gocache'
$env:GOMODCACHE = Join-Path $projectRoot '.gomodcache'
$version = $env:UNIFIEDPROXY_VERSION
if ([string]::IsNullOrWhiteSpace($version)) {
    $version = (& git -C $projectRoot tag --points-at HEAD | Select-Object -First 1)
    if ([string]::IsNullOrWhiteSpace($version)) {
        $shortCommit = (& git -C $projectRoot rev-parse --short HEAD)
        $version = if ($LASTEXITCODE -eq 0 -and $shortCommit) { "dev-$shortCommit" } else { 'dev' }
    }
}
$version = ($version.Trim() -replace '[^0-9A-Za-z._+\-]', '-')
$linkerFlags = "-s -w -X main.version=$version"
Push-Location $projectRoot
try {
    New-Item -ItemType Directory -Force -Path dist | Out-Null
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'go test failed' }
    go build -trimpath "-ldflags=$linkerFlags" -o unifiedproxy.exe .
    if ($LASTEXITCODE -ne 0) { throw 'Windows build failed' }
    go build -trimpath "-ldflags=$linkerFlags" -o dist\unifiedproxy.exe .
    if ($LASTEXITCODE -ne 0) { throw 'Windows dist build failed' }
    $env:GOOS = 'linux'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    go build -trimpath "-ldflags=$linkerFlags" -o dist\unifiedproxy-linux-amd64 .
    if ($LASTEXITCODE -ne 0) { throw 'Linux build failed' }
    Write-Host "Built unifiedproxy.exe, dist\unifiedproxy.exe and dist\unifiedproxy-linux-amd64 (version $version)"
}
finally {
    Pop-Location
    Remove-Item Env:GOOS, Env:GOARCH, Env:CGO_ENABLED -ErrorAction SilentlyContinue
}
