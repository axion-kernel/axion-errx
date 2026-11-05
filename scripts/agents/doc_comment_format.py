"""校验中文文档注释的摘要、空行及三段式列表。"""

from __future__ import annotations

import re
from collections.abc import Sequence

DOC_MARKERS = ("要做什么：", "具体怎么做：", "输出什么：")
CHINESE_RE = re.compile(r"[\u3400-\u4dbf\u4e00-\u9fff\U00020000-\U000323af]")


def validate_doc_comment(block: Sequence[str]) -> list[str]:
    """validate_doc_comment 校验函数文档的中文段落结构。

    要做什么：
    - 要求中文摘要、固定顺序的三节标题和各节中文列表。

    具体怎么做：
    - 保留行注释中的空行，检查说明语言及占位空行文本。
    - 检查标题缺失、重复和乱序，再核对标题前空行与列表内容。

    输出什么：
    - 返回按检测顺序收集的问题；缺少标题或顺序错误时不继续检查列表。
    """
    if not block:
        return ["缺少文档注释"]
    problems: list[str] = []
    lines: list[str] = []
    for raw in block:
        raw = raw.strip()
        if not raw.startswith("//"):
            return ["文档注释必须使用 // 行注释"]
        lines.append(raw[2:].strip())

    if not lines[0] or lines[0] in DOC_MARKERS or lines[0].startswith("- "):
        problems.append("文档注释首行必须是中文摘要")
    for line in lines:
        if line and not CHINESE_RE.search(line):
            problems.append("文档注释说明必须使用中文")
            break
    if any("[空行]" in line for line in lines):
        problems.append("段落之间必须使用空的 // 行，不能写 [空行]")

    indexes: list[int] = []
    for marker in DOC_MARKERS:
        matches = [index for index, line in enumerate(lines) if line == marker]
        if not matches:
            problems.append(f"文档注释缺少「{marker}」")
        elif len(matches) > 1:
            problems.append(f"文档注释标题「{marker}」只能出现一次")
        if matches:
            indexes.append(matches[0])
    if len(indexes) != len(DOC_MARKERS):
        return problems
    if indexes != sorted(indexes):
        problems.append("文档注释标题必须依次为：要做什么、具体怎么做、输出什么")
        return problems

    for section, start in enumerate(indexes):
        marker = DOC_MARKERS[section]
        if start == 0 or lines[start - 1] != "":
            problems.append(f"文档注释「{marker}」前必须空一行 //")
        end = indexes[section + 1] if section + 1 < len(indexes) else len(lines)
        items = [line for line in lines[start + 1:end] if line]
        if not items:
            problems.append(f"文档注释「{marker}」下至少需要一条中文子列表")
        elif any(not line.startswith("- ") or not CHINESE_RE.search(line[2:]) for line in items):
            problems.append(f"文档注释「{marker}」下必须使用 - 中文说明 列表")
    return problems
