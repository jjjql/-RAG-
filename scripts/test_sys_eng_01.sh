#!/usr/bin/env bash
# SYS-ENG-01：超时 + 熔断 / 降级 — 集中跑相关 Go 单测（无外部 Qdrant/Redis 依赖的包内测试）。
set -euo pipefail
ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
echo "==> SYS-ENG-01: go test (circuitbreaker, embedding, vector, downstream)"
go test ./internal/circuitbreaker/... ./internal/embedding/... ./internal/vector/... ./internal/downstream/... -count=1
echo "==> SYS-ENG-01: OK"
