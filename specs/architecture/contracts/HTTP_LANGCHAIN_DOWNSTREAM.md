# 南向 HTTP 契约：网关 ↔ LangChain 智能问答（架构 v0.1）

> **维护**：@Architect  
> **实现**：@Dev_Go（`internal/downstream` **`langchain_http`**）、LangChain 侧由业务以 **FastAPI / LangServe / 自定义 HTTP** 暴露（@Dev_Python 或独立服务）。  
> **北向**：仍以 **`interface/openapi.yaml`** 为准；本文件仅约束**网关进程之外**、**智能问答（`done.source=rag`）** 的 HTTP JSON 载荷。  
> **产品映射**：**FR-U01**（提问、可选上下文、错误可读）、**FR-U03**（未命中规则走智能问答）；**不包含**外部模型推理耗时是否计入 NFR（见 `SYSTEM_DESIGN.md` §2.3）。

---

## 1. 集成形态（锁定）

| 项 | 约定 |
|---|---|
| **南向技术栈** | **LangChain** 编排的 RAG/链路由**独立 HTTP 服务**承载（推荐 **Python + FastAPI**）；网关**不内嵌** Python 运行时。 |
| **调用方向** | **仅网关 → LangChain**（同步请求 / 单响应 JSON）；LangChain **不得**回调网关北向用户接口。 |
| **传输** | **HTTPS 推荐**（生产）；开发可用 `http://127.0.0.1:端口`。 |
| **超时** | 由网关 **`downstream.timeout_ms`** 控制；LangChain 侧应尽早失败并返回 JSON 错误体（见 §5）。 |

---

## 2. 端点与方法

- **Method**：`POST`  
- **Path**：默认 **`/v1/rag/invoke`**（可在 `config.yaml` 的 `downstream.http_path` 覆盖；**须以 `/` 开头**）。  
- **完整 URL**：`{http_base_url}{http_path}`，例如 `http://langchain:1989/v1/rag/invoke`。

---

## 3. 请求（JSON，`Content-Type: application/json`）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `query` | string | 是 | 用户自然语言问题（与北向 `QARequest.query` 一致） |
| `sessionId` | string | 否 | 北向透传，供 LangChain **Memory / 多轮** 使用（FR-U01 可选上下文字段） |
| `traceId` | string | 否 | 与北向 `X-Trace-Id` 对齐，便于 LangChain 日志关联 |

**示例：**

```json
{
  "query": "公司年假政策是什么？",
  "sessionId": "sess-001",
  "traceId": "550e8400-e29b-41d4-a716-446655440000"
}
```

### 3.1 可选鉴权（部署约定）

- 若配置 **`downstream.http_api_key_header`**（如 `X-API-Key`）且提供密钥（建议环境变量 **`GATEWAY_DOWNSTREAM_HTTP_API_KEY`**），网关将在请求中附带该头；**不在 PRD 强制**，由部署安全模型决定。

---

## 4. 成功响应（HTTP 200，`application/json`）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `answer` | string | 是 | 面向用户的可见答案正文（可为多段 Markdown/纯文本） |
| `explanation` | string | 否 | **FR-U01** 可选「解释」；网关将其**追加**到 `answer` 之后展示（见 §6） |

**示例：**

```json
{
  "answer": "根据知识库，年假为 10 个工作日。",
  "explanation": "来源：员工手册 §3.2"
}
```

---

## 5. 失败响应

| HTTP | 约定 |
|------|------|
| **4xx / 5xx** | 响应体**建议**为 JSON：`{ "error": "机器可读码", "message": "人类可读说明" }`；字段名允许扩展，网关将 **`message` 或 body 字符串** 作为北向 SSE `error` 的说明来源之一。 |
| **超时 / 连接失败** | 由网关映射为 SSE **`event:error`**（中文 `code`/`message` 见实现），满足 **NFR-G01** 子集。 |

---

## 6. 网关北向行为（摘要）

- 未命中规则且走本下游时：将 LangChain 返回的 **`answer`（+ 可选 `explanation` 拼接）** 作为 **一条或多条** `event:delta` 的 `text`（**v0.1 实现为单条 `delta`**），随后 **`event:done`** 且 **`source`=`rag`**。  
- **`sessionId` / `traceId`** 必须原样传入请求 JSON（若北向未传则省略字段）。

---

## 7. LangChain 侧实现提示（非规范）

- 使用 **FastAPI** 声明 Pydantic 模型与上述字段对齐；链路由 **LangChain `Runnable` / LCEL** 组装；向量库与模型选型由业务决定，**不在本契约冻结**。  
- 联调清单：健康检查、空 `query` 校验、超时行为、与网关 `traceId` 对账日志。

---

## 8. 修订记录

| 版本 | 日期 | 说明 |
|------|------|------|
| v0.1 | 2026-03-27 | 首版：**南向锁定 LangChain（HTTP JSON）**；默认路径 `/v1/rag/invoke`；请求/响应字段冻结为网关 v0.1 实现依据。 |
