# 手工验收：GET /v1/health（最小 E2E）

## 前置

- 仓库根目录存在 `config.yaml`；`server.http_addr` 决定监听端口（默认示例 `:8080`）。

## 步骤

1. 启动网关：

   ```bash
   go run ./cmd/gateway -config config.yaml
   ```

2. 另开终端：

   ```bash
   curl -sS -D - "http://127.0.0.1:8080/v1/health" -H "Accept: application/json"
   ```

   （请与 `config.yaml` 中 `server.http_addr` 端口一致。）

## 期望

- HTTP **200**
- `Content-Type` 含 `application/json`
- 响应体：`{"status":"ok"}`
- 响应头含 **`X-Trace-Id`**（合法 UUID）

## QA 结论（填写）

- 执行人：________  
- 日期：________  
- 结论：☐ **【测试通过】** / ☐ 不通过（附现象）________  
