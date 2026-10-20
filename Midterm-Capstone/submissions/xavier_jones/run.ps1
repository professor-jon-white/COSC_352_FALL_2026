$ErrorActionPreference = 'Stop'
Set-Location $PSScriptRoot
go test ./...
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
go build -o public-safety.exe .
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
& ./public-safety.exe @args
