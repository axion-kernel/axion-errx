"""基于 Go AST 检查当前变更的中文文档注释及段间空行。"""

from __future__ import annotations

import json
from pathlib import Path
import re
import subprocess

from doc_comment_format import CHINESE_RE, validate_doc_comment

ROOT = Path(__file__).resolve().parents[2]


def changed_lines(root: Path, relative: str) -> set[int] | None:
    def git(*args):
        return subprocess.run(["git", *args], cwd=root, capture_output=True, text=True, encoding="utf-8", check=False)

    if git("rev-parse", "--verify", "HEAD").returncode:
        return None
    if git("ls-files", "--error-unmatch", "--", relative).returncode:
        return None
    result = git("diff", "--no-ext-diff", "--unified=0", "HEAD", "--", relative)
    if result.returncode:
        raise RuntimeError(result.stderr)
    lines: set[int] = set()
    for start, count in re.findall(r"@@ -\d+(?:,\d+)? \+(\d+)(?:,(\d+))? @@", result.stdout):
        width = int(count) if count else 1
        # 删除空注释行时 Git 没有新增行；检查删除位置两侧，避免空行规则被绕过。
        lines.update(range(int(start), int(start) + (width if width else 2)))
    return lines


def validate_short_comment(block: list[str]) -> list[str]:
    if not block:
        return ["缺少文档注释"]
    lines = [raw.removeprefix("//").strip() for raw in block]
    problems: list[str] = []
    for index, line in enumerate(lines):
        if not block[index].startswith("//"):
            problems.append("文档注释必须使用 // 行注释")
        if line and not CHINESE_RE.search(line):
            problems.append("文档注释说明必须使用中文")
        if "[空行]" in line:
            problems.append("段落之间必须使用空的 // 行，不能写 [空行]")
        if index and line.endswith("：") and not line.startswith("- ") and lines[index - 1]:
            problems.append("文档注释段落之间必须空一行 //")
    return problems


def find_violations(root: Path = ROOT) -> list[str]:
    result = subprocess.run(["go", "run", "./scripts/agents/cmd/doc_comments", "--root", str(root)],
                            cwd=ROOT, capture_output=True, text=True, encoding="utf-8", check=False)
    if result.returncode:
        raise RuntimeError(result.stdout + result.stderr)
    changes: dict[str, set[int] | None] = {}
    violations: list[str] = []
    for entry in json.loads(result.stdout):
        relative = entry["path"]
        if relative not in changes:
            changes[relative] = changed_lines(root, relative)
        changed = changes[relative]
        if changed is not None and not any(entry["start"] <= line <= entry["end"] for line in changed):
            continue
        block = entry["comment"] or []
        problems = validate_doc_comment(block) if entry["function"] else validate_short_comment(block)
        violations.extend(f"{relative}:{entry['start']}: {entry['name']}: {problem}" for problem in problems)
    return violations
