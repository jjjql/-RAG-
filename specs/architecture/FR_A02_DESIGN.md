# FR-A02 设计说明（正则规则 / DAT）— 架构落盘

> **来源**：`specs/requirements.md` §3.1 **FR-A02**（正则表达式 / DAT 管理）及 **FR-U02**（规则极速应答）中的正则部分。  
> **读者**：@Architect / @Dev_Go / @QA。  
> **北向契约**：`interface/openapi.yaml`（`AdminRegex`、`RegexRule*`、`POST /v1/qa` SSE）。  
> **正则引擎**：与 **Go `regexp`（RE2 语义）** 一致，与 `interface/PM_PRODUCT_REVIEW.md` §2.3 对齐。

---

## 1. 需求映射（FR-A02 → 架构）

| PRD 要点 | 架构结论 | 实现位置（Go） |
|---|---|---|
| 非法正则必须拒绝保存 | 创建/更新前 **`regexp.Compile`**，失败则 **400** + 可读 `message` | `internal/rules/redis_regex_store.go`（`compilePattern` / `ErrInvalidRegex`） |
| 冲突策略写死 | **`priority` 降序**；同 `priority` 按 **`updatedAt` 新者优先**；遍历取**首个** `MatchString(query)` | `internal/rules/regex_memory_index.go`（`ReplaceAll` 内排序 + `Match`） |
| 可选作用域 | 与精确规则一致：`ScopeKey(nil/"")` 为全局桶；用户 `scope` 与规则 `scope` **规范化后须相同**才参与匹配 | `internal/rules/regex_coordinator.go` `MatchRegex` |
| 增删改查 | 管理端 **`/v1/admin/rules/regex`** 集合 + **`/v1/admin/rules/regex/{ruleId}`** 单条 | `internal/northbound/http/admin_regex.go` |
| FR-A03 生效时效 | 写 Redis 后 **Pub/Sub** `rag:regex:changed` → 全实例 **`Reload`** 重建内存索引（与精确规则同构） | `cmd/gateway/main.go` `subscribeRegexReload` |

**与 FR-A01 的跨类型裁决**（精确优先）：用户请求若带非空 **`key`** 且精确命中，则 **不再尝试正则**；见 `internal/northbound/http/qa.go` 编排顺序。

---

## 2. Redis 与消息模型（摘要）

| 键 / 频道 | 用途 |
|---|---|
| `rag:regex:rule:{uuid}` | 单条规则 JSON |
| `rag:regex:ids` | 规则 ID 集合 |
| `rag:regex:changed` | 变更通知（payload 实现可忽略，订阅侧执行全量 `Reload`） |

---

## 3. 可测试性

| 层级 | 说明 |
|---|---|
| **单测** | `internal/rules/regex_memory_index_test.go`（优先级与同 priority 新者）；`redis_regex_store_test.go`（非法 pattern、CRUD） |
| **系统级** | **SYS-FUNC-02**（`SYSTEM_DESIGN.md` §3.1）；Docker 手工验收 **`test/e2e/SYS_FUNC_02_DOCKER.md`** |

---

## 4. 与自动化流水线衔接

- **架构**：本文 + `SYSTEM_DESIGN.md` §1.1 中 **FR-A02 / SYS-FUNC-02** 条目为验收事实源。  
- **开发**：实现须与 **`interface/openapi.yaml`** 一致；若契约变更须递增 `info.version` 并同步本文。  
- **QA**：按 **`test/e2e/SYS_FUNC_02_DOCKER.md`** 出结论；全链路脚本化待 `scripts/test_integration.sh` 补齐后挂钩 CI。

---

> **修订记录**  
> - **2026-03-27（@Architect）**：首版；对齐当前仓库 Go 实现与 OpenAPI **0.2.3**。
