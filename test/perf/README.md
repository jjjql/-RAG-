# 性能验收（SYS-PERF-01 / SYS-PERF-02）

> 与 `SYSTEM_DESIGN.md` §3.2、§2.3 **NFR-P01 / NFR-P02** 对齐。  
> **闸门**：须在 **`SYS_ACCEPTANCE_PIPELINE.md`** 中 **SYS-FUNC-05** 及更早条目 **QA【测试通过】** 后再执行专项压测。

## SYS-PERF-01（规则极速 P99 &lt; 15ms）

- **前置**：固定规则集 + 固定问题集；**不含**外部模型时间。  
- **工具**：推荐 [k6](https://k6.io/)，脚本见同目录 `k6_rules_smoke.js`（示例，需按环境改 `BASE_URL`）。  
- **判据**：在固化并发、Payload、实例数下输出 **P99 &lt; 15ms**（以网关进程内打点或 k6 客户端观测为准，口径需与架构一致）。

## SYS-PERF-02（智能问答前置段 P99 &lt; 100ms）

- **前置**：未命中规则样本中含「走向量」比例；外部 RAG 为**可控替身**；网关按 §2.3 **起止点**打点（含 Embedding + 向量检索）。  
- **当前仓库**：向量库与完整打点仍待补齐；脚本与指标字典落地后再做正式签字。

## 运行示例

```bash
# 需网关已启动且可访问（默认 http://127.0.0.1:8080）；脚本会在 setup 中创建精确规则 k6-smoke-key
k6 run test/perf/k6_rules_smoke.js

# 按 NFR-P01 收紧 P99&lt;15ms（本机/CI 可能因负载抖动失败，供正式验收）
STRICT_PERF=1 k6 run test/perf/k6_rules_smoke.js
```

（需本机安装 k6；未安装时仅保留本文档作为验收占位。）
