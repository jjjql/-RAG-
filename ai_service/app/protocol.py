"""uint32 大端长度前缀 + UTF-8 JSON，对齐 UDS_EMBEDDING.md 与 Go internal/embedding/frame.go。"""

from __future__ import annotations

import json
import struct
from typing import Any

MAX_PAYLOAD_LEN = 4_194_304  # 4 MiB，与 Go maxPayloadLen 一致


def go_trim_embed_text(s: str) -> str:
    """与 Go embedding.trimEmbedText 一致的空白裁剪（仅空格/制表/换行）。"""

    while len(s) > 0 and s[0] in " \t\n\r":
        s = s[1:]
    while len(s) > 0 and s[-1] in " \t\n\r":
        s = s[:-1]
    return s


class FrameTooLarge(Exception):
    """单帧 JSON 超过上限。"""


def encode_frame(obj: dict[str, Any]) -> bytes:
    payload = json.dumps(obj, ensure_ascii=False, separators=(",", ":")).encode("utf-8")
    if len(payload) > MAX_PAYLOAD_LEN:
        raise FrameTooLarge()
    return struct.pack("!I", len(payload)) + payload


def decode_frame_header(n: int) -> None:
    if n == 0:
        raise ValueError("帧长度为 0")
    if n > MAX_PAYLOAD_LEN:
        raise FrameTooLarge()


def parse_json_payload(raw: bytes) -> dict[str, Any]:
    try:
        data = json.loads(raw.decode("utf-8"))
    except (UnicodeDecodeError, json.JSONDecodeError) as e:
        raise ValueError(f"JSON 解析失败: {e}") from e
    if not isinstance(data, dict):
        raise ValueError("根节点须为 JSON Object")
    return data
