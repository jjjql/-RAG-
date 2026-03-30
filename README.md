# RAG 语义网关

产品需求见 `specs/requirements.md`，系统设计见 `specs/architecture/SYSTEM_DESIGN.md`，工程约束见 `.cursorrules`。**系统级需求（SYS-*）串行验收闸门**见 `specs/architecture/SYS_ACCEPTANCE_PIPELINE.md`。北向契约（OpenAPI）见 `interface/openapi.yaml`；**用户问答为 SSE**（`POST /v1/qa`）；**精确/正则规则与 DAT 的管理接口**及健康检查为 **JSON**。范例见 `interface/EXAMPLES.md`。

**Windows + Docker**：`docker compose up -d` 启动 **redis + gateway**（见根目录 `docker-compose.yml`）。健康检查见 `test/e2e/DOCKER_WINDOWS.md`；**SYS-FUNC-01**（精确规则 + SSE）见 `test/e2e/SYS_FUNC_01_DOCKER.md`；**SYS-FUNC-02 / FR-A02**（正则规则 + SSE）见 `test/e2e/SYS_FUNC_02_DOCKER.md` 与 `specs/architecture/FR_A02_DESIGN.md`；**SYS-FUNC-03** 见 `test/e2e/SYS_FUNC_03_DOCKER.md`。  
**串行验收闸门**见 `specs/architecture/SYS_ACCEPTANCE_PIPELINE.md`；一键预检（Docker 可用时自动起 compose）：**`bash scripts/test_integration.sh`**（Git Bash / WSL / Linux）。
