# SYS-ENG-01：熔断 / 降级（最小实现说明）

> **对应**：`.cursorrules` §3「跨进程调用强制设置 100ms 超时，必须具备熔断降级逻辑」与 `SYSTEM_DESIGN.md` §3.3 **SYS-ENG-01**。  
> **事实来源**：实现位于 `internal/circuitbreaker/`；对 **Embedding**、**下游 RAG** 可选启用（`config.yaml`）。

## 行为（本期）

- **超时**：沿用既有 **`internal/embedding`** 与 **`downstream.Client.Complete`** 的 **context 超时**（默认 100ms 档，可配置）。  
- **熔断**：连续失败达到 **`failure_threshold`** 次后进入 **开路**状态 **`open_seconds` 秒**，期间 **快速失败**（不发起真实调用），属 **降级**。  
- **恢复**：开路窗口结束后下一次调用 **试探**；成功则关闭，失败则再次开路。

## 配置键

见根目录 **`config.yaml`**：`embedding.circuit_breaker`、`downstream.circuit_breaker`（字段：`enabled`、`failure_threshold`、`open_seconds`）。

## 验收

- 单测：`internal/circuitbreaker/breaker_test.go`。  
- 故障注入 E2E（多实例、Qdrant 等）仍可在后续迭代扩展；本期以 **代码路径 + 配置 + 单测** 作为 SYS-ENG-01 **可自动化**部分。

| 日期 | 变更 |
|---|---|
| 2026-03-30 | 首版：与实现同步，供流水线与 @Reviewer 勾选。 |
