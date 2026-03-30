# SYS-FUNC-02 / FR-A02（Docker / Windows）验收步骤

> 对应 PRD：**FR-A02**（正则规则库）+ **FR-U02**（规则极速应答-正则）+ **FR-A03**（变更生效，与 Pub/Sub 热更新一致）+ **FR-A01**（精确优先，见 §5）。  
> 架构说明：`specs/architecture/FR_A02_DESIGN.md`。  
> 依赖：`docker compose` 启动 **redis + gateway**（仓库根目录 `docker-compose.yml`）。

## 1. 启动

```powershell
cd D:\study\rag\-RAG-
docker compose up -d
```

等待 `redis` **healthy** 且 `gateway` 为 `Up`。

## 2. 创建正则规则（管理 API）

**本期**：管理 API **不要求** `Authorization`；生产请用网络边界保护管理面。

```powershell
curl.exe -sS -D - -X POST "http://127.0.0.1:8080/v1/admin/rules/regex" `
  -H "Content-Type: application/json" `
  --data-binary "@test/e2e/body_regex_create.json"
```

期望：**201**，响应体含 `id`、`pattern`、`dat`、`priority`。

## 2.1 非法正则（须拒绝）

```powershell
curl.exe -sS -D - -X POST "http://127.0.0.1:8080/v1/admin/rules/regex" `
  -H "Content-Type: application/json" `
  --data-binary "@test/e2e/body_regex_invalid.json"
```

期望：**400**，响应体为 JSON，`message` 含非法原因（与实现一致即可）。

## 3. 用户问答（SSE，仅 `query`，不走精确键）

未传 `key` 时跳过精确匹配，进入正则；`query` 须被上一步 `pattern` 命中。

```powershell
curl.exe -sS -N -D - -X POST "http://127.0.0.1:8080/v1/qa" `
  -H "Content-Type: application/json" `
  -H "Accept: text/event-stream" `
  --data-binary "@test/e2e/body_qa_regex.json"
```

期望：**200**，`Content-Type: text/event-stream`，依次出现：

- `event: meta`
- `event: delta`，`data.text` 与创建规则时的 **dat**（`来自正则规则`）一致
- `event: done`，`source`=`rule_regex`，`ruleId` 与 §2 创建返回 **id** 一致

## 4. 列表 / 详情 / 更新 / 删除（可选 smoke）

将 §2 响应中的 `id` 代入 `RULE_ID`：

```powershell
$RULE_ID="<粘贴 uuid>"
curl.exe -sS "http://127.0.0.1:8080/v1/admin/rules/regex?page=1&pageSize=20"
curl.exe -sS "http://127.0.0.1:8080/v1/admin/rules/regex/$RULE_ID"
curl.exe -sS -X PATCH "http://127.0.0.1:8080/v1/admin/rules/regex/$RULE_ID" `
  -H "Content-Type: application/json" `
  -d "{\"dat\":\"更新后DAT\"}"
curl.exe -sS -D - -X DELETE "http://127.0.0.1:8080/v1/admin/rules/regex/$RULE_ID"
```

删除成功后对同一 `query` 再调 `/v1/qa` 应不再命中该正则（若亦无其他规则，将进入后续 embed/占位路径）。

## 5. 精确优先（FR-A01 ∩ FR-A02，可选）

1. 创建正则：`pattern` 为 `^hello$`，`dat`=`from-regex`，`priority`=10。  
2. 创建精确：`key`=`hello`，`dat`=`from-exact`（`body_exact_create.json` 可改为 `hello` / `from-exact`）。  
3. `POST /v1/qa` 请求体：`{"query":"hello","key":"hello"}`（须带 `Accept: text/event-stream`）。

期望：SSE 的 `done.source`=`rule_exact`，`delta` 文本为 **`from-exact`**（非 `from-regex`）。

## 6. 停止

```powershell
docker compose down
```

## QA 结论

- 建议记录：§2 **201**、§2.1 **400**、§3 **SSE `rule_regex`**、（可选）§5 **`rule_exact`** 是否通过。
