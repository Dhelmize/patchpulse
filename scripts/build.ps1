# Refresh PATH if running in an existing shell before environment reload
$env:Path = [System.Environment]::GetEnvironmentVariable("Path","Machine") + ";" + [System.Environment]::GetEnvironmentVariable("Path","User")

# PowerShell Cross-Compilation Script for PatchPulse
$targets = @(
    @{ OS = "windows"; ARCH = "arm64"; EXT = ".exe" },
    @{ OS = "windows"; ARCH = "amd64"; EXT = ".exe" },
    @{ OS = "linux";   ARCH = "arm64"; EXT = "" },
    @{ OS = "linux";   ARCH = "amd64"; EXT = "" },
    @{ OS = "darwin";  ARCH = "arm64"; EXT = "" }
)

New-Item -ItemType Directory -Force -Path "dist" | Out-Null
$env:CGO_ENABLED = "0"

foreach ($target in $targets) {
    $out = "dist/patchpulse-$($target.OS)-$($target.ARCH)$($target.EXT)"
    Write-Host "Building $out ($($target.OS)/$($target.ARCH))..."
    $env:GOOS = $target.OS
    $env:GOARCH = $target.ARCH
    go build -ldflags "-s -w" -o $out ./cmd/patchpulse
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Failed to build for $($target.OS)/$($target.ARCH)"
        exit 1
    }
}

Write-Host "Cross-compilation completed successfully in ./dist/"
