# 内部契约（非北向 HTTP）

| 文件 | 说明 |
|---|---|
| **UDS_EMBEDDING.md** | Go 网关 ↔ Python 侧车：**Embedding** 的 UDS 帧格式、JSON 字段、错误码、超时与联调清单（**v1.0 冻结**，2026-03-26）。 |
| **uds_embedding.schema.json** | 上述载荷的 **JSON Schema**（Draft 2020-12），**`$id`：`uds_embedding_v1.0_frozen.json`**。 |
| **UDS_INTERNAL_ALIGNMENT.md** | @Dev_Go / @Dev_Python **内部设计对齐勾选**与 **@Architect 归档回执**（与 UDS v1.0 配套）。 |
| **HTTP_LANGCHAIN_DOWNSTREAM.md** | 网关 → **LangChain（南向 HTTP）**：智能问答 **invoke** JSON 契约（**v0.1**）；对齐 **FR-U01 / FR-U03**。 |

北向 HTTP 仍以仓库根目录 **`interface/openapi.yaml`** 为准；本目录**不替代** OpenAPI。
