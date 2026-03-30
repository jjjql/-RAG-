# Windows + Docker：健康检查（GET /v1/health）

## 前提

- 已安装 **Docker Desktop**（WSL2 后端推荐），且 `docker compose` 可用。
- 在仓库根目录 `D:\study\rag\-RAG-`（或你的克隆路径）打开终端（PowerShell / CMD）。

## 构建并启动

```powershell
cd D:\study\rag\-RAG-
docker compose build
docker compose up -d
```

（当前 `docker-compose.yml` 包含 **redis** 与 **gateway**；仅测 health 时也可只起 gateway，但 **SYS-FUNC-01** 需 Redis，请用完整 `up -d`。）

## 探测 health（宿主机执行）

```powershell
curl.exe -sS -D - "http://127.0.0.1:8080/v1/health" -H "Accept: application/json"
```

### 期望

- HTTP **200**
- 响应体：`{"status":"ok"}`
- 响应头含 **`X-Trace-Id`**
- `Content-Type` 含 `application/json`

## 查看日志 / 停止

```powershell
docker compose logs -f gateway
docker compose down
```

## 说明

- `docker-compose.yml` 将宿主机的 **`config.yaml`** 只读挂载到容器 **`/app/config.yaml`**，并通过 **`GATEWAY_REDIS_ADDR`** 指向 compose 内 Redis；修改监听地址后执行 `docker compose up -d` 即可生效（无需重建镜像，除非改代码）。
- 容器内监听 `server.http_addr`（默认 `:8080`），映射到宿主 **`8080:8080`**。
