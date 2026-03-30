# 北向 HTTP 调用范例

> 契约来源：`openapi.yaml`。以下假设网关基址为 `http://localhost:8080`（请按实际部署替换 `BASE`）。  
> **用户问答**一节为 **SSE**；**管理端规则（精确/正则）** 与 **健康检查** 为 **JSON**。  
> 实现尚未就绪时，这些命令仅作**契约对齐**与联调参考。

## 环境变量（可选）

```bash
export BASE=http://localhost:8080
export TRACE=550e8400-e29b-41d4-a716-446655440000   # 可选，不传则服务端生成
```

> **管理端（OpenAPI 0.2.3+）**：示例**不携带** `Authorization`；生产环境请用网络隔离或前置网关保护 `/v1/admin/**`。

---

## 1. 健康检查

```bash
curl -sS -D - "${BASE}/v1/health" \
  -H "Accept: application/json"
```

**期望响应示例**（200）：

```json
{"status":"ok"}
```

响应头中可能包含 `X-Trace-Id`。

---

## 2. 用户提问 `POST /v1/qa`（**SSE 流式**）

请求体仍为 **JSON**；**响应**为 **`text/event-stream`**（Server-Sent Events）。  
`curl` 需 **`--no-buffer`（或 `-N`）**，否则可能等连接结束才一次性打印。

**最简**（仅必填字段 `query`）：

```bash
curl -sS -N -D - -X POST "${BASE}/v1/qa" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -d '{"query":"今天天气怎么样？"}'
```

**带可选作用域、会话、追踪 ID**：

```bash
curl -sS -N -D - -X POST "${BASE}/v1/qa" \
  -H "Content-Type: application/json" \
  -H "Accept: text/event-stream" \
  -H "X-Trace-Id: ${TRACE}" \
  -d '{
    "query": "如何重置密码？",
    "sessionId": "sess-001",
    "scope": "tenant-a"
  }'
```

**成功时响应体形态**（200，`Content-Type: text/event-stream`；以下为 SSE 文本示例）：

```text
event: meta
data: {"traceId":"550e8400-e29b-41d4-a716-446655440000"}

event: delta
data: {"text":"请点击"}

event: delta
data: {"text":"登录页「忘记密码」。"}

event: done
data: {"source":"rule_exact","ruleId":"123e4567-e89b-12d3-a456-426614174000","traceId":"550e8400-e29b-41d4-a716-446655440000"}
```

- 将所有 `delta` 的 `data.text` **按顺序拼接** 即完整可见答案。  
- `done` 的 `data.source`：`rule_exact` | `rule_regex` | `semantic_cache` | `rag`。  
- 流中也可能出现 `event: error`（`data` 为 `ErrorBody` 形状），随后连接结束。

若服务端校验 `Accept` 且未携带 `text/event-stream`，可能返回 **406** + JSON 错误体。

---

## 3. 管理端：精确规则

### 3.1 创建 `POST /v1/admin/rules/exact`

```bash
curl -sS -D - -X POST "${BASE}/v1/admin/rules/exact" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -H "X-Trace-Id: ${TRACE}" \
  -d '{
    "scope": "tenant-a",
    "key": "人工客服",
    "dat": "您好，人工客服时间为 9:00-18:00。"
  }'
```

**201 响应体示例**：

```json
{
  "id": "123e4567-e89b-12d3-a456-426614174000",
  "scope": "tenant-a",
  "key": "人工客服",
  "dat": "您好，人工客服时间为 9:00-18:00。",
  "createdAt": "2026-03-26T10:00:00Z",
  "updatedAt": "2026-03-26T10:00:00Z"
}
```

### 3.2 列表 `GET /v1/admin/rules/exact`

```bash
curl -sS -D - "${BASE}/v1/admin/rules/exact?page=1&pageSize=20&scope=tenant-a" \
  -H "Accept: application/json"
```

### 3.3 获取详情 `GET /v1/admin/rules/exact/{ruleId}`

```bash
RULE_ID=123e4567-e89b-12d3-a456-426614174000
curl -sS -D - "${BASE}/v1/admin/rules/exact/${RULE_ID}" \
  -H "Accept: application/json"
```

### 3.4 部分更新 `PATCH /v1/admin/rules/exact/{ruleId}`

```bash
curl -sS -D - -X PATCH "${BASE}/v1/admin/rules/exact/${RULE_ID}" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{"dat": "更新后的应答内容"}'
```

### 3.5 删除 `DELETE /v1/admin/rules/exact/{ruleId}`

```bash
curl -sS -D - -X DELETE "${BASE}/v1/admin/rules/exact/${RULE_ID}" \
  -H "Accept: application/json"
```

（成功时可能为 **204 No Content**，无响应体。）

---

## 4. 管理端：正则规则

### 4.1 创建 `POST /v1/admin/rules/regex`

```bash
curl -sS -D - -X POST "${BASE}/v1/admin/rules/regex" \
  -H "Content-Type: application/json" \
  -H "Accept: application/json" \
  -d '{
    "scope": "tenant-a",
    "pattern": "^订单(\\d+)$",
    "dat": "您查询的订单号为 $1，正在为您查询…",
    "priority": 10
  }'
```

### 4.2 列表 / 详情 / 更新 / 删除

路径与精确规则对称，将路径中的 `exact` 换为 `regex` 即可，例如：

```bash
curl -sS "${BASE}/v1/admin/rules/regex?page=1&pageSize=20" \
  -H "Accept: application/json"
```

---

## 5. Windows PowerShell 示例（用户提问，SSE）

`Invoke-RestMethod` 对流式不友好，可用 **.NET `HttpClient`** 自行读流，或继续用 **curl.exe**（Windows 10+ 自带）：

```powershell
$base = "http://localhost:8080"
curl.exe -sS -N -X POST "$base/v1/qa" `
  -H "Content-Type: application/json" `
  -H "Accept: text/event-stream" `
  -d '{\"query\":\"你好\"}'
```

---

## 6. 错误响应示例（契约）

HTTP 4xx/5xx 时，响应体形态参考：

```json
{
  "code": "INVALID_QUERY",
  "message": "query 不能为空",
  "traceId": "550e8400-e29b-41d4-a716-446655440000"
}
```

具体 `code` 枚举以实现与产品约定为准。
