# E2E：持久化语义去重（FR-U04，可选）

> **前置**：Qdrant 已创建集合，**向量维度与侧车 Embedding 一致**；网关 `embedding.enabled=true`、`vector.enabled=true`、`vector.mode=qdrant`、`vector.semantic_dedup.enabled=true`。  
> **目的**：验证 **首次 RAG** 后 **第二次语义相近问题** 走 **`semantic_dedup`** 且 **不重复调用**下游（可通过下游访问日志或 Mock 计数确认）。

## 步骤（示意）

1. 启动 **Redis + 网关 + ai_service + Qdrant**，加载上述配置。  
2. `POST /v1/qa`（`Accept: text/event-stream`）发送问题 **A**，确保 `done.source=rag`。  
3. 等待写回完成（可短暂 sleep 或轮询 Qdrant `count`）。  
4. 再次 `POST /v1/qa` 发送 **与 A 语义极相近、字面不同** 的问题 **B**（阈值依赖 `score_threshold`）。  
5. 断言：`done.source` 为 **`semantic_dedup`**（共用主集合时）或 **`semantic_dedup`**（独立集合二次检索命中）。  

## QA 签核

- 通过：**【测试通过】**（注明环境：Qdrant 版本、集合名、阈值）。  
- 不通过：**【测试不通过，请 @Dev_Go 和 @Dev_Python 一起分析问题并修复 bug】**。
