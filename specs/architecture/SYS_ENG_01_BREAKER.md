# SYS-ENG-01：熔断 / 降级（实现说明）

> **对应**：`.cursorrules` §3「跨进程调用强制设置 100ms 超时，必须具备熔断降级逻辑」与 `SYSTEM_DESIGN.md` §3.3 **SYS-ENG-01**（Go→Python UDS、Go→**Qdrant**、Go→下游）。  
> **事实来源**：`internal/circuitbreaker/`；包装 **Embedding**、**向量检索（Qdrant）**、**下游 RAG**（均见 `config.yaml`）。

## 行为（本期）

- **超时**：  
  - **Embedding**：`embedding.timeout_ms` + `context`（`internal/embedding`）。  
  - **Qdrant 检索**：`vector.timeout_ms`（默认 100）；`internal/vector.CircuitStore` 对单次 `Search` 使用 `context.WithTimeout`；`http.Client.Timeout` 与之对齐。  
  - **下游 RAG**：`downstream.timeout_ms` + `downstream.Client.Complete`。  
- **熔断**：连续失败达到 **`failure_threshold`** 次后进入 **开路** **`open_seconds` 秒**，期间 **快速失败**（`circuitbreaker.ErrOpen`），属 **降级**。  
- **恢复**：开路窗口结束后下一次调用 **试探**；成功则关闭，失败则再次开路。  
- **向量检索失败（非开路）**：`qa.go` **降级继续 RAG**（日志留痕）；**开路**返回 SSE `VECTOR_CIRCUIT_OPEN`。

## 配置键

根目录 **`config.yaml`**：

| 组件 | 路径 |
|------|------|
| 侧车 Embedding | `embedding.circuit_breaker`、`embedding.timeout_ms` |
| 向量（Qdrant） | `vector.circuit_breaker`、`vector.timeout_ms`（**`vector.enabled=true` 且 `mode=qdrant`** 时生效） |
| 下游 RAG | `downstream.circuit_breaker`、`downstream.timeout_ms` |

字段：`enabled`、`failure_threshold`、`open_seconds`。

## 验收

- **矩阵与一键脚本**：**`SYS_ENG_01_ACCEPTANCE.md`**；`bash scripts/test_sys_eng_01.sh`（`go test`：`circuitbreaker`、`embedding`、`vector`、`downstream`）。  
- **单测要点**：`internal/circuitbreaker`；`internal/vector/circuit_store_test.go`（超时、开路）；`internal/embedding`（`Embed` 超时、`CircuitService` 开路）；`internal/downstream`（Mock `Complete` 超时、`WrapAnswerer` 开路、**LangChain HTTP** `Answer` 在短 `context` 下 `DeadlineExceeded`/`Canceled`）。  
- **Mock 故障注入**：`downstream.mode: mock` 时可用 **`mock_delay_ms`**（或 `GATEWAY_DOWNSTREAM_MOCK_DELAY_MS`）拉长应答，便于本地验 **`downstream.timeout_ms`**。  
- **闸门**：本需求属 **`SYS_ACCEPTANCE_PIPELINE.md` 次序 7**；以 **验收矩阵单测绿** + 上表配置一致为代码级收口。  
- 多实例、跨容器 Qdrant 故障注入 **E2E** 仍为 **可选扩展**（不阻塞上述收口）。

| 日期 | 变更 |
|---|---|
| 2026-03-30 | 首版：embed/下游。 |
| 2026-03-30 | 增补 **Go→Qdrant**：`CircuitStore`、`vector.*` 配置与 `qa.go` L3。 |
| 2026-03-30 | 增补 **`SYS_ENG_01_ACCEPTANCE.md`**、`scripts/test_sys_eng_01.sh`、下游 **LangChain 超时单测**、**`mock_delay_ms`** 说明。 |
