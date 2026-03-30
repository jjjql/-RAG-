"""Embedding 推理封装；默认 fastembed，可注入 Mock 供测试。"""

from __future__ import annotations

import logging
import math
import os
import threading
from typing import Protocol, runtime_checkable

logger = logging.getLogger(__name__)

# 可通过环境变量覆盖；体积小、便于侧车冷启动
_DEFAULT_MODEL = os.environ.get(
    "RAG_AI_EMBED_MODEL",
    "BAAI/bge-small-en-v1.5",
)


@runtime_checkable
class EmbedBackend(Protocol):
    """推理后端抽象。"""

    def embed(self, text: str, model_hint: str | None) -> tuple[list[float], str]:
        """返回 (向量, 实际模型标识)。"""
        ...


class MockEmbedBackend:
    """确定性假向量，不下载权重；供 pytest 与本地无模型环境。"""

    def embed(self, text: str, model_hint: str | None) -> tuple[list[float], str]:
        dim = 8
        # 简单可复现：与文本长度相关
        base = float((sum(ord(c) for c in text) % 997) / 997.0)
        vec = [math.sin(base + i * 0.1) for i in range(dim)]
        model = model_hint or "mock-embed"
        return vec, model


class FastEmbedBackend:
    """fastembed 推理；首次调用时加载模型（ping 路径不触发）。"""

    def __init__(self, model_name: str | None = None) -> None:
        self._model_name = model_name or _DEFAULT_MODEL
        self._lock = threading.Lock()
        self._model: object | None = None

    def _ensure_loaded(self) -> None:
        with self._lock:
            if self._model is not None:
                return
            try:
                from fastembed import TextEmbedding
            except ImportError as e:
                raise RuntimeError("未安装 fastembed，请安装 ai_service 依赖") from e
            logger.info("加载 fastembed 模型: %s", self._model_name)
            self._model = TextEmbedding(model_name=self._model_name)

    def embed(self, text: str, model_hint: str | None) -> tuple[list[float], str]:
        self._ensure_loaded()
        model = self._model
        assert model is not None
        # fastembed：对单条文本生成向量
        gen = model.embed([text])  # type: ignore[union-attr]
        row = next(gen)
        try:
            flat = row.tolist()  # type: ignore[union-attr]
        except AttributeError:
            flat = list(row)  # type: ignore[arg-type]
        vec = [float(x) for x in flat]
        if not vec:
            raise RuntimeError("模型返回空向量")
        if not all(math.isfinite(x) for x in vec):
            raise RuntimeError("embedding 含非有限数")
        # model_hint 仅作提示；响应填实际侧车模型名
        return vec, self._model_name


def create_default_backend() -> EmbedBackend:
    """由环境变量 RAG_AI_BACKEND=mock|fastembed 选择实现。"""

    kind = os.environ.get("RAG_AI_BACKEND", "fastembed").lower().strip()
    if kind == "mock":
        return MockEmbedBackend()
    return FastEmbedBackend()
