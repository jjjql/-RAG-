# Go ↔ Python 内部设计对齐与架构归档（UDS Embedding）

> **角色**：@Dev_Go、`internal/embedding`；@Dev_Python、`ai_service` UDS 服务；**裁决与归档**：@Architect。  
> **事实源**：`UDS_EMBEDDING.md` + `uds_embedding.schema.json`（与北向 `interface/openapi.yaml` **独立**）。

---

## 1. 双端对齐确认清单（实现前 / 联调后均可勾选）

请 @Dev_Go、@Dev_Python 各自确认已理解且无实现歧义（在 PR 或评审纪要中勾选等同）：

| # | 确认项 | @Dev_Go | @Dev_Python |
|---:|---|:---:|:---:|
| 1 | **帧格式**：`uint32` **大端**长度前缀 + **N 字节 UTF-8 JSON**；`N` 不含前缀 4 字节 | ☐ | ☐ |
| 2 | **单事务**：1 请求帧 → 1 响应帧；同连接**顺序**复用，禁止交错 | ☐ | ☐ |
| 3 | **`protocolVersion`**：wire 层固定 **1** | ☐ | ☐ |
| 4 | **`embed`**：`requestId` UUID；`text` 非空；响应 `dimensions` 与 `embedding` 长度一致 | ☐ | ☐ |
| 5 | **`ping`/`pong`**：`ping` 不加载大模型；`pong` 可带 `serverVersion` | ☐ | ☐ |
| 6 | **`error.code`**：与 `UDS_EMBEDDING.md` §5 枚举一致；错误帧仍带 `requestId` | ☐ | ☐ |
| 7 | **超时**：网关侧车调用 **默认 100ms**（`.cursorrules`），可配置 | ☐ | ☐ |
| 8 | **JSON Schema**：载荷可与 `uds_embedding.schema.json` 校验一致（实现允许附加只读字段须双方约定） | ☐ | ☐ |

---

## 2. 双端共同声明（建议粘贴到 PR 描述或会议纪要）

- 我俩已就 **`UDS_EMBEDDING.md` + `uds_embedding.schema.json`（当前仓库版本）** 完成对齐，**无开放问题**。  
- 建议向 @Architect 发送暗号：**【UDS 双端内部设计已对齐，请 @Architect 归档】**

---

## 3. 架构师归档回执（@Architect）

| 项 | 内容 |
|---|---|
| **归档日期** | **2026-03-26** |
| **依据** | 团队指令：在内部设计对齐前提下完成 UDS **v1.0 冻结**与文档归档；**§1 勾选**由双端在后续联调 PR / 纪要中补链（见备注）。 |
| **已执行** | `UDS_EMBEDDING.md` 升级为 **v1.0 冻结**；`uds_embedding.schema.json` 更新 **`$id`/说明**；`contracts/README.md`；`SYSTEM_DESIGN.md` §1.1 / §6.3 / §6.4；本文档 §3。 |
| **wire 版本** | 帧内 JSON **`protocolVersion: 1`**（与「契约文档 v1.0」不同维度） |

**备注**：若联调发现与文档不符，双端须提 **PR** 修订契约并遵循 `UDS_EMBEDDING.md` §9.2（非兼容须升 `protocolVersion` 或新契约）。
