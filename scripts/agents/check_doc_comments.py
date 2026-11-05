"""基于 Go AST 检查当前变更的中文文档注释及段间空行。"""

from __future__ import annotations

import json
from pathlib import Path
import re
import subprocess

from doc_comment_format import CHINESE_RE, validate_doc_comment

ROOT = Path(__file__).resolve().parents[2]


def changed_lines(root: Path, relative: str) -> set[int] | None:
    """changed_lines 计算相对 HEAD 的新增或删除邻接行。

    要做什么：
    - 让注释校验覆盖代码变更及仅修改注释的差异。

    具体怎么做：
    - 无 HEAD 或未跟踪文件返回全量标记，否则解析零上下文差异。
    - 删除没有新增行时保留删除位置两侧，防止删除空注释行绕过校验。

    输出什么：
    - 返回一基行号集合或全量标记；差异命令失败抛出运行错误。
    """
    def git(*args):
        """git 在指定测试或项目根查询版本信息。

        要做什么：
        - 为变更范围判定提供统一 Git 调用。

        具体怎么做：
        - 传入分立参数，以 UTF-8 捕获输出，不自动检查返回码。

        输出什么：
        - 返回进程结果，状态由外层调用者判断。
        """
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
    """validate_short_comment 校验类型和字段等简短说明。

    要做什么：
    - 要求中文行注释，并在可识别的新标题段落前留空行。

    具体怎么做：
    - 保留原注释行用于前缀校验，检查非空说明是否含中文及空行占位文本。
    - 遇到非列表且以全角冒号结尾的标题，检查前一行是否为空。

    输出什么：
    - 返回问题列表；本函数不要求函数文档的固定三节。
    """
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
    """find_violations 将 AST 文档条目与变更行组合校验。

    要做什么：
    - 对受检根内的函数、方法及其他文档说明生成定位诊断。

    具体怎么做：
    - 在工具所属仓运行 Go 提取器，解析条目并按路径缓存变更行。
    - 跳过未触及当前差异的条目，分别调用完整或简短文档校验。

    输出什么：
    - 返回带路径、行号和名称的违规列表；提取失败抛出运行错误。
    """
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
