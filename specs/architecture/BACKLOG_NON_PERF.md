# 非性能类剩余 backlog（架构师滚动）

> **说明**：与 **`SYS_ACCEPTANCE_PIPELINE.md`** 中 **SYS-PERF-*** 无关的待办；**性能 / k6 / P99** 不在此表展开。

| 项 | 状态 | 建议负责 | 备注 |
|---|------|----------|------|
| **FR-A01 精确 CRUD** | **已闭环（2026-03-30）** | @Dev_Go | **`FR_A01_EXACT_CRUD.md`** |
| **相似请求合并（coalesce）** | 未实现 | @Dev_Go | 须在契约或设计说明中写死「相似」键；见 `SYSTEM_DESIGN.md` §6.3 |
| **向量检索 L3（Qdrant）+ 语义缓存** | 未接库 | @Dev_Go | `internal/vector`、与 `qa.go` 编排；下游仍 Mock/LangChain |
| **Prometheus 指标全量字典** | 部分 | @Architect / @Dev_Go | 见 **`SYS_OBSERVABILITY_METRICS.md`**；stage/outcome 枚举待固化 |
| **NFR-G02 管理端鉴权** | 网关不实现 | 部署 / 前置网关 | 见 `SYSTEM_DESIGN.md` §1.1 |
| **北向 OpenAPI 产品签核** | 待 @PM | @PM | `interface/PM_PRODUCT_REVIEW.md` 通过后递增 `info.version` |
| **Python 侧车生产联调签核** | 待双端 | @Dev_Python / @Dev_Go | **`UDS_INTERNAL_ALIGNMENT.md` §1** |

| 日期 | 变更 |
|---|---|
| 2026-03-30 | 首版：承接「除性能测试外」未完成任务清单。 |
