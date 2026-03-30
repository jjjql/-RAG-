# Go 侧模块级开发需求（@Architect）

> **拆分说明**：本文承接 `SYSTEM_DESIGN.md` **§4** 中 **Go** 部分，供 **@Dev_Go** 估点、拆分任务与自检。  
> **总览与边界**仍以 `SYSTEM_DESIGN.md` 全文为准；若模块职责与 PRD/契约冲突，以 PRD（`specs/requirements.md`）与 `interface/openapi.yaml` 为准并回写系统设计。  
> **系统级需求开发顺序**：须遵守 **`SYS_ACCEPTANCE_PIPELINE.md`**（上一项 **QA【测试通过】** 后才可开工下一项 SYS-*）。  
> **配套**：Python 侧见同目录 **`PYTHON_DEV_REQUIREMENTS.md`**。  
> **UDS（Embedding）内部契约**：**`contracts/UDS_EMBEDDING.md`（v1.0 冻结）** + **`contracts/uds_embedding.schema.json`** + **`contracts/UDS_INTERNAL_ALIGNMENT.md`**（与 @Dev_Python 对齐；联调 PR 补签 §1）。

---

## 1. 范围

- **代码目录**：`cmd/`、`internal/`（与 `.cursorrules` 一致）。  
- **本期**：北向 **HTTP**（用户 SSE + 管理 JSON）；**北向 gRPC 不交付**（`internal/northbound/grpc` 仅占位）。

---

## 2. 模块级需求表（`internal/` + `cmd/`）

| 模块 | 目标 | 关键需求点（可测试） | 备注 |
|---|---|---|---|
| `cmd/gateway` | 启动/退出 | 读取 `config.yaml`；优雅停机；退出码行为明确 |  |
| `internal/config` | 配置 | 外部组件地址、超时、开关、合并策略参数全部可配置 | 与 `.cursorrules` Viper 一致 |
| `internal/northbound/http` | 用户 + 管理北向 | 严格实现 `interface/openapi.yaml`；用户 **`POST /v1/qa` 为 SSE**（`meta`/`delta`/`done`/`error`）；管理端 JSON；错误模型与 `X-Trace-Id` 与契约一致 | **本期无北向 gRPC**；对齐 FR-Uxx / 管理 FR-Axx |
| `internal/admin` | 管理 API | 精确/正则 CRUD；`priority` / `updated_at` 字段；校验与 **精确优先于正则**、正则内部优先级落地；**本期网关不内建管理端鉴权**（见 `SYSTEM_DESIGN.md` §1.1） | 对齐 FR-A01、FR-A02 |
| `internal/rules` + `internal/cache` | 极速拦截 | 精确 O(1) 或等价；正则预编译并按 **priority↓、同 priority 最新更新** 遍历；**精确命中短路**；scope 隔离 | **不得**在命中路径访问 Redis；满足 NFR-P01 |
| `internal/rulesync` | 热更新 | Redis 存储模型；Pub/Sub 事件；原子替换内存索引；可选全量回源 | 对齐 `.cursorrules` |
| `internal/embedding` | UDS 客户端 | **严格实现** `specs/architecture/contracts/UDS_EMBEDDING.md`（uint32 BE 长度前缀 + JSON）；**100ms** 级超时；错误分类；连接/池化（避免冷连接放大 P99）；支持 **ping** 就绪探测 | 耗时计入 **NFR-P02**（`SYSTEM_DESIGN.md` §2.3）；载荷校验可对照 `uds_embedding.schema.json` |
| `internal/vector` | L3 | `VectorStore` 接口；Qdrant 实现；阈值配置；Mock | 检索耗时计入 **NFR-P02**（§2.3）；单测可离线 |
| `internal/downstream` | 智能问答 | **`mock`** 与 **`langchain_http`**（契约 **`contracts/HTTP_LANGCHAIN_DOWNSTREAM.md` v0.1）；超时；`sessionId`/`traceId` 透传 | 对齐 **FR-U01**、**FR-U03** |
| `internal/coalesce` | 合并 | 相似 key 定义；等待超时；并发正确性 | 对齐 `.cursorrules` |
| `internal/observability` | 可观测 | trace_id；分阶段耗时；Prometheus 埋点 | 支撑 SYS-PERF 口径拆分 |

---

## 3. 单测要求（引用 `.cursorrules`）

- `internal/*` **每个包**具备 `_test.go`。  
- **向量**与**下游**必须可 **Mock**，保证单测不依赖外网。

---

> 修订记录：由架构师在变更模块边界或新增 `internal` 包时更新本表，并在 `SYSTEM_DESIGN.md` 中保留 §4 索引指向本文。
>
> - **2026-03-27（@Dev_Go）**：`internal/embedding` 已落地 UDS/TCP 帧客户端（`uint32` BE + JSON、`embed`/`ping`、默认 100ms embed 超时、单连接互斥顺序复用）；`config.yaml` 增加 `embedding` 段；`POST /v1/qa` 在 `embedding.enabled=true` 时于精确未命中后调用侧车 embed（向量库与 RAG 仍占位）。单测覆盖帧编解码与 TCP 替身对端。
> - **2026-03-27（@Architect）**：**本期裁剪网关侧管理端鉴权**：自 `GO_DEV_REQUIREMENTS` 移除 `internal/authn` 行；**NFR-G02** 由部署层落实；OpenAPI 递增至 **0.2.3**；@Dev_Go 移除 `admin.bearer_token` 与 `Authorization` 校验实现。
> - **2026-03-27（@Architect）**：**FR-A02 设计验收闭环**：落盘 **`FR_A02_DESIGN.md`**（需求映射、Redis 频道、优先级策略）；**SYS-FUNC-02** E2E 步骤见 **`test/e2e/SYS_FUNC_02_DOCKER.md`**。Go 侧实现要点：`internal/rules`（`RedisRegexStore` / `RegexMemoryIndex` / `RegexCoordinator`）、`internal/northbound/http/admin_regex.go` 与 `qa.go` 正则分支；单测见 `regex_*_test.go`。**【FR-A02 设计已归档，请 @QA 按 SYS_FUNC_02 验收；@Dev_Go 仅在有差距时增量修复】**
> - **2026-03-27（@Architect）**：**SYS-FUNC-03 / SYS-ENG-02/03**：新增 **`internal/downstream`**（mock）、**`internal/observability`**（`gateway_qa_completed_total`）、**`GET /metrics`**；**`config.downstream`**；**`scripts/test_integration.sh`** 串行覆盖 SYS-FUNC-01～05 + metrics。正式闸门仍以 **`SYS_ACCEPTANCE_PIPELINE.md`** 与 @QA【测试通过】为准。
> - **2026-03-27（@Architect）**：**FR-U01 + 南向 LangChain**：冻结 **`HTTP_LANGCHAIN_DOWNSTREAM.md` v0.1**；Go 侧 **`langchain_http`** 客户端 + **`sessionId` 透传**；见 **`FR_U01_LANGCHAIN.md`**。**【请 @Dev_Go 联调真实 LangChain 服务并完成 E2E；@Dev_Python 按契约实现 FastAPI 端点】**
> - **2026-03-27（@Architect）**：**SYS-FUNC-01 双端确认**：见 **`SYS_FUNC_01_DEV_CONFIRM.md`**。**【请 @Dev_Go 按 §2 自检并回复暗号 → @Reviewer → @QA】**
> - **2026-03-30（@Architect）**：**SYS-ENG-01 熔断**：`internal/circuitbreaker`；`embedding.CircuitService`、`downstream.WrapAnswerer`；`config.yaml` `circuit_breaker`；**SYS-PERF-02 部分**：`gateway_qa_phase_duration_seconds`（见 **`SYS_OBSERVABILITY_METRICS.md`**）。流水线次序 **2～4、7～9** 已随集成/单测归档。
> - **2026-03-30（@Architect）**：**FR-A01 闭环**：精确规则 **`PATCH` / `DELETE`**（`internal/rules` + `admin_exact.go`）；追溯 **`FR_A01_EXACT_CRUD.md`**；**`scripts/test_integration.sh`** 已含 FR-A01 段。
