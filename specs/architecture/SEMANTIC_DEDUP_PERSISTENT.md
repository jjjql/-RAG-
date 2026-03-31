# 跨进程持久化语义去重（Qdrant 写回）（@Architect）

> **对应 PRD**：`specs/requirements.md` **FR-U04**。  
> **与「进行中合并」关系**：**`COALESCE_DESIGN.md`** 描述的是 **RAG 调用合并**（内存/Redis 锁）；本文描述 **跨重启、跨网关实例** 的 **向量库持久命中**，二者可同时开启。

## 1. 目标

- 用户问题经 **Embedding** 向量化后，在 **Qdrant** 中检索与历史 **RAG 成功应答** 足够相近的条目（余弦相似度由 Qdrant `score` 与配置阈值体现）。  
- **命中**：不调用外部 RAG，直接返回已持久化答案；SSE **`done.source`=`semantic_dedup`**（与管理员预置 L3 的 **`semantic_cache`** 区分）。  
- **未命中且 RAG 成功**：将 **(向量, 答案, 元数据)** **Upsert** 至 Qdrant，供后续请求复用。

## 2. 行为（实现）

1. **前置**：`embedding.enabled=true`，`vector.enabled=true`，`vector.mode=qdrant`，且 **`vector.semantic_dedup.enabled=true`**。否则启动校验失败或功能关闭。  
2. **读路径**（在精确/正则之后）：  
   - 先执行既有 **L3** `Store.Search`（主集合 `vector.qdrant.collection`）。  
     - 若命中且 `payload.source=rag_writeback` → **`HitKind=dedup`** → SSE **`semantic_dedup`**。  
     - 若命中且无写回标记 → **`semantic_cache`**（与旧行为一致）。  
   - 若配置 **`semantic_dedup.collection` 非空且与主集合不同**：主 L3 **未命中** 后，再对 **去重专用集合** 执行一次 `Search`；命中一律视为 **`semantic_dedup`**。  
3. **写路径**：`POST /v1/qa` 在 **`source=rag`** 成功返回正文后，**异步** `PUT .../collections/{coll}/points?wait=true` 写入一点：`payload.text`（答案）、`payload.source=rag_writeback`、`query`、`trace_id`、`created_at`。  
4. **失败策略**：写回失败 **仅打日志**，**不改变** 已成功下发的 SSE。检索失败在 **独立集合** 模式下可降级 RAG（与 L3 一致）。

## 3. 配置（`config.yaml`）

| 键 | 含义 |
|----|------|
| `vector.semantic_dedup.enabled` | 是否启用持久化去重 |
| `vector.semantic_dedup.collection` | 空：与 `qdrant.collection` **共用**（依赖 `payload.source` 区分）；非空且≠主集合：**二次检索 + 写回目标** |
| `vector.semantic_dedup.score_threshold` | `0` 表示沿用 `vector.score_threshold` |

环境变量：`GATEWAY_VECTOR_SEMANTIC_DEDUP_ENABLED`、`GATEWAY_VECTOR_SEMANTIC_DEDUP_COLLECTION`、`GATEWAY_VECTOR_SEMANTIC_DEDUP_SCORE_THRESHOLD`。

## 4. 运维与契约

- **集合维度**：须与侧车 **Embedding 模型维度一致**，否则 Qdrant 拒绝写入/检索。  
- **数据治理**：写回会持续增长点数；需运维侧 **TTL/淘汰/独立集合容量** 策略（本期不内建自动淘汰）。  
- **北向契约**：`interface/openapi.yaml` **`AnswerSource.semantic_dedup`**（**0.2.4+**）。

## 5. 验收

- 单测：`internal/vector/qdrant_test.go`（`HitKind`、`UpsertWriteAnswer`）。  
- 集成：需可用 Qdrant + 侧车；见 **`test/e2e/SEMANTIC_DEDUP_DOCKER.md`**（步骤级）。  
- 指标：`gateway_qa_completed_total{source="semantic_dedup"}`；Histogram **`vector_dedup`**（独立集合二次检索）。

| 日期 | 变更 |
|------|------|
| 2026-03-31 | 首版：FR-U04 对齐实现与配置说明。 |
