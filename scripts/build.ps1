param(
  [string]$OutputDir = "dist",
  [string]$Name = "gs",
  [switch]$Clean,
  [switch]$Race,
  [switch]$TrimPath
)

$ErrorActionPreference = "Stop"

$repoRoot = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$repoRootFull = [System.IO.Path]::GetFullPath($repoRoot)

if ([System.IO.Path]::IsPathRooted($OutputDir)) {
  $outDir = [System.IO.Path]::GetFullPath($OutputDir)
} else {
  $outDir = [System.IO.Path]::GetFullPath((Join-Path $repoRootFull $OutputDir))
}

if ($Clean -and (Test-Path -LiteralPath $outDir)) {
  $separator = [System.IO.Path]::DirectorySeparatorChar
  $insideRepo = $outDir.StartsWith($repoRootFull + $separator, [System.StringComparison]::OrdinalIgnoreCase)
  if (-not $insideRepo) {
    throw "Refusing to clean output directory outside repository: $outDir"
  }
  Remove-Item -LiteralPath $outDir -Recurse -Force
}

New-Item -ItemType Directory -Path $outDir -Force | Out-Null

$exeName = $Name
if ([System.IO.Path]::DirectorySeparatorChar -eq "\" -and -not $exeName.EndsWith(".exe", [System.StringComparison]::OrdinalIgnoreCase)) {
  $exeName = "$exeName.exe"
}

$outPath = Join-Path $outDir $exeName
$buildArgs = @("build")

if ($Race) {
  $buildArgs += "-race"
}

if ($TrimPath) {
  $buildArgs += "-trimpath"
}

$buildArgs += @("-ldflags", "-s -w")
$buildArgs += @("-o", $outPath, "./cmd/gs")

Push-Location $repoRootFull
try {
  & go @buildArgs
  if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
  }
} finally {
  Pop-Location
}

Write-Host "Built $outPath"
