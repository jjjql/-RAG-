# 相似请求合并（coalesce）设计（@Architect）

> **来源**：`.cursorrules` §5「类似请求拦截」；`SYSTEM_DESIGN.md` §2.2。  
> **范围**：对 **智能问答下游 `downstream.Complete`** 做合并：**`mode: local`** 时为进程内 singleflight；**`mode: redis`** 时由 **Redis** 记录进行中与结果，**多网关实例**上对同键只触发一次下游，其余实例轮询结果键直至就绪（与单机行为一致：等待首包成功后复用应答）。

## 合并键（规范化 · 文本模式）

当 **`coalesce.semantic: false`**（默认）时，与实现 `coalesce.Key` 一致：

```
key = scopeNorm + "\x00" + trimSpace(lower(query))
```

- `scopeNorm`：`strings.TrimSpace(*scope)`；未传 `scope` 时为空串。  
- `query`：规范化后参与合并（大小写不敏感、首尾空白剔除）。  
- **不包含** `sessionId`（避免同题多会话无法合并）；若未来产品要求合并包含会话，须递增契约评审。

## 语义相似合并（`coalesce.semantic: true`）

- **前提**：**`coalesce.enabled: true`** 且 **`embedding.enabled: true`**（侧车可用）；网关启动时若 `semantic=true` 但未启用 embedding 将 **Fatal**。  
- **判定**：对 **同一 `scopeNorm` 分区**（即 `coalesce.Key` 的 `\x00` 前段）内，将当前请求的 **query embedding** 与「进行中语义组」的代表向量做 **L2 归一化后的余弦相似度**；若 **≥ `coalesce.similarity_threshold`**（默认 **0.95**），则并入该组，**共用一次** `downstream.Complete`。  
- **与 Qdrant 语义缓存关系**：向量合并仅作用于 **RAG 合并**；**L3 `semantic_cache`** 仍为独立检索路径（先 embed → Qdrant → 未命中再走 RAG）。  
- **local 模式**：进程内 `SemanticLocal`（每 scope 一组 leader 列表 + `singleflight`）。  
- **redis 模式**：`rag:coalesce:sem:active:{sha256(scopeNorm)}` 记录活跃组 id；`rag:coalesce:sem:vec:{groupId}` 存 JSON 向量（TTL）；`rag:coalesce:sem:lock|res:{groupId}` 与文本 Redis 合并同理。**仅 Leader** 在完成后 `DEL vec` + `SREM active`，避免 follower 超时误删。  
- **`semantic_max_active`**：单 scope 活跃组数上限（默认 256）；超出时当次请求 **不再合并**（直接执行 `fn`），防止 `SMEMBERS` 过大。

## 实现

- 包：`internal/coalesce`；接口 **`Merger`**：`Passthrough`（关闭合并）、**`RAG`**（`golang.org/x/sync/singleflight`）、**`Redis`**（跨进程）。  
- 合并调用使用 **`context.WithTimeout(context.Background(), downstream.timeout_ms)`**，避免首请求取消拖死其余等待方。  
- **Redis 模式**（键名前缀 `rag:coalesce:`）：对规范化 key 做 **SHA256** 摘要；**`res:`** 存 JSON 结果（成功文本或错误串）、**`lock:`** 为 `SET NX` 占锁；Leader 写完结果后 `DEL` 锁；Follower **轮询 `GET res`** 直至命中或超时（超时上界与 `lock_ttl_seconds` 相关）。

## 配置（`config.yaml`）

- **`coalesce.enabled`**：是否合并。  
- **`coalesce.mode`**：`local` | `redis`。  
- **`coalesce.semantic`**：是否启用 **向量余弦** 语义合并（见上节）。  
- **`coalesce.similarity_threshold`**：余弦下限，默认 `0.95`。  
- **`coalesce.semantic_max_active`**：redis 语义模式下单 scope 活跃组上限。  
- **`coalesce.lock_ttl_seconds`**：占锁 TTL，须 **大于** `downstream.timeout_ms` 量级。  
- **`coalesce.result_ttl_seconds`**：结果键保留时间，供其它实例读取；亦形成短期「同题」缓存。  
- 环境变量：`GATEWAY_COALESCE_SEMANTIC`、`GATEWAY_COALESCE_SIMILARITY_THRESHOLD`、`GATEWAY_COALESCE_SEMANTIC_MAX_ACTIVE`。

## 验收

- 单测：`internal/coalesce/rag_test.go`、`redis_test.go`；**`merged_downstream_test.go`**；**`cosine_test.go`、`semantic_local_test.go`、`semantic_redis_test.go`**（语义合并）。  
- 集成：`scripts/test_integration.sh` **并行同 query** 段（Compose 下 **`GATEWAY_COALESCE_ENABLED=true`**，默认 `mode=local`）。  
- 可观测：`gateway_qa_phase_duration_seconds{phase="coalesce"}`（合并块内耗时，含等待与同次下游调用）。  
- 配置：`config.yaml` **`coalesce`** 段。

| 日期 | 变更 |
|---|---|
| 2026-03-30 | 首版：键规范与 RAG 合并范围。 |
| 2026-03-30 | 增补 **`mode: redis`**：多实例合并与结果键说明。 |
| 2026-03-30 | **闭环**：键公式与实现对齐；集成脚本 + `phase=coalesce` 指标。 |
| 2026-03-31 | **语义合并**：`coalesce.semantic` + 余弦阈值；`SemanticLocal` / `SemanticRedis`；`qa.go` 在 RAG 前按需 embed；见 `internal/coalesce/semantic_*.go`。 |
