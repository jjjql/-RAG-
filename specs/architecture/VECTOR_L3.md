# 向量语义缓存（L3 / Qdrant）设计（@Architect）

> **来源**：`SYSTEM_DESIGN.md` §2.2 三级查询；PRD **NFR-P02**（统计至交下游前，含向量）。  
> **契约**：北向 `AnswerSource.semantic_cache`（`interface/openapi.yaml`）。

## 行为

1. **未命中**精确/正则后，若 **`embedding.enabled`** 且 **`vector.enabled`**：侧车 **`Embed`** 得到向量。  
2. **`vector.mode`**：  
   - **`noop`**：不检索（与未接库等价，便于默认集成）。  
   - **`qdrant`**：HTTP 调用 Qdrant **`POST /collections/{collection}/points/search`**，取 **limit=1**，`score ≥ score_threshold` 且 payload **`text`** 非空则命中。  
3. 命中：SSE **`done.source`=`semantic_cache`**（预置/管理灌库点；`payload.source` 非 `rag_writeback`）；未命中：继续 **下游 RAG**（或提示无下游）。  
4. **持久化语义去重（FR-U04）**：与 **`SEMANTIC_DEDUP_PERSISTENT.md`** 一致；主集合命中 **`payload.source=rag_writeback`** 时 SSE **`semantic_dedup`**；可选独立 `semantic_dedup.collection` 二次检索。

## 配置

见根目录 **`config.yaml`** 的 **`vector`** 段；Qdrant 地址 **`qdrant.url`**、集合名、阈值；**`vector.semantic_dedup`** 见 **`SEMANTIC_DEDUP_PERSISTENT.md`**。

## 验收

- 单测：`internal/vector`（httptest 替身 Qdrant）。  
- 默认 **Docker 集成脚本**不强制启动 Qdrant（`vector.mode: noop`）。

| 日期 | 变更 |
|---|---|
| 2026-03-30 | 首版：noop + Qdrant HTTP 最小实现。 |
| 2026-03-31 | 与 **FR-U04** 对齐：`payload.source=rag_writeback` 与 **`semantic_dedup`** 来源说明。 |
