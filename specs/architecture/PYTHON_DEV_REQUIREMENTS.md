# Python 侧模块级开发需求（@Architect）

> **拆分说明**：本文承接 `SYSTEM_DESIGN.md` **§4** 中 **Python** 部分，供 **@Dev_Python** 估点、拆分任务与自检。  
> **总览与边界**仍以 `SYSTEM_DESIGN.md` 全文为准；若与 Go 侧 UDS 契约不一致，以双方冻结的 **`interface/` 或 `specs/architecture/contracts/`** 为准并同步修订。  
> **系统级需求开发顺序**：涉及侧车与联调时，须遵守 **`SYS_ACCEPTANCE_PIPELINE.md`**（与 Go 侧同一 SYS-* 闸门）。  
> **配套**：Go 侧见同目录 **`GO_DEV_REQUIREMENTS.md`**。  
> **UDS（Embedding）内部契约**：**`contracts/UDS_EMBEDDING.md`（v1.0 冻结）** + **`contracts/uds_embedding.schema.json`** + **`contracts/UDS_INTERNAL_ALIGNMENT.md`**（与 @Dev_Go 对齐；联调 PR 补签 §1）。

---

## 1. 范围

- **代码目录**：`ai_service/`（与 `.cursorrules` 一致）。  
- **角色**：Embedding **侧车**，经 **UDS** 与网关通信；**进程级隔离**，异常不得拖死 Go 主进程。

---

## 2. 模块级需求表（`ai_service/`）

| 模块 | 目标 | 关键需求点（可测试） | 备注 |
|---|---|---|---|
| UDS 服务进程 | Embedding 侧车 | **实现** `contracts/UDS_EMBEDDING.md`：**uint32 BE 帧 + JSON**；监听路径可配置（默认 `/tmp/rag_gateway.sock`）；支持 **embed** 与 **ping/pong** | 进程级隔离；与 Go 帧格式逐字节一致 |
| **LangChain HTTP（南向）** | 智能问答 | **实现** **`contracts/HTTP_LANGCHAIN_DOWNSTREAM.md` v0.1**：`POST /v1/rag/invoke`（路径可配置）；请求体 `query` / `sessionId` / `traceId`；响应 `answer` / 可选 `explanation`；错误 JSON | 可与 Embedding 侧车**同仓分进程**或独立部署；**不在 UDS 契约内** |
| Pydantic 模型 | 契约 | 请求/响应与 **`uds_embedding.schema.json`** 及文档 §3–§4 一致；`error.code` 枚举一致 | 与 Go 客户端对齐 |
| `embedder` | 推理 | 模型加载失败处理；批处理（可选）；推理耗时监控 | 可用小模型做 CI |
| `tests` | 质量 | `pytest` 覆盖契约与核心推理 | 引用 `.cursorrules` |

---

## 3. 质量要求（引用 `.cursorrules`）

- 使用 **Pydantic**；核心逻辑 **Type Hinting**。  
- **pytest** 覆盖 UDS 契约与核心推理路径（可与 Go 侧 Mock 客户端对照）。

---

> 修订记录：由架构师在变更侧车职责或 UDS 契约时更新本表，并在 `SYSTEM_DESIGN.md` 中保留 §4 索引指向本文。
>
> - **2026-03-27（@Architect）**：**SYS-FUNC-01**：精确命中路径**不依赖** `ai_service`；双端确认单见 **`SYS_FUNC_01_DEV_CONFIRM.md` §3**。**【请 @Dev_Python 回复「无交付项、不阻塞」暗号，供流水线与 @QA 归档】**
