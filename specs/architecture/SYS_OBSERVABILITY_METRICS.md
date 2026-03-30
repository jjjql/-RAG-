# 可观测性指标（SYS-PERF-02 / SYS-ENG-02 锚点）

> **维护**：@Architect  
> **事实来源**：`SYSTEM_DESIGN.md` §2.3（NFR-P02 起止点与子阶段拆分意图）。

## 已实现（Prometheus）

| 指标名 | 类型 | 标签 | 含义 |
|---|---|---|---|
| `gateway_qa_completed_total` | Counter | `source` | `POST /v1/qa` 收口路径（rule_exact / rule_regex / rag / error_*） |
| `gateway_qa_phase_duration_seconds`（namespace `gateway`） | Histogram | `phase` | **`embed`**：单次 `Embed` 耗时；**`rag_prep`**：从进入 `handleQA` 到即将调用下游 `Complete` 之前的累计耗时（含规则判定与可选 embed，**不含**外部 RAG 本体） |

Histogram 桶：`0.001`～`1.0` 秒（与 §2.3「网关进程内打点」一致，供 SYS-PERF-02 后续固化 P99）。

## 待扩展（不阻塞本期闸门）

- 向量库检索子阶段、`coalesce` 等待时间 — 待 `internal/vector` 与合并模块落地后追加 `phase` 枚举。

| 日期 | 变更 |
|---|---|
| 2026-03-30 | 首版：`gateway_qa_phase_duration_seconds` 与 SYS-PERF-02 对齐说明。 |
