#!/usr/bin/env bash
# 按 SYS_ACCEPTANCE_PIPELINE 串行预检：SYS-FUNC-01 → 02 → 03 → 05，并探测 /metrics（SYS-ENG-02/03）。
# 用法：在仓库根目录执行 bash scripts/test_integration.sh
# 环境变量：GATEWAY_URL（默认 http://127.0.0.1:8080）；若网关未就绪且本机有 docker，则尝试 docker compose up -d --build。
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
BASE="${GATEWAY_URL:-http://127.0.0.1:8080}"

wait_health() {
  local i=0
  while [ "$i" -lt 120 ]; do
    if curl -sf "$BASE/v1/health" >/dev/null 2>&1; then
      return 0
    fi
    i=$((i + 1))
    sleep 1
  done
  return 1
}

if ! curl -sf "$BASE/v1/health" >/dev/null 2>&1; then
  if command -v docker >/dev/null 2>&1; then
    echo "[init] 启动 docker compose（构建可能较久）..."
    docker compose -f "$ROOT/docker-compose.yml" up -d --build
    wait_health || {
      echo "[fail] 等待 $BASE/v1/health 超时"
      exit 1
    }
  else
    echo "[fail] 网关不可达且无 docker：请先启动网关或设置 GATEWAY_URL"
    exit 1
  fi
fi

echo "== SYS-FUNC-01 精确规则 + SSE + 409 冲突"
KEY="itest-exact-${RANDOM}${RANDOM}"
DAT="dat-exact-${RANDOM}"
RESP="$(curl -sS -X POST "$BASE/v1/admin/rules/exact" \
  -H "Content-Type: application/json" \
  -d "{\"key\":\"$KEY\",\"dat\":\"$DAT\"}")"
ID="$(echo "$RESP" | tr -d '\n' | sed -n 's/.*\"id\":\"\([^\"]*\)\".*/\1/p')"
test -n "$ID" || { echo "$RESP"; exit 1; }

SSE="$(curl -sS -N --max-time 20 -X POST "$BASE/v1/qa" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d "{\"query\":\"x\",\"key\":\"$KEY\"}" | tr -d '\r')"
echo "$SSE" | grep -q "rule_exact" || { echo "$SSE"; exit 1; }
echo "$SSE" | grep -Fq "$DAT" || { echo "$SSE"; exit 1; }

CODE="$(curl -sS -o /dev/null -w "%{http_code}" -X POST "$BASE/v1/admin/rules/exact" \
  -H "Content-Type: application/json" \
  -d "{\"key\":\"$KEY\",\"dat\":\"dup\"}")"
test "$CODE" = "409" || { echo "expect 409 got $CODE"; exit 1; }

echo "== FR-A01 精确规则 PATCH + DELETE（管理 API）"
PATCH_DAT="patched-${RANDOM}"
PR="$(curl -sS -X PATCH "$BASE/v1/admin/rules/exact/$ID" \
  -H "Content-Type: application/json" \
  -d "{\"dat\":\"$PATCH_DAT\"}")"
echo "$PR" | grep -q "\"id\":\"$ID\"" || { echo "$PR"; exit 1; }

SSE="$(curl -sS -N --max-time 20 -X POST "$BASE/v1/qa" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d "{\"query\":\"x\",\"key\":\"$KEY\"}" | tr -d '\r')"
echo "$SSE" | grep -Fq "$PATCH_DAT" || { echo "$SSE"; exit 1; }

CODE="$(curl -sS -o /dev/null -w "%{http_code}" -X DELETE "$BASE/v1/admin/rules/exact/$ID")"
test "$CODE" = "204" || { echo "expect 204 got $CODE"; exit 1; }

CODE="$(curl -sS -o /dev/null -w "%{http_code}" -X GET "$BASE/v1/admin/rules/exact/$ID")"
test "$CODE" = "404" || { echo "expect 404 got $CODE"; exit 1; }

echo "== SYS-FUNC-02 非法正则 400 + 正则 SSE"
CODE="$(curl -sS -o /dev/null -w "%{http_code}" -X POST "$BASE/v1/admin/rules/regex" \
  -H "Content-Type: application/json" \
  --data-binary "@$ROOT/test/e2e/body_regex_invalid.json")"
test "$CODE" = "400" || { echo "expect 400 got $CODE"; exit 1; }

RRESP="$(curl -sS -X POST "$BASE/v1/admin/rules/regex" \
  -H "Content-Type: application/json" \
  -d '{"pattern":"^ITORD[0-9]+$","dat":"regex-dat-ok","priority":5}')"
RID="$(echo "$RRESP" | tr -d '\n' | sed -n 's/.*\"id\":\"\([^\"]*\)\".*/\1/p')"
test -n "$RID" || { echo "$RRESP"; exit 1; }

SSE="$(curl -sS -N --max-time 20 -X POST "$BASE/v1/qa" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"query":"ITORD999"}' | tr -d '\r')"
echo "$SSE" | grep -q "rule_regex" || { echo "$SSE"; exit 1; }
echo "$SSE" | grep -q "regex-dat-ok" || { echo "$SSE"; exit 1; }

echo "== SYS-FUNC-03 未命中规则 → mock RAG（SSE source=rag）"
SSE="$(curl -sS -N --max-time 20 -X POST "$BASE/v1/qa" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"query":"no-rule-match-xyz-unique-12345"}' | tr -d '\r')"
echo "$SSE" | grep -q "\"source\":\"rag\"" || { echo "$SSE"; exit 1; }
echo "$SSE" | grep -q "Mock RAG" || { echo "$SSE"; exit 1; }

echo "== 相似请求合并 coalesce（并行同 query，Compose 须 GATEWAY_COALESCE_ENABLED=true）"
CQ="coalesce-parallel-${RANDOM}${RANDOM}"
TMP1="$(mktemp)"
TMP2="$(mktemp)"
curl -sS -N --max-time 25 -X POST "$BASE/v1/qa" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d "{\"query\":\"$CQ\"}" | tr -d '\r' >"$TMP1" &
P1=$!
curl -sS -N --max-time 25 -X POST "$BASE/v1/qa" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d "{\"query\":\"$CQ\"}" | tr -d '\r' >"$TMP2" &
P2=$!
wait "$P1" || { rm -f "$TMP1" "$TMP2"; exit 1; }
wait "$P2" || { rm -f "$TMP1" "$TMP2"; exit 1; }
grep -q "\"source\":\"rag\"" "$TMP1" || { echo "$TMP1"; rm -f "$TMP1" "$TMP2"; exit 1; }
grep -q "\"source\":\"rag\"" "$TMP2" || { echo "$TMP2"; rm -f "$TMP1" "$TMP2"; exit 1; }
grep -Fq "Mock RAG" "$TMP1" || { echo "$TMP1"; rm -f "$TMP1" "$TMP2"; exit 1; }
grep -Fq "Mock RAG" "$TMP2" || { echo "$TMP2"; rm -f "$TMP1" "$TMP2"; exit 1; }
rm -f "$TMP1" "$TMP2"

echo "== SYS-FUNC-05 规则变更后同问稳定（正则 PATCH + 重复问答）"
curl -sS -X PATCH "$BASE/v1/admin/rules/regex/$RID" \
  -H "Content-Type: application/json" \
  -d '{"dat":"regex-v2-stable"}' >/dev/null
sleep 2
j=0
while [ "$j" -lt 3 ]; do
  SSE="$(curl -sS -N --max-time 20 -X POST "$BASE/v1/qa" \
    -H "Content-Type: application/json" \
    -H "Accept: text/event-stream" \
    -d '{"query":"ITORD999"}' | tr -d '\r')"
  echo "$SSE" | grep -q "regex-v2-stable" || { echo "$SSE"; exit 1; }
  j=$((j + 1))
done

echo "== SYS-ENG-02 /metrics 可抓取"
M="$(curl -sS "$BASE/metrics")"
echo "$M" | grep -q "gateway_qa_completed_total" || { echo "$M" | head -20; exit 1; }
echo "$M" | grep -q 'phase="coalesce"' || { echo "$M" | head -40; exit 1; }

echo ""
echo "[ok] 集成预检全部通过（SYS-FUNC-01/02/03/05 + coalesce 并行 + /metrics）。正式闸门仍以 @QA【测试通过】为准，见 SYS_ACCEPTANCE_PIPELINE.md。"
