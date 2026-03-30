# FR-U01 + 南向 LangChain — 架构追溯（@Architect）

> **PRD**：`specs/requirements.md` **FR-U01**（提问、可选上下文、错误清晰、正常必有回答）。  
> **南向锁定**：智能问答由 **LangChain HTTP 服务**承担，契约见 **`contracts/HTTP_LANGCHAIN_DOWNSTREAM.md`（v0.1）**。  
> **北向**：`POST /v1/qa`（SSE）不变；`sessionId` **透传**至南向 JSON。

## 验收对照（功能）

| FR-U01 要点 | 实现抓手 |
|-------------|----------|
| 用户提交自然语言 | 北向 `query` → 南向 `query` |
| 可选上下文字段 | 北向 `sessionId` → 南向 `sessionId`（多轮/Memory） |
| 必有回答（正常） | 南向 200 且 `answer` 非空 → 网关 `delta`+`done(rag)` |
| 错误可理解 | 网关 SSE `error` + HTTP 4xx/5xx / 超时映射中文 `message` |
| 可选解释 | 南向 `explanation` → 网关拼接进可见文本（见契约 §6） |

## 相关代码

- `internal/downstream`：`langchain_http`、`mock`  
- `internal/northbound/http/qa.go`：`AnswerInput` 含 `SessionID`  
- `config.yaml`：`downstream.mode: langchain_http` + `http_base_url` / `http_path`

## 联调

1. 启动 LangChain FastAPI，实现 `POST {http_path}`，符合 **HTTP_LANGCHAIN_DOWNSTREAM.md**。  
2. 网关 `downstream.enabled: true`、`mode: langchain_http`、`http_base_url` 指向该服务。  
3. `POST /v1/qa` 带 `sessionId` 与随机 `query`，期望 SSE `rag` 与答案正文。

### Docker：LangChain 契约 MOCK 容器（无 LLM）

- **文档**：`test/e2e/LANGCHAIN_DOCKER_MOCK.md`  
- **Compose**：`docker compose -f docker-compose.yml -f docker-compose.langchain.yml up -d --build`  
- **脚本**：`scripts/test_langchain_e2e.ps1`（Windows）或 `scripts/test_langchain_e2e.sh`（Bash）  
- **镜像源码**：`langchain_mock/`（FastAPI，仅返回固定 JSON，不调用模型）

---

**【开发需求已更新，请 @Dev_Go 按 `HTTP_LANGCHAIN_DOWNSTREAM.md` 完成联调与单测；LangChain 侧请 @Dev_Python 或业务服务按同契约实现】**
