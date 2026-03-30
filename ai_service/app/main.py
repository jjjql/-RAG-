"""UDS/TCP 帧服务入口：uint32 BE + JSON，对齐 UDS_EMBEDDING.md v1.0。"""

from __future__ import annotations

import argparse
import asyncio
import logging
import os
import struct
import sys
from typing import Any

from pydantic import ValidationError

from app.embedder import EmbedBackend, create_default_backend
from app.protocol import (
    MAX_PAYLOAD_LEN,
    FrameTooLarge,
    go_trim_embed_text,
    parse_json_payload,
)
from app.schemas import (
    EmbedRequest,
    EmbedSuccessResponse,
    ErrorCode,
    PingRequest,
    PongResponse,
    error_json,
)

logger = logging.getLogger(__name__)

SERVER_VERSION = os.environ.get("RAG_AI_SERVER_VERSION", "ai_service-0.1.0")


def _safe_request_id(data: dict[str, Any] | None) -> str:
    if not data:
        return "00000000-0000-0000-0000-000000000000"
    rid = data.get("requestId")
    if isinstance(rid, str) and rid.strip():
        return rid.strip()
    return "00000000-0000-0000-0000-000000000000"


def _error_frame(request_id: str, code: ErrorCode, message: str) -> bytes:
    return error_json(request_id, code, message).encode("utf-8")


def _wrap_frame(payload: bytes) -> bytes:
    if len(payload) > MAX_PAYLOAD_LEN:
        raise FrameTooLarge()
    return struct.pack("!I", len(payload)) + payload


def _ok_embed_response(req_id: str, embedding: list[float], model: str) -> bytes:
    body = EmbedSuccessResponse(
        protocolVersion=1,
        requestId=req_id,
        dimensions=len(embedding),
        embedding=embedding,
        model=model,
    )
    raw = body.model_dump_json(by_alias=True, exclude_none=True).encode("utf-8")
    return _wrap_frame(raw)


def _ok_pong_response(req_id: str) -> bytes:
    body = PongResponse(
        protocolVersion=1,
        requestId=req_id,
        kind="pong",
        serverVersion=SERVER_VERSION,
    )
    raw = body.model_dump_json(by_alias=True, exclude_none=True).encode("utf-8")
    return _wrap_frame(raw)


def _error_response_bytes(request_id: str, code: ErrorCode, message: str) -> bytes:
    raw = _error_frame(request_id, code, message)
    return _wrap_frame(raw)


async def _read_frame(reader: asyncio.StreamReader) -> bytes:
    hdr = await reader.readexactly(4)
    (n,) = struct.unpack("!I", hdr)
    if n == 0:
        raise ValueError("帧长度为 0")
    if n > MAX_PAYLOAD_LEN:
        raise FrameTooLarge()
    return await reader.readexactly(n)


async def _write_all(writer: asyncio.StreamWriter, data: bytes) -> None:
    writer.write(data)
    await writer.drain()


def _build_dispatch_result(
    data: dict[str, Any],
    embedder: EmbedBackend,
) -> bytes:
    rid = _safe_request_id(data)
    pv = data.get("protocolVersion")
    if pv != 1:
        return _error_response_bytes(
            rid,
            "UNSUPPORTED_VERSION",
            f"不支持的 protocolVersion: {pv!r}（需要 1）",
        )

    kind = data.get("kind")
    if kind == "ping":
        try:
            pr = PingRequest.model_validate(data)
        except ValidationError as e:
            return _error_response_bytes(
                rid,
                "VALIDATION_ERROR",
                f"ping 请求参数非法: {e}",
            )
        return _ok_pong_response(str(pr.request_id))

    if kind not in (None, "embed"):
        return _error_response_bytes(
            rid,
            "VALIDATION_ERROR",
            f"未知 kind: {kind!r}",
        )

    try:
        er = EmbedRequest.model_validate(data)
    except ValidationError as e:
        return _error_response_bytes(
            rid,
            "VALIDATION_ERROR",
            f"embed 请求参数非法: {e}",
        )

    text = go_trim_embed_text(er.text)
    if not text:
        return _error_response_bytes(
            str(er.request_id),
            "VALIDATION_ERROR",
            "text 不得为空（trim 后长度须 ≥ 1）",
        )

    try:
        vec, model = embedder.embed(text, er.model)
    except MemoryError:
        return _error_response_bytes(
            str(er.request_id),
            "MODEL_OOM",
            "侧车内存不足",
        )
    except RuntimeError as e:
        msg = str(e)
        if "权重加载" in msg or "model loading" in msg.lower():
            return _error_response_bytes(
                str(er.request_id),
                "MODEL_NOT_READY",
                msg,
            )
        return _error_response_bytes(
            str(er.request_id),
            "INFERENCE_FAILED",
            msg,
        )
    except Exception as e:  # noqa: BLE001 — 侧车须返回一帧错误，避免静默断连
        logger.exception("推理未捕获异常 requestId=%s", er.request_id)
        return _error_response_bytes(
            str(er.request_id),
            "INTERNAL",
            str(e) or "内部错误",
        )

    if len(vec) < 1:
        return _error_response_bytes(
            str(er.request_id),
            "INFERENCE_FAILED",
            "空 embedding",
        )

    return _ok_embed_response(str(er.request_id), vec, model)


async def _dispatch(
    raw: bytes,
    embedder: EmbedBackend,
) -> bytes:
    try:
        data = parse_json_payload(raw)
    except ValueError as e:
        return _wrap_frame(
            _error_frame(
                "00000000-0000-0000-0000-000000000000",
                "VALIDATION_ERROR",
                str(e),
            )
        )

    loop = asyncio.get_running_loop()
    return await loop.run_in_executor(
        None,
        lambda: _build_dispatch_result(data, embedder),
    )


async def _handle_client(
    reader: asyncio.StreamReader,
    writer: asyncio.StreamWriter,
    embedder: EmbedBackend,
) -> None:
    peer = writer.get_extra_info("peername")
    try:
        while True:
            try:
                raw = await _read_frame(reader)
            except asyncio.IncompleteReadError:
                break
            except FrameTooLarge:
                logger.warning("对端帧超大，关闭连接 peer=%s", peer)
                break
            except (OSError, ConnectionError) as e:
                logger.debug("读帧连接错误: %s", e)
                break

            try:
                out = await _dispatch(raw, embedder)
            except FrameTooLarge:
                logger.warning("响应帧超大 peer=%s", peer)
                break

            try:
                await _write_all(writer, out)
            except (OSError, ConnectionError, BrokenPipeError) as e:
                logger.debug("写响应失败: %s", e)
                break
    finally:
        try:
            writer.close()
            await writer.wait_closed()
        except Exception:  # noqa: BLE001
            pass


async def _run_server(
    embedder: EmbedBackend,
    *,
    unix_path: str | None,
    tcp_host: str,
    tcp_port: int,
) -> None:
    server: asyncio.Server | asyncio.AbstractServer

    if unix_path:
        try:
            if os.path.exists(unix_path):
                os.unlink(unix_path)
        except OSError as e:
            logger.warning("无法删除已有 socket 文件 %s: %s", unix_path, e)
        server = await asyncio.start_unix_server(
            lambda r, w: _handle_client(r, w, embedder),
            path=unix_path,
        )
        logger.info("UDS 监听 unix:%s", unix_path)
    else:
        server = await asyncio.start_server(
            lambda r, w: _handle_client(r, w, embedder),
            host=tcp_host,
            port=tcp_port,
        )
        logger.info("TCP 监听 %s:%s", tcp_host, tcp_port)

    async with server:
        await server.serve_forever()


def _parse_args(argv: list[str] | None = None) -> argparse.Namespace:
    p = argparse.ArgumentParser(description="RAG 网关 Embedding 侧车（UDS/TCP 帧）")
    p.add_argument(
        "--unix-path",
        default=os.environ.get("RAG_AI_SOCKET_PATH", "/tmp/rag_gateway.sock"),
        help="Unix socket 路径；空字符串表示改用 TCP",
    )
    p.add_argument(
        "--tcp-host",
        default=os.environ.get("RAG_AI_TCP_HOST", "127.0.0.1"),
    )
    p.add_argument(
        "--tcp-port",
        type=int,
        default=int(os.environ.get("RAG_AI_TCP_PORT", "18080")),
    )
    return p.parse_args(argv)


def main(argv: list[str] | None = None) -> None:
    logging.basicConfig(
        level=os.environ.get("RAG_AI_LOG_LEVEL", "INFO"),
        format="%(asctime)s %(levelname)s %(name)s %(message)s",
    )
    args = _parse_args(argv)

    transport = os.environ.get("RAG_AI_TRANSPORT", "").strip().lower()
    unix_path: str | None
    if transport == "tcp":
        unix_path = None
    elif transport == "unix":
        unix_path = args.unix_path or "/tmp/rag_gateway.sock"
    else:
        # 未显式指定时：Windows 默认 TCP，POSIX 默认 UDS
        if sys.platform == "win32":
            unix_path = None
        else:
            unix_path = args.unix_path or "/tmp/rag_gateway.sock"

    if unix_path == "":
        unix_path = None

    embedder = create_default_backend()

    try:
        asyncio.run(
            _run_server(
                embedder,
                unix_path=unix_path,
                tcp_host=args.tcp_host,
                tcp_port=args.tcp_port,
            )
        )
    except KeyboardInterrupt:
        logger.info("收到中断，退出")


if __name__ == "__main__":
    main()
