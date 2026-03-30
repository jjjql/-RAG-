"""pytest：UDS/TCP 帧契约 + mock 嵌入（无需下载权重）。"""

from __future__ import annotations

import asyncio
import json
import queue
import socket
import struct
import threading
import uuid

import pytest

from app.embedder import MockEmbedBackend
from app.main import _handle_client


def _frame(payload: bytes) -> bytes:
    return struct.pack("!I", len(payload)) + payload


def _read_frame(conn: socket.socket, timeout: float = 5.0) -> bytes:
    conn.settimeout(timeout)
    hdr = conn.recv(4)
    assert len(hdr) == 4
    (n,) = struct.unpack("!I", hdr)
    chunks: list[bytes] = []
    got = 0
    while got < n:
        part = conn.recv(n - got)
        assert part, "连接提前关闭"
        chunks.append(part)
        got += len(part)
    return b"".join(chunks)


def _start_tcp_server() -> int:
    q: queue.Queue[int] = queue.Queue()

    async def _amain() -> None:
        embedder = MockEmbedBackend()
        srv = await asyncio.start_server(
            lambda r, w: _handle_client(r, w, embedder),
            "127.0.0.1",
            0,
        )
        assert srv.sockets
        port = srv.sockets[0].getsockname()[1]
        q.put(int(port))
        async with srv:
            await srv.serve_forever()

    def _runner() -> None:
        asyncio.run(_amain())

    th = threading.Thread(target=_runner, daemon=True)
    th.start()
    return q.get(timeout=10)


@pytest.fixture(scope="module")
def tcp_port() -> int:
    return _start_tcp_server()


def test_ping_pong(tcp_port: int) -> None:
    req = {
        "protocolVersion": 1,
        "requestId": str(uuid.uuid4()),
        "kind": "ping",
    }
    raw = json.dumps(req, separators=(",", ":")).encode("utf-8")
    with socket.create_connection(("127.0.0.1", tcp_port), timeout=5) as s:
        s.sendall(_frame(raw))
        body = _read_frame(s)
    res = json.loads(body.decode("utf-8"))
    assert res["protocolVersion"] == 1
    assert res["requestId"] == req["requestId"]
    assert res["kind"] == "pong"
    assert "serverVersion" in res


def test_embed_mock_dimensions(tcp_port: int) -> None:
    req = {
        "protocolVersion": 1,
        "requestId": str(uuid.uuid4()),
        "text": "hello 契约",
    }
    raw = json.dumps(req, separators=(",", ":")).encode("utf-8")
    with socket.create_connection(("127.0.0.1", tcp_port), timeout=5) as s:
        s.sendall(_frame(raw))
        body = _read_frame(s)
    res = json.loads(body.decode("utf-8"))
    assert "error" not in res
    assert res["dimensions"] == len(res["embedding"])
    assert res["dimensions"] == 8
    assert isinstance(res["model"], str)


def test_embed_empty_text_validation_error(tcp_port: int) -> None:
    req = {
        "protocolVersion": 1,
        "requestId": str(uuid.uuid4()),
        "text": "   \t",
    }
    raw = json.dumps(req, separators=(",", ":")).encode("utf-8")
    with socket.create_connection(("127.0.0.1", tcp_port), timeout=5) as s:
        s.sendall(_frame(raw))
        body = _read_frame(s)
    res = json.loads(body.decode("utf-8"))
    assert res.get("error", {}).get("code") == "VALIDATION_ERROR"


def test_unsupported_protocol_version(tcp_port: int) -> None:
    rid = str(uuid.uuid4())
    req = {"protocolVersion": 2, "requestId": rid, "text": "x"}
    raw = json.dumps(req, separators=(",", ":")).encode("utf-8")
    with socket.create_connection(("127.0.0.1", tcp_port), timeout=5) as s:
        s.sendall(_frame(raw))
        body = _read_frame(s)
    res = json.loads(body.decode("utf-8"))
    assert res["error"]["code"] == "UNSUPPORTED_VERSION"
    assert res["requestId"] == rid


def test_sequential_round_trips_same_connection(tcp_port: int) -> None:
    """契约：同连接顺序多帧（禁止交错）。"""
    with socket.create_connection(("127.0.0.1", tcp_port), timeout=5) as s:
        ping = {
            "protocolVersion": 1,
            "requestId": str(uuid.uuid4()),
            "kind": "ping",
        }
        s.sendall(_frame(json.dumps(ping, separators=(",", ":")).encode("utf-8")))
        r1 = json.loads(_read_frame(s).decode("utf-8"))
        assert r1["kind"] == "pong"

        emb = {
            "protocolVersion": 1,
            "requestId": str(uuid.uuid4()),
            "text": "seq",
        }
        s.sendall(_frame(json.dumps(emb, separators=(",", ":")).encode("utf-8")))
        r2 = json.loads(_read_frame(s).decode("utf-8"))
        assert r2["dimensions"] == len(r2["embedding"])
