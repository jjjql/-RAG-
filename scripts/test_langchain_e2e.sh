#!/usr/bin/env bash
# LangChain MOCK 容器 + 网关 E2E（与 test_langchain_e2e.ps1 等价）
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "== docker compose up（redis + gateway + langchain-mock）"
docker compose -f docker-compose.yml -f docker-compose.langchain.yml up -d --build

BASE="${GATEWAY_URL:-http://127.0.0.1:8080}"
i=0
while [ "$i" -lt 90 ]; do
  if curl -sf "$BASE/v1/health" >/dev/null 2>&1; then
    break
  fi
  i=$((i + 1))
  sleep 1
done
if ! curl -sf "$BASE/v1/health" >/dev/null 2>&1; then
  echo "[fail] 网关不可达: $BASE/v1/health"
  exit 1
fi

UID="$(od -An -N4 -tu4 /dev/urandom 2>/dev/null | tr -d ' ' || echo $$)"
BODY="{\"query\":\"langchain-e2e-no-rule-$UID\",\"sessionId\":\"sess-e2e-$UID\"}"
SSE="$(curl -sS -N --max-time 45 -X POST "$BASE/v1/qa" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d "$BODY" | tr -d '\r')"

echo "$SSE" | grep -q "\"source\":\"rag\"" || { echo "$SSE"; exit 1; }
echo "$SSE" | grep -q "MOCK LangChain" || { echo "$SSE"; exit 1; }

echo "[ok] LangChain MOCK 全链路通过"
echo "停止: docker compose -f docker-compose.yml -f docker-compose.langchain.yml down"
