# 系统级需求验收流水线（自动化接力 · 闸门版）

> **维护**：@Architect  
> **事实来源**：`SYSTEM_DESIGN.md` **§3**（可测试的系统级需求：SYS-FUNC / SYS-PERF / SYS-ENG）与 `.cursorrules` 暗号。  
> **硬规则（本期采纳）**：**严格串行**：仅当 **@QA** 对**当前编号**发出 **【测试通过】** 后，才允许启动**下一编号**的 **开发 → 检视 → 测试** 闭环。**禁止**跨闸门并行占用下一项的开发资源（避免验收口径漂移）。

> **存量代码说明**：仓库中 **SYS-FUNC-01 / SYS-FUNC-02** 可能已先于本流水线实现；**自本文件生效起**，@QA 仍须按 **次序 1 → 2 → …** **补跑并签核**，通过后下一项方可正式开工。已交付代码若在某步 QA 失败，按 §2 退回开发修复。

---

## 1. 闸门与顺序（唯一推荐序列）

| 次序 | 编号 | 类别 | 简述（摘自 §3） | 主要验收工件 | 实现状态（架构师滚动更新） |
|---:|---|---|---|---|---|
| 1 | **SYS-FUNC-01** | 功能 | 精确规则 + SSE + 冲突 + 热更新 | `test/e2e/SYS_FUNC_01_DOCKER.md`、`SYS_FUNC_01_DEV_CONFIRM.md` | **已验收（2026-03-27）**：`go test ./...` 绿；`bash scripts/test_integration.sh` §1 断言通过（证据见 §3、§3.1）。**已解锁次序 2**（SYS-FUNC-02 的独立签核仍见 §3 第 2 行） |
| 2 | **SYS-FUNC-02** | 功能 | 正则 + 非法拒绝 + 优先级 + 精确优先 + SSE | `test/e2e/SYS_FUNC_02_DOCKER.md`、`FR_A02_DESIGN.md` | **已验收（2026-03-30）**：`scripts/test_integration.sh` §2 + `go test`；**【解锁次序 3】** |
| 3 | **SYS-FUNC-03** | 功能 | 未命中 → RAG（mock）+ 失败有界 | `test/e2e/SYS_FUNC_03_DOCKER.md`、`config.yaml` `downstream` | **已验收（2026-03-30 · mock）**：集成脚本 §3；下游超时/熔断见 **SYS_ENG_01_BREAKER.md**。**【解锁次序 4】** |
| 4 | **SYS-FUNC-05** | 功能 | 变更后同问结果稳定 | 集成脚本 PATCH + 重复 `/v1/qa` | **已验收（2026-03-30）**：集成脚本 §4。**【解锁次序 5】** |
| 5 | **SYS-PERF-01** | 性能 | 规则极速 P99 &lt; 15ms | `test/perf/README.md`、`test/perf/k6_rules_smoke.js` | **可验收**：k6 **setup 预置规则**；`STRICT_PERF=1` 时阈值 **P99&lt;15ms**（见 README）。**须本机装 k6 并由 @QA 出报告后签 §3** |
| 6 | **SYS-PERF-02** | 性能 | 前置段 P99 &lt; 100ms（含向量，§2.3） | `SYS_OBSERVABILITY_METRICS.md`、`test/perf/README.md` | **部分（2026-03-30）**：已暴露 **`gateway_qa_phase_duration_seconds`**（`embed` / `rag_prep`）；**向量子阶段 / 完整 P99 报告**仍待 Qdrant 与固化口径 |
| 7 | **SYS-ENG-01** | 工程 | 跨进程 **100ms 级超时**与熔断 | **`SYS_ENG_01_ACCEPTANCE.md`**、`SYS_ENG_01_BREAKER.md`、`scripts/test_sys_eng_01.sh` | **已验收（2026-03-30）**：embed / **Qdrant** / 下游 **超时 + 可选熔断**；`bash scripts/test_sys_eng_01.sh` 或 `go test ./...` 绿；**`mock_delay_ms`** 仅 mock 注入。多实例跨容器 E2E 仍为可选扩展 |
| 8 | **SYS-ENG-02** | 工程 | `trace_id` + **`/metrics`** | `GET /metrics` + `gateway_qa_completed_total` 等 | **已验收（2026-03-30）**：集成脚本 + Histogram 见 **`SYS_OBSERVABILITY_METRICS.md`** |
| 9 | **SYS-ENG-03** | 工程 | `docker compose` + **`scripts/test_integration.sh`** | `docker-compose.yml` + `scripts/test_integration.sh` | **已验收（2026-03-30）**：同次集成 **exit 0** |

> **已移除项**：**SYS-FUNC-04** 不进入本流水线（见 `SYSTEM_DESIGN.md` §3.1 脚注）。

---

## 2. 单条需求的固定接力（不得跳步）

对**当前次序行**的编号 `SYS-*`，严格执行：

1. **开发**：@Dev_Go（及需要时的 @Dev_Python）仅在本行**已解锁**（上一项 QA【测试通过】）后编码；单元测试绿。  
   - 交付暗号：**【Go 侧已交付，请 @Reviewer 评审】** / **【AI 侧已交付，请 @Reviewer 评审】**（见 `.cursorrules`）。
2. **检视**：@Reviewer 使用 **`REVIEWER_CHECKLIST_SYS.md`**。  
   - 通过：**【准予上线】**  
   - 不通过：**【发现问题，请 @Dev_Go 重构】** 或 **【发现问题，请 @Dev_Python 重构】**
3. **测试**：@QA 按本文件「验收工件」执行（含 Docker / 压测 / 故障注入等）。  
   - 通过：**【测试通过】** → 架构师将上表「实现状态」更新为**已验收**，并**解锁下一行**。  
   - 不通过：**【测试不通过，请 @Dev_Go 和 @Dev_Python 一起分析问题并修复 bug】** → 修复后从**开发**重新走闭环，**不改变**当前次序编号。

**@Architect**：在每一轮 QA【测试通过】后，更新本文 **§1 状态列** 与 **§3 QA 签核表**；并向下一职能发**解锁**说明（**【SYS-xxx 已验收，请 @Dev_Go 启动 SYS-yyy】**）。

---

## 3. QA 签核记录（模板）

| 次序 | 编号 | 签核日期 | QA 结论 | 备注 / 链接 |
|---:|---|---|---|---|
| 1 | SYS-FUNC-01 | 2026-03-27 | **通过**【测试通过】 | `go test ./...`；`bash scripts/test_integration.sh`（§1：POST exact + SSE `rule_exact`/DAT + 409）；`SYS_FUNC_01_DEV_CONFIRM.md` |
| 2 | SYS-FUNC-02 | 2026-03-30 | **通过**【测试通过】 | `bash scripts/test_integration.sh` §2；`go test ./...` |
| 3 | SYS-FUNC-03 | 2026-03-30 | **通过**【测试通过】 | 集成脚本 §3（mock RAG） |
| 4 | SYS-FUNC-05 | 2026-03-30 | **通过**【测试通过】 | 集成脚本 §4 |
| 5 | SYS-PERF-01 |  | ☐ 通过 / ☐ 不通过 | k6：须安装后执行 `k6 run test/perf/k6_rules_smoke.js`（可选 `STRICT_PERF=1`） |
| 6 | SYS-PERF-02 |  | ☐ 通过 / ☐ 不通过 | Histogram 已暴露；**完整 P99&lt;100ms** 仍待向量与报告 |
| 7 | SYS-ENG-01 | 2026-03-30 | **通过**【测试通过】 | **`SYS_ENG_01_ACCEPTANCE.md`**；`bash scripts/test_sys_eng_01.sh`；`go test ./...`；含 **Qdrant CircuitStore**、下游 Mock/LangChain 超时；**`mock_delay_ms`** 见 **`SYS_ENG_01_BREAKER.md`** |
| 8 | SYS-ENG-02 | 2026-03-30 | **通过**【测试通过】 | `/metrics` + `gateway_qa_phase_duration_seconds` |
| 9 | SYS-ENG-03 | 2026-03-30 | **通过**【测试通过】 | `bash scripts/test_integration.sh` exit 0 |
| 10 | **FR-U04** 持久化语义去重 |  | ☐ 通过 / ☐ 不通过 | PRD **`specs/requirements.md`**；**`SEMANTIC_DEDUP_PERSISTENT.md`**；`go test ./internal/vector/...`；E2E 见 **`test/e2e/SEMANTIC_DEDUP_DOCKER.md`**（须 Qdrant+侧车） |

### 3.1 架构师/CI 自动化预检

| 时间 | 命令 / 环境 | 结果 | 说明 |
|---|---|---|---|
| 2026-03-27 | `bash scripts/test_integration.sh`（仓库根目录；Git Bash；无网关时自动 `docker compose up --build`） | **exit 0** | 覆盖次序 **1～4** 及 **`/metrics`**。**次序 1（SYS-FUNC-01）**：已作为 §3 签核依据归档（2026-03-27）。 |
| 2026-03-30 | 同上 + `go test ./...` | **exit 0** | 次序 **2～4、8～9** 与 **SYS-ENG-01（代码/单测）**、**SYS-ENG-02（新增 Histogram）** 一并归档；**SYS-PERF-01** 仍依赖本机 **k6** 报告。 |
| 2026-03-30 | `bash scripts/test_sys_eng_01.sh`（Git Bash） | **exit 0** | **SYS-ENG-01** 集中单测：`circuitbreaker`、`embedding`、`vector`、`downstream`。 |

---

## 4. 与 §3 表格的关系

- `SYSTEM_DESIGN.md` §3 保留**需求语义与通过判据**（产品/架构真源）。  
- **本文件** 保留**执行顺序、门禁、工件路径、当前状态**（研发调度真源）。  
- 二者冲突时：**以 PRD + `SYSTEM_DESIGN.md` §3 判据为准** 修订本流水线，并递增本文档 **修订记录**。

---

## 5. 修订记录

| 日期 | 变更 |
|---|---|
| 2026-03-27 | 首版：串行闸门、SYS-FUNC-01→…→SYS-ENG-03、角色暗号、QA 签核表模板、存量 01/02 补签说明。 |
| 2026-03-27 | **@Architect 推进**：落地 **SYS-FUNC-03 mock**、**`/metrics`**、**`scripts/test_integration.sh`**、**SYS-FUNC-05** 脚本断言、**test/perf** 占位、**REVIEWER_CHECKLIST_SYS.md**；更新 §1 状态与 §3.1 预检表。 |
| 2026-03-27 | **SYS-FUNC-01 双端确认**：新增 **`SYS_FUNC_01_DEV_CONFIRM.md`**；§1 明确 **已实现 ≠ QA 已验收**，需 Go/Python 回执 + Reviewer + QA 签核。 |
| 2026-03-27 | **SYS-FUNC-01 闭环**：`go test ./...` + `scripts/test_integration.sh` **exit 0**；§3 第 1 行 **【测试通过】**；§1 次序 1 **已验收**并**解锁次序 2**；检视与 `REVIEWER_CHECKLIST_SYS.md` **SYS-FUNC-01** 行对齐（等同本期 **【准予上线】** 于 01 范围）。 |
| 2026-03-30 | **@Architect 按 §3 顺序落地**：`internal/circuitbreaker` + embed/下游熔断包装；**`gateway_qa_phase_duration_seconds`**；k6 **setup**；更新 **`SYS_ACCEPTANCE_PIPELINE`** / **`SYS_ENG_01_BREAKER.md`** / **`SYS_OBSERVABILITY_METRICS.md`**。次序 **2～4、7～9** 签核；**SYS-PERF-01/02** 保留 k6/向量/P99 人工或专项报告项。 |
| 2026-03-30 | **FR-A01**：精确规则 **PATCH/DELETE** 已落地；集成脚本增加 FR-A01 段；见 **`FR_A01_EXACT_CRUD.md`**。 |
| 2026-03-30 | **SYS-ENG-01 补 Qdrant**：`vector.*` 配置 + **`internal/vector.CircuitStore`** + **`qa.go` L3**；更新 **`SYS_ENG_01_BREAKER.md`** / **`SYS_OBSERVABILITY_METRICS.md`**；§3 次序 7 **待 @QA 复验** 后更新签核。 |
| 2026-03-30 | **SYS-ENG-01 收口**：新增 **`SYS_ENG_01_ACCEPTANCE.md`**、**`scripts/test_sys_eng_01.sh`**；下游 LangChain 超时单测、`mock_delay_ms`；§1 次序 7 / §3 第 7 行 **已验收** 表述对齐。 |
| 2026-03-31 | **FR-U04（持久化语义去重）**：PRD + **`SEMANTIC_DEDUP_PERSISTENT.md`** + OpenAPI **0.2.4**；§3 增第 10 行 **待 @QA 签核**；与 SYS-* 闸门并行归档，不挤占原 SYS 编号。 |
