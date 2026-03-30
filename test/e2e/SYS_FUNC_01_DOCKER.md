# SYS-FUNC-01（Docker / Windows）验收步骤

> 依赖：`docker compose` 启动 **redis + gateway**（见仓库根目录 `docker-compose.yml`）。

## 1. 启动

```powershell
cd D:\study\rag\-RAG-
docker compose up -d
```

等待 `redis` **healthy** 且 `gateway` 为 `Up`。

## 2. 创建精确规则（管理 API）

**本期**：管理 API **不要求** `Authorization` 头；生产环境请用网络边界或前置网关保护管理面。

```powershell
curl.exe -sS -D - -X POST "http://127.0.0.1:8080/v1/admin/rules/exact" `
  -H "Content-Type: application/json" `
  --data-binary "@test/e2e/body_exact_create.json"
```

期望：**201**，响应体含 `id`、`key`、`dat`。

重复提交同 KEY 应 **409** `EXACT_KEY_CONFLICT`。

## 3. 用户问答（SSE）

请求体须携带与 OpenAPI 一致的 **`key`**（与精确规则 `key` 相同）；`query` 可为自然语言，未传 `key` 时不会走精确命中。

```powershell
curl.exe -sS -N -D - -X POST "http://127.0.0.1:8080/v1/qa" `
  -H "Content-Type: application/json" `
  -H "Accept: text/event-stream" `
  --data-binary "@test/e2e/body_qa.json"
```

期望：**200**，`Content-Type: text/event-stream`，依次出现：

- `event: meta`
- `event: delta`，`data.text` 与上一步 **DAT** 一致
- `event: done`，`source`=`rule_exact`，`ruleId` 与创建返回 **id** 一致

## 4. 停止

```powershell
docker compose down
```

## QA 结论

- 最近一次自动化验证：Docker 下 **201 + SSE 命中** 已通过（与 `body_exact_create.json` / `body_qa.json` 一致）。
