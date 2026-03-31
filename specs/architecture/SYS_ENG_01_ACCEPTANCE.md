# SYS-ENG-01：验收矩阵（100ms 级超时 + 熔断 / 降级）

> **对应**：`SYSTEM_DESIGN.md` §3.3 **SYS-ENG-01**；实现说明见 **`SYS_ENG_01_BREAKER.md`**。  
> **一键复验**：仓库根目录执行 `bash scripts/test_sys_eng_01.sh`（或等价地 `go test` 下列包）。

## 1. 单元测试与断言

| 范围 | 包 / 文件 | 断言要点 |
|------|-----------|----------|
| 熔断器 | `internal/circuitbreaker` | 开路 / 半开 / 关闭与阈值行为 |
| 向量（Qdrant） | `internal/vector/circuit_store_test.go` | `Search` 超时（`context`）；开路时快速失败（`ErrOpen`） |
| 下游 Mock | `internal/downstream/downstream_test.go` | `Complete` 在短 `context` 下返回 **`context.DeadlineExceeded`**（`errors.Is`） |
| 下游熔断包装 | `internal/downstream/breaker_answerer_test.go` | 连续失败达阈值后 **`circuitbreaker.ErrOpen`** |
| 下游 LangChain HTTP | `internal/downstream/langchain_http_test.go` | 慢服务端 + 短 `context` → **`DeadlineExceeded` / `Canceled`** |
| 嵌入侧车 | `internal/embedding/client_test.go`、`circuit_test.go` | `TestClient_Embed_Timeout`（短 `Timeout` + 慢服务端）；`TestCircuitService_Embed_Open`（`ErrOpen`） |

## 2. 配置与故障注入（可选）

| 键 | 用途 |
|----|------|
| `embedding.timeout_ms`、`embedding.circuit_breaker.*` | UDS 嵌入超时与熔断 |
| `vector.timeout_ms`、`vector.circuit_breaker.*` | Qdrant 检索 |
| `downstream.timeout_ms`、`downstream.circuit_breaker.*` | 下游 RAG |
| `downstream.mock_delay_ms`（**仅 `mode: mock`**） | 人为拉长 Mock 应答，用于本地验证超时（**不设或 0 则不影响生产路径**） |

环境变量覆盖见 `internal/config`（如 `GATEWAY_DOWNSTREAM_MOCK_DELAY_MS`）。

## 3. E2E 说明

多实例、跨容器故障注入的 **全链路 E2E** 为可选扩展，**不阻塞** SYS-ENG-01 在代码与单测层面的收口；主闸门以 **§1 单测绿** + **`SYS_ENG_01_BREAKER.md`** 行为一致为准。

## 4. 修订记录

| 日期 | 变更 |
|------|------|
| 2026-03-30 | 首版：验收矩阵 + `scripts/test_sys_eng_01.sh` 对齐 |
