"""
LangChain 南向契约 MOCK（HTTP_LANGCHAIN_DOWNSTREAM.md v0.1）。
不调用真实 LLM，仅返回固定结构 JSON，供网关 Docker 全链路联调。
"""
from __future__ import annotations

from fastapi import FastAPI
from pydantic import BaseModel, Field

app = FastAPI(title="LangChain RAG Invoke MOCK", version="0.1.0")


class InvokeRequest(BaseModel):
    query: str = Field(..., min_length=1)
    sessionId: str | None = None
    traceId: str | None = None


class InvokeResponse(BaseModel):
    answer: str
    explanation: str | None = None


@app.get("/health")
def health() -> dict[str, str]:
    return {"status": "ok"}


@app.post("/v1/rag/invoke", response_model=InvokeResponse)
def invoke(body: InvokeRequest) -> InvokeResponse:
    # 模拟 LangChain / RAG 管道：不访问模型，仅回显与标记
    snippet = body.query.strip().replace("\n", " ")[:280]
    sid = body.sessionId or "(none)"
    tid = body.traceId or "(none)"
    return InvokeResponse(
        answer=f"[MOCK LangChain 容器] 已收到问题：{snippet}",
        explanation=f"未调用 LLM；sessionId={sid} traceId={tid}",
    )
