# LangChain MOCK 容器全链路（规则未命中 → `langchain_http`）

> **目标**：本机 Docker 启动 **Redis + 网关 + LangChain 契约 MOCK 容器**（**不调用 LLM**），验证 **未命中精确/正则规则** 时走 **`POST {base}/v1/rag/invoke`**，北向 SSE **`done.source=rag`**。  
> **契约**：`specs/architecture/contracts/HTTP_LANGCHAIN_DOWNSTREAM.md`  
> **Compose**：`docker-compose.yml` + `docker-compose.langchain.yml`

## 1. 启动

```powershell
cd D:\study\rag\-RAG-
docker compose -f docker-compose.yml -f docker-compose.langchain.yml up -d --build
```

等待 `langchain-mock` **healthy**、`gateway` **Up**。网关通过环境变量覆盖为 **`downstream.mode=langchain_http`**，指向 **`http://langchain-mock:1989`**（见 `docker-compose.langchain.yml`）。

## 2. 用户问答（未命中规则）

请求体 **不传 `key`**，且 `query` **不匹配** 已配置正则（默认无规则时即未命中），将直达下游 MOCK。

```powershell
curl.exe -sS -N --max-time 30 -X POST "http://127.0.0.1:8080/v1/qa" `
  -H "Content-Type: application/json" `
  -H "Accept: text/event-stream" `
  -d "{\"query\":\"langchain-docker-mock-未命中规则-" + [guid]::NewGuid().ToString() + "\",\"sessionId\":\"sess-e2e-1\"}"
```

期望：**200**，SSE 中出现 **`"source":"rag"`**，`delta` 文本含 **`[MOCK LangChain 容器]`**（来自 MOCK 服务 `answer` + `explanation` 拼接）。

## 3. 停止

```powershell
docker compose -f docker-compose.yml -f docker-compose.langchain.yml down
```

## 4. 直接探测 MOCK 服务（可选）

```powershell
curl.exe -sS "http://127.0.0.1:1989/health"
curl.exe -sS -X POST "http://127.0.0.1:1989/v1/rag/invoke" -H "Content-Type: application/json" -d "{\"query\":\"ping\"}"
```
