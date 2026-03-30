# Go ↔ Python UDS 契约：Embedding 服务（架构 **v1.0 冻结**）

> **维护**：@Architect；**实现**：@Dev_Go（`internal/embedding` 客户端）、@Dev_Python（`ai_service` 侧车监听与推理）。  
> **对齐**：`.cursorrules`（**100ms 级超时**、UDS、可观测 `trace_id`）、`SYSTEM_DESIGN.md` §2.3（Embedding 耗时计入 **NFR-P02** 前置段）。  
> **机器可读载荷**：同目录 **`uds_embedding.schema.json`**（JSON Schema Draft 2020-12）。  
> **wire 版本**：帧内 JSON **`protocolVersion` = 1**；**非兼容**变更须递增 **`protocolVersion`**（或另立 v2 契约）并走 PR。  
> **对齐与归档留痕**：同目录 **`UDS_INTERNAL_ALIGNMENT.md`**（双端确认清单 + 架构归档回执）。

---

## 1. 传输与连接

| 项 | 约定 |
|---|---|
| **载体** | **Unix Domain Socket（SOCK_STREAM）**；Linux/macOS 为主。Windows 若暂无 UDS，由部署文档约定 **TCP 回退**（如 `127.0.0.1:端口`），但**帧格式与本契约相同**，且须在 `config.yaml` 显式开关，避免与生产 UDS 口径分叉。 |
| **默认路径** | `/tmp/rag_gateway.sock`（与 `.cursorrules` 一致）；**必须可配置**（网关与侧车读取各自配置，路径不一致则无法连通）。 |
| **字节序** | 长度前缀 **uint32 大端（big-endian）**。 |
| **帧格式** | **一帧 = 4 字节长度 N + N 字节 UTF-8 JSON 文本**。`N` **不包含**这 4 字节本身。 |
| **N 上限** | **4_194_304（4 MiB）**；超出则对端应丢弃并关闭连接，或返回 `FRAME_TOO_LARGE`（见 §5）。 |
| **单事务** | 一次完整调用：**客户端写 1 帧请求 → 服务端读满并解析 → 服务端写 1 帧响应 → 客户端读满并解析**。 |
| **连接复用** | 允许在**同一连接**上**顺序**发送多对「请求帧/响应帧」；**禁止**在未读完响应前交错发送第二个请求（除非未来定义多路复用版本）。 |
| **半包/粘包** | 必须按长度读满 `N` 字节再 `json.Unmarshal`；禁止依赖分隔符。 |

---

## 2. JSON 通用约束

- 根节点为 **JSON Object**。  
- UTF-8，**禁止 BOM**。  
- 字段名使用 **camelCase**（与北向 OpenAPI 习惯一致）。  
- `requestId`：**必填**，**UUID** 字符串（RFC 4122），用于日志与跨进程关联；**响应必须回显同一 `requestId`**。  
- `traceId`：**选填**，与 HTTP `X-Trace-Id` 同形（UUID 字符串）；侧车应写入日志/metrics。  
- `protocolVersion`：**必填**，当前固定 **1**。

---

## 3. 请求（Request）

### 3.1 `kind` 字段

| `kind` | 含义 |
|--------|------|
| **（省略）** 或 **`embed`** | 计算一条文本的向量（默认）。 |
| **`ping`** | 健康探测，**不得**加载大模型；用于启动探测与编排就绪检查。 |

### 3.2 嵌入请求（`embed`）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `protocolVersion` | int | 是 | 1 |
| `requestId` | string | 是 | UUID |
| `kind` | string | 否 | 缺省视为 `embed` |
| `traceId` | string | 否 | 追踪 |
| `text` | string | 是 | 待嵌入文本；**不得为空**（trim 后长度 ≥ 1） |
| `model` | string | 否 | 期望模型标识；侧车可忽略或校验 |

### 3.3 健康请求（`ping`）

| 字段 | 类型 | 必填 |
|------|------|------|
| `protocolVersion` | int | 是 |
| `requestId` | string | 是 |
| `kind` | string | 是，值 `ping` |

---

## 4. 响应（Response）

### 4.1 成功：嵌入（`embed`）

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `protocolVersion` | int | 是 | 1 |
| `requestId` | string | 是 | 与请求一致 |
| `dimensions` | int | 是 | `embedding` 长度，且 > 0 |
| `embedding` | number[] | 是 | **float32 语义**；JSON 中为 number；全有限数 |
| `model` | string | 是 | 实际使用的模型标识（侧车填写） |

### 4.2 成功：健康（`ping` → `pong`）

| 字段 | 类型 | 必填 |
|------|------|------|
| `protocolVersion` | int | 是 |
| `requestId` | string | 是 |
| `kind` | string | 是，值 `pong` |
| `serverVersion` | string | 否 | 侧车版本/commit，便于排障 |

### 4.3 失败：统一错误体

响应仍为一帧 JSON，**HTTP 状态码不适用**；用字段表示失败：

| 字段 | 类型 | 必填 |
|------|------|------|
| `protocolVersion` | int | 是 |
| `requestId` | string | 是 |
| `error` | object | 是 |

`error` 子字段：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `code` | string | 是 | 机器可读，见 §5 |
| `message` | string | 是 | 人类可读 |
| `details` | object | 否 | 扩展诊断（键值对） |

---

## 5. `error.code` 枚举（v1.0 冻结）

| code | 含义 | 典型场景 |
|------|------|----------|
| `UNSUPPORTED_VERSION` | 不支持的 `protocolVersion` | 主版本不匹配 |
| `VALIDATION_ERROR` | 参数非法 | 空 `text`、缺字段、JSON 结构不符 schema |
| `FRAME_TOO_LARGE` | 帧超大 | N > 4MiB |
| `MODEL_NOT_READY` | 模型未就绪 | 权重加载中 |
| `MODEL_OOM` | 侧车内存不足 | 推理失败 |
| `INFERENCE_FAILED` | 推理失败（泛化） | 非 OOM 的模型错误 |
| `INTERNAL` | 未分类内部错误 | 侧车 bug / 未捕获异常 |

**网关侧**（Go）在超时、对端断连、读不到完整帧时，应映射为本地错误（不必强行塞进 UDS JSON），并在日志中带 `requestId`/`traceId`。

---

## 6. 超时、并发与背压（工程约束）

| 项 | 约定 |
|---|---|
| **客户端超时** | 网关调用侧车 **默认 100ms**（`.cursorrules`）；可配置，但生产变更需架构/评审确认对 NFR-P02 的影响。 |
| **服务端** | 侧车应尽快返回；无法在时限内完成时仍应返回一帧 **`INFERENCE_FAILED` 或 `MODEL_NOT_READY`**（优于静默断连）。 |
| **连接数** | Go 端宜 **连接池** 或 **每请求短连接**；须避免冷连接在 P99 放大（见 `GO_DEV_REQUIREMENTS.md`）。 |
| **侧车隔离** | Python OOM/崩溃**不得**拖死 Go；Go 侧须 **捕获 EOF/reset** 并走熔断/降级策略（产品文案由北向层统一）。 |

---

## 7. 示例（仅演示帧内 JSON，不含 4 字节长度前缀）

**Embed 请求体：**

```json
{
  "protocolVersion": 1,
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "traceId": "6ba7b810-9dad-11d1-80b4-00c04fd430c8",
  "text": "你好，世界",
  "model": "fastembed-example"
}
```

**Embed 成功响应体：**

```json
{
  "protocolVersion": 1,
  "requestId": "550e8400-e29b-41d4-a716-446655440000",
  "dimensions": 384,
  "model": "fastembed-example",
  "embedding": [0.01, -0.02, 0.03]
}
```

**Ping / Pong：**

```json
{"protocolVersion":1,"requestId":"550e8400-e29b-41d4-a716-446655440001","kind":"ping"}
```

```json
{"protocolVersion":1,"requestId":"550e8400-e29b-41d4-a716-446655440001","kind":"pong","serverVersion":"ai_service-0.1.0"}
```

---

## 8. 联调验收清单（@QA 可选用）

1. **ping**：冷启动侧车后先发 `ping`，期望 **1 帧 `pong`**，延迟可高于 100ms（仅就绪检查）。  
2. **embed**：固定 `text` 与模型，**dimensions** 与 `len(embedding)` 一致；同一请求 **requestId** 回显。  
3. **错误**：空 `text` → `VALIDATION_ERROR`；错误帧仍合法 JSON 且带 `requestId`。  
4. **超时**：侧车注入睡眠 > 客户端 deadline，Go 侧应超时失败且不阻塞北向事件循环（具体断言由集成测试编写）。

---

## 9. 修订与冻结

### 9.1 冻结记录（v1.0）

| 项 | 内容 |
|---|---|
| **状态** | **v1.0 冻结**（本文 + `uds_embedding.schema.json`） |
| **冻结 / 归档日期** | **2026-03-26** |
| **依据** | **`UDS_INTERNAL_ALIGNMENT.md` §3 架构师归档回执**；双端 **§1 勾选**须在联调 PR / 纪要中补链 |

### 9.2 后续修订策略

- **兼容澄清**：可修订文字/示例，保持 **`protocolVersion: 1`**。  
- **非兼容扩展**：须 **`protocolVersion` 2** 或新契约；双端同步升级。  
- **禁止**仅改一端导致 silent mismatch。

---

**架构师派发说明**：请 @Dev_Go / @Dev_Python 以本文 + `uds_embedding.schema.json` 为**唯一事实来源**实现编解码；变更 **必须** PR 并同步更新**两文件**与本节。
