# SYS-FUNC-03 / FR-U03（Docker）验收步骤

> **前提**：`config.yaml` 中 **`downstream.enabled: true`** 且 **`mode: mock`**（默认示例已开启）；未命中精确/正则时走 **mock 智能问答**，SSE **`done.source`=`rag`**。  
> 自动化：仓库根目录 **`bash scripts/test_integration.sh`**（含本项断言）。

## 手工 curl

```powershell
curl.exe -sS -N -X POST "http://127.0.0.1:8080/v1/qa" `
  -H "Content-Type: application/json" `
  -H "Accept: text/event-stream" `
  -d "{\"query\":\"任意不与规则冲突的长随机串\"}"
```

期望：流内含 **`event: done`** 且 JSON 中 **`"source":"rag"`**；`delta` 拼接文本与 **`downstream.mock_text`** 一致。

## 下游故障 / 超时（NFR-G01 子集）

将 `downstream.timeout_ms` 设为 **1**，且使用可注入延迟的替身时（未来 HTTP 模式），应出现流内 **`event:error`** 或明确错误语义且不永久挂死。当前 **mock** 无延迟时以配置项说明为主。
