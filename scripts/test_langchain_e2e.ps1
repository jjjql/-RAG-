# LangChain MOCK 容器 + 网关：规则未命中走 langchain_http（无 LLM）
# 用法：在仓库根目录  powershell -ExecutionPolicy Bypass -File scripts/test_langchain_e2e.ps1
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root

Write-Host "== 启动 compose（redis + gateway + langchain-mock）..."
docker compose -f docker-compose.yml -f docker-compose.langchain.yml up -d --build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$base = "http://127.0.0.1:8080"
$mock = "http://127.0.0.1:1989"

Write-Host "== 等待网关健康..."
$ok = $false
for ($i = 0; $i -lt 90; $i++) {
  try {
    $r = Invoke-WebRequest -Uri "$base/v1/health" -UseBasicParsing -TimeoutSec 2
    if ($r.StatusCode -eq 200) { $ok = $true; break }
  } catch {}
  Start-Sleep -Seconds 1
}
if (-not $ok) {
  Write-Host "[fail] 网关不可达: $base/v1/health"
  exit 1
}

Write-Host "== 探测 LangChain MOCK /health..."
try {
  $h = Invoke-WebRequest -Uri "$mock/health" -UseBasicParsing -TimeoutSec 5
  Write-Host $h.Content
} catch {
  Write-Host "[warn] MOCK /health: $_"
}

$uid = [guid]::NewGuid().ToString()
$body = @{
  query     = "langchain-e2e-no-rule-$uid"
  sessionId = "sess-e2e-$uid"
} | ConvertTo-Json -Compress

Write-Host "== POST /v1/qa（应走 LangChain MOCK，SSE source=rag）..."
$out = & curl.exe -sS -N --max-time 45 -X POST "$base/v1/qa" `
  -H "Content-Type: application/json" `
  -H "Accept: text/event-stream" `
  -d $body 2>&1

$text = $out -join "`n"
if ($text -notmatch "rag") {
  Write-Host "[fail] 未找到 source=rag 或响应异常:"
  Write-Host $text
  exit 1
}
if ($text -notmatch "MOCK LangChain") {
  Write-Host "[fail] 未找到 MOCK LangChain 应答片段:"
  Write-Host $text
  exit 1
}

Write-Host "[ok] LangChain MOCK 全链路通过（规则未命中 -> langchain_http）。"
Write-Host "（停止栈：docker compose -f docker-compose.yml -f docker-compose.langchain.yml down）"
exit 0
