# SYS-FUNC-01：双端开发确认单（@Architect 派发）

> **对应系统级需求**：`SYSTEM_DESIGN.md` **§3.1** **SYS-FUNC-01**（PRD：**FR-A01** + **FR-U02** + **FR-A03** 中与精确规则相关的可测部分）。  
> **正式闸门**：仍以 **`SYS_ACCEPTANCE_PIPELINE.md`** 为准——**@QA** 对次序 **1** 发出 **【测试通过】** 后，该项才视为**已验收**；本文仅固化 **@Dev_Go / @Dev_Python** 的**职责边界与自检结论**。

---

## 1. 架构结论（是否「已完成」）

| 维度 | 结论 |
|---|---|
| **实现主体** | **Go 网关 + Redis**；`docker-compose.yml` 中 **SYS-FUNC-01 最小栈为 `redis` + `gateway`****，不含 `ai_service`**。 |
| **用户路径（精确命中）** | 命中 **内存精确索引**，问答路径**不调用** Python UDS；与 **FR-U02**「规则极速」一致。 |
| **已有自动化** | `scripts/test_integration.sh` **§1 段**：创建精确规则 → SSE 含 `rule_exact` 与 DAT → 重复 KEY **409**。 |
| **@Dev_Python** | 对 **SYS-FUNC-01** **无功能代码交付义务**；仅需书面确认 **不阻塞** 本条验收（见 §3）。 |

---

## 2. @Dev_Go 确认清单（须逐项自检并回复暗号）

对照 **`test/e2e/SYS_FUNC_01_DOCKER.md`** 与 **`SYSTEM_DESIGN.md` §3.1** 行 **SYS-FUNC-01**：

1. **管理 API**：`POST /v1/admin/rules/exact` 创建成功；同一 KEY **409**（`EXACT_KEY_CONFLICT` 或等价语义与 OpenAPI 一致）。  
2. **用户 SSE**：`POST /v1/qa` + `Accept: text/event-stream`，传与规则一致的 **`key`**；响应含 **`event: delta`**（内容与 DAT 一致）、**`event: done`** 且 **`source`=`rule_exact`**。  
3. **持久化 + 热路径**：规则 **Redis 持久化**；问答命中 **不走 Redis 读规则**（内存索引）；写规则后 **本地内存重载**（单实例）；多实例场景下 **Pub/Sub** 行为与 **`SYSTEM_DESIGN.md` §1.1** 描述一致（若尚未 E2E 多实例，请在回执中**明确**「仅单实例已测 / 多实例待 QA」）。  
4. **单元测试**：相关包 **`go test` 绿**。

**回执暗号（开发交付检视前）**：在协作频道回复 **【Go 侧 SYS-FUNC-01 自检通过，请 @Reviewer 评审】**（若有多实例未测，同条注明）。

---

## 3. @Dev_Python 确认清单

1. **范围**：**SYS-FUNC-01** 验收路径 **不依赖** `ai_service` 进程；Python 侧 **无本条专属实现项**。  
2. **阻塞性**：确认当前 **`docker-compose.yml`** 与 **`config.yaml`** 默认组合下，**不强制** 启动侧车即可完成 **`SYS_FUNC_01_DOCKER.md` / 集成脚本 §1**（若团队另有「全 compose 含侧车」的 CI 矩阵，请说明侧车**未启动/失败**时是否仍应通过 **01**——默认 **01 不要求侧车**）。

**回执暗号**：回复 **【AI 侧对 SYS-FUNC-01 无交付项，不阻塞验收；请 @Reviewer / @QA 按流水线继续】**。

---

## 4. 与 PRD 缺口说明（避免误判「100% FR-A01」）

- **SYS-FUNC-01** 验收的是 **系统级**「创建精确规则 + 用户命中 SSE + 重复 KEY 拒绝」等（见 §3.1 表）。  
- **FR-A01** 产品表中的 **PATCH/DELETE/列表** 等管理面能力若未全部实现，**不自动等价于 SYS-FUNC-01 未完成**——但若 @PM 要求「FR-A01 全表闭环」，须**另开**需求/验收项，由架构师更新 **§3** 与流水线。

---

## 5. 修订记录

| 日期 | 变更 |
|---|---|
| 2026-03-27 | 首版：双端确认单、暗号、与 SYS-FUNC-01 / FR-A01 边界说明。 |
| 2026-03-27 | **闭环**：`go test ./...` 与 `scripts/test_integration.sh` §1 已通过；**@QA【测试通过】** 已记入 **`SYS_ACCEPTANCE_PIPELINE.md` §3**；§1 次序 1 **已验收**并解锁次序 2。 |

---

## 6. 流水线暗号（归档）

- **@Reviewer**（01 范围）：**【准予上线】**（与 `REVIEWER_CHECKLIST_SYS.md` **SYS-FUNC-01** 行对齐，2026-03-27）。  
- **@QA**：**【测试通过】**（SYS-FUNC-01，2026-03-27）。  
- **解锁下一闸门**：**【SYS-FUNC-01 已验收，请 @Dev_Go / @QA 按 `SYS_ACCEPTANCE_PIPELINE.md` 推进次序 2（SYS-FUNC-02）签核】**（若 §3 第 2 行尚未签字，须单独补跑 **SYS_FUNC_02** 工件）。
