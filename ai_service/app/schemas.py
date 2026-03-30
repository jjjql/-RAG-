"""UDS 载荷 Pydantic 模型，对齐 specs/architecture/contracts/uds_embedding.schema.json。"""

from __future__ import annotations

from typing import Any, Literal
from uuid import UUID

from pydantic import BaseModel, ConfigDict, Field, field_validator

from app.protocol import go_trim_embed_text

# 契约 §5 错误码（v1.0 冻结）
ErrorCode = Literal[
    "UNSUPPORTED_VERSION",
    "VALIDATION_ERROR",
    "FRAME_TOO_LARGE",
    "MODEL_NOT_READY",
    "MODEL_OOM",
    "INFERENCE_FAILED",
    "INTERNAL",
]


class EmbedRequest(BaseModel):
    """嵌入请求（kind 省略时由上层视为 embed）。"""

    model_config = ConfigDict(extra="forbid", populate_by_name=True)

    protocol_version: int = Field(alias="protocolVersion")
    request_id: UUID = Field(alias="requestId")
    trace_id: UUID | None = Field(default=None, alias="traceId")
    kind: Literal["embed"] | None = None
    text: str
    model: str | None = None

    @field_validator("text")
    @classmethod
    def text_non_empty_after_trim(cls, v: str) -> str:
        t = go_trim_embed_text(v)
        if len(t) < 1:
            raise ValueError("text 不得为空（trim 后长度须 ≥ 1）")
        return t


class PingRequest(BaseModel):
    model_config = ConfigDict(extra="forbid", populate_by_name=True)

    protocol_version: int = Field(alias="protocolVersion")
    request_id: UUID = Field(alias="requestId")
    trace_id: UUID | None = Field(default=None, alias="traceId")
    kind: Literal["ping"]


class ErrorBody(BaseModel):
    model_config = ConfigDict(extra="forbid", populate_by_name=True)

    code: ErrorCode
    message: str = Field(min_length=1)
    details: dict[str, Any] | None = None


class ErrorResponse(BaseModel):
    model_config = ConfigDict(extra="forbid", populate_by_name=True)

    protocol_version: Literal[1] = Field(default=1, alias="protocolVersion")
    request_id: str = Field(alias="requestId")
    error: ErrorBody


class EmbedSuccessResponse(BaseModel):
    model_config = ConfigDict(extra="forbid", populate_by_name=True)

    protocol_version: Literal[1] = Field(alias="protocolVersion", default=1)
    request_id: str = Field(alias="requestId")
    dimensions: int = Field(ge=1)
    embedding: list[float]
    model: str = Field(min_length=1)

    @field_validator("embedding")
    @classmethod
    def embedding_non_empty(cls, v: list[float]) -> list[float]:
        if len(v) < 1:
            raise ValueError("embedding 非空")
        return v


class PongResponse(BaseModel):
    model_config = ConfigDict(extra="forbid", populate_by_name=True)

    protocol_version: Literal[1] = Field(alias="protocolVersion", default=1)
    request_id: str = Field(alias="requestId")
    kind: Literal["pong"] = "pong"
    server_version: str | None = Field(default=None, alias="serverVersion")


def error_json(
    request_id: str,
    code: ErrorCode,
    message: str,
    details: dict[str, Any] | None = None,
) -> str:
    r = ErrorResponse(
        protocolVersion=1,
        requestId=request_id,
        error=ErrorBody(code=code, message=message, details=details),
    )
    return r.model_dump_json(by_alias=True, exclude_none=True)
