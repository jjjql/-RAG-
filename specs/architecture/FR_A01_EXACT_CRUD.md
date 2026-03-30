# FR-A01：精确 KEY/DAT — 实现与验收（@Architect）

> **PRD**：`specs/requirements.md` **FR-A01**（增删改查、同 scope KEY 唯一、重复拒绝）。  
> **北向契约**：`interface/openapi.yaml` — `GET/POST /v1/admin/rules/exact`，`GET/PATCH/DELETE /v1/admin/rules/exact/{ruleId}`。

## 实现要点（Go）

| 能力 | 说明 |
|------|------|
| **PATCH** | `ExactRulePatch`：`scope` / `key` / `dat` 可选；至少一项；变更 `(scope,key)` 时重建 Redis 二级索引并检测 **409** |
| **DELETE** | 删除规则 JSON、`rag:exact:ids` 成员及对应 `rag:exact:idx:*`，**204** 无正文 |
| **热更新** | 写后 **`rag:exact:changed`** + 协调器 **`Reload`** |

## 验收

- 单测：`internal/rules/redis_store_test.go`（`TestRedisExactStore_UpdateAndDelete`）  
- 集成：`scripts/test_integration.sh` 中 **FR-A01 PATCH + DELETE** 段  

| 日期 | 变更 |
|---|---|
| 2026-03-30 | 首版：补齐 PATCH/DELETE，与 OpenAPI 对齐。 |
