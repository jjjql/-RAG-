# 系统级需求检视清单（@Reviewer）

> 配合 `SYS_ACCEPTANCE_PIPELINE.md`：**当前编号** 开发交付后，检视通过再交 @QA。  
> 逐项勾选；不通过按 `.cursorrules` 发 **【发现问题，请 @Dev_Go 重构】** 等。  
> **归档（2026-03-27）**：**SYS-FUNC-01** 已与 `scripts/test_integration.sh` §1 及 `go test ./...` 一并闭环，详见 **`SYS_ACCEPTANCE_PIPELINE.md` §3 / §5**（等同本期对 01 的 **【准予上线】**）。  
> **归档（2026-03-30）**：次序 **2～4、7～9** 已与集成脚本 / 单测及 **`SYS_ENG_01_BREAKER.md`** 对齐；**SYS-PERF-*** 仍按 **`SYS_ACCEPTANCE_PIPELINE.md` §3** 待 k6 或专项报告补签。

## 通用

- [ ] 变更与 `interface/openapi.yaml` / `SYSTEM_DESIGN.md` §3 判据一致，无静默偏离契约  
- [ ] `error` 路径不泄露敏感内部栈信息；SSE / JSON 错误模型可读  
- [ ] `context` 超时与取消在下游/侧车调用上生效（SYS-ENG-01 相关路径）

## 按编号（交付物对齐 §3）

| 编号 | 检视要点 |
|---|---|
| SYS-FUNC-01 | 精确热路径不读 Redis；409 冲突；Pub/Sub 重载无死锁 |
| SYS-FUNC-02 | 非法正则不入库；`priority` / `updatedAt` 顺序与 `FR_A02_DESIGN.md` 一致；精确优先 |
| SYS-FUNC-03 | 未命中 → RAG/mock 的 SSE `source=rag`；下游超时/失败有界；**合并**：文本键与可选 **`coalesce.semantic`（余弦阈值）** 见 **`COALESCE_DESIGN.md`** |
| FR-U04 | 持久化语义去重：`semantic_dedup` / 写回 Qdrant；**`SEMANTIC_DEDUP_PERSISTENT.md`**；OpenAPI `AnswerSource` 与 **`PM_PRODUCT_REVIEW.md` §1.4** 一致 |
| SYS-FUNC-05 | 变更后重复请求结果一致（与 E2E 步骤一致） |
| SYS-PERF-* | 压测口径文档化；脚本可复现 |
| SYS-ENG-01 | embed / **vector（Qdrant）**/ 下游：`timeout_ms` + 可选熔断；验收矩阵 **`SYS_ENG_01_ACCEPTANCE.md`**；`VECTOR_CIRCUIT_OPEN` 与降级路径见 **`SYS_ENG_01_BREAKER.md`** |
| SYS-ENG-02 | `/metrics` 暴露且指标名可检索 |
| SYS-ENG-03 | `scripts/test_integration.sh` 在 CI/文档环境下可执行 |
