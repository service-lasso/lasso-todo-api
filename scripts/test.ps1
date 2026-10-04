$ErrorActionPreference = 'Stop'
node scripts/test-service.mjs
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
