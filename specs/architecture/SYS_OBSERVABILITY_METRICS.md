# 可观测性指标（SYS-PERF-02 / SYS-ENG-02 锚点）

> **维护**：@Architect  
> **事实来源**：`SYSTEM_DESIGN.md` §2.3（NFR-P02 起止点与子阶段拆分意图）。

## 已实现（Prometheus）

| 指标名 | 类型 | 标签 | 含义 |
|---|---|---|---|
| `gateway_qa_completed_total` | Counter | `source` | `POST /v1/qa` 收口路径（rule_exact / rule_regex / semantic_cache / **semantic_dedup** / rag / error_*） |
| `gateway_qa_phase_duration_seconds`（namespace `gateway`） | Histogram | `phase` | **`embed`**：单次 `Embed`；**`vector`**：L3 `Store.Search`；**`vector_dedup`**：持久化去重 **独立集合** 二次检索（FR-U04）；**`rag_prep`**：进入 `handleQA` 至 **`Coalesce.Do` 之前**（含规则与可选 embed/vector）；**`coalesce`**：**`Coalesce.Do` / `Semantic.Merge` 块**耗时（见 **`COALESCE_DESIGN.md`**） |

Histogram 桶：`0.001`～`1.0` 秒（与 §2.3「网关进程内打点」一致，供 SYS-PERF-02 后续固化 P99）。

## 待扩展

- 更细粒度的 **「纯等待」** 与 **「Leader 执行下游」** 拆分（若 SYS-PERF-02 需要）。

| 日期 | 变更 |
|---|---|
| 2026-03-30 | 首版：`gateway_qa_phase_duration_seconds` 与 SYS-PERF-02 对齐说明。 |
| 2026-03-31 | **FR-U04**：`source=semantic_dedup`；`phase=vector_dedup`。 |
