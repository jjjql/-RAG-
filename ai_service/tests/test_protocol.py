"""协议辅助函数单测。"""

from app.protocol import go_trim_embed_text


def test_go_trim_embed_text() -> None:
    assert go_trim_embed_text("  ab \n") == "ab"
    assert go_trim_embed_text("x") == "x"
