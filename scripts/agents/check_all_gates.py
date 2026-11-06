#!/usr/bin/env python3
"""统一门禁入口。"""

from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Sequence

from check_doc_comments import find_violations

ROOT_DIR = Path(__file__).resolve().parents[2]
SKIP_DIR_NAMES = {
    ".cache",
    ".git",
    ".venv",
    "__pycache__",
    "bin",
    "dist",
    "node_modules",
    "vendor",
}
FORBIDDEN_PYTHON_PATHS = (
    "/opt/homebrew/bin/" + "python3",
    "/usr/bin/" + "python3",
)
REQUIRED_DOC_PATHS = (
    "AGENTS.md",
    "README.md",
    "docs/README.md",
    "docs/agents/index.md",
    "docs/agents/errata.md",
    "docs/arch/architecture.md",
    "docs/code-style/go.md",
    "docs/spec/requirements-baseline.md",
)
REQUIRED_AGENTS_SECTIONS = (
    "## 目录",
    "## AI 易错勘误",
)
DEFAULT_PACKAGE_ROOTS = (
    "internal",
    "pkg",
)
PACKAGE_PATTERN = re.compile(r"^\s*package\s+([A-Za-z_][A-Za-z0-9_]*)\b")


class GateError(RuntimeError):
    """GateError 表示统一入口可转换为失败退出码的门禁异常。"""


def run_cmd(cmd: Sequence[str], *, check: bool = True) -> subprocess.CompletedProcess[str]:
    """run_cmd 在项目根同步运行门禁命令。

    要做什么：
    - 为检查器提供统一的子进程输出和失败诊断。

    具体怎么做：
    - 继承环境并以 UTF-8 捕获输出，将缺失命令转换为门禁错误。
    - 按检查开关决定是否将非零退出转换为带标准流的异常。

    输出什么：
    - 返回已完成进程对象；需要阻断时抛出 GateError。
    """
    try:
        proc = subprocess.run(
            cmd,
            cwd=ROOT_DIR,
            text=True,
            encoding="utf-8",
            errors="replace",
            capture_output=True,
            check=False,
            env=os.environ.copy(),
        )
    except FileNotFoundError as err:
        raise GateError(f"命令不存在: {cmd[0]}") from err

    if check and proc.returncode != 0:
        raise GateError(
            f"命令执行失败: {' '.join(cmd)}\n"
            f"stdout:\n{proc.stdout}\n"
            f"stderr:\n{proc.stderr}"
        )
    return proc


def _should_skip_dir(dir_path: Path) -> bool:
    """_should_skip_dir 判断目录是否命中扫描排除项。

    要做什么：
    - 保留缓存、依赖和产物目录的既有排除范围。

    具体怎么做：
    - 将目录转换为项目相对路径，逐级匹配排除名称。

    输出什么：
    - 返回是否跳过；目录必须位于项目根内。
    """
    rel_parts = dir_path.relative_to(ROOT_DIR).parts if dir_path != ROOT_DIR else ()
    return any(part in SKIP_DIR_NAMES for part in rel_parts)


def _target_roots(env_name: str) -> tuple[str, ...]:
    """_target_roots 读取受检包根配置。

    要做什么：
    - 为不同结构门禁保留环境覆盖入口。

    具体怎么做：
    - 空设置使用默认根，否则按逗号分割并清理空项。

    输出什么：
    - 返回目录名称元组，不验证目录存在性。
    """
    raw = os.environ.get(env_name, "").strip()
    if not raw:
        return DEFAULT_PACKAGE_ROOTS
    return tuple(item.strip() for item in raw.split(",") if item.strip())


def _iter_go_dirs() -> list[Path]:
    """_iter_go_dirs 收集包含 Go 文件的非排除目录。

    要做什么：
    - 为父子包结构检查建立目录集合。

    具体怎么做：
    - 递归查找 Go 路径，过滤排除目录并按父目录去重排序。

    输出什么：
    - 返回排序后的目录列表，不解析源码。
    """
    dirs: set[Path] = set()
    for file_path in ROOT_DIR.rglob("*.go"):
        if _should_skip_dir(file_path.parent):
            continue
        dirs.add(file_path.parent)
    return sorted(dirs)


def _go_files_in_dir(dir_path: Path) -> list[Path]:
    """_go_files_in_dir 列出目录直属 Go 文件。

    要做什么：
    - 区分直接包含源码的包与仅包含子目录的节点。

    具体怎么做：
    - 匹配当前层 Go 后缀并过滤非文件路径，再排序。

    输出什么：
    - 返回直属文件列表，不递归搜索。
    """
    return sorted(path for path in dir_path.glob("*.go") if path.is_file())


def _read_package_name(file_path: Path) -> str:
    """_read_package_name 读取源码中的包声明名称。

    要做什么：
    - 为目录与包名一致性检查提供声明值。

    具体怎么做：
    - 按 UTF-8 逐行匹配既有包声明正则，采用首个匹配。

    输出什么：
    - 返回包名；找不到声明时抛出带相对路径的门禁错误。
    """
    for line in file_path.read_text(encoding="utf-8").splitlines():
        match = PACKAGE_PATTERN.match(line)
        if match is not None:
            return match.group(1)
    raise GateError(f"文件缺少 package 声明: {file_path.relative_to(ROOT_DIR)}")


def run_docs_gate() -> None:
    """run_docs_gate 检查文档入口与根规则章节。

    要做什么：
    - 保持小型共享库的必需文档和导航结构。

    具体怎么做：
    - 比较登记路径的存在性，再检查根 AGENTS.md 的固定章节文本。

    输出什么：
    - 成功打印通过信息；缺失路径或章节抛出门禁错误。
    """
    missing = [item for item in REQUIRED_DOC_PATHS if not (ROOT_DIR / item).exists()]
    if missing:
        raise GateError("缺少以下文档文件:\n" + "\n".join(f"- {item}" for item in missing))

    text = (ROOT_DIR / "AGENTS.md").read_text(encoding="utf-8")
    missing_sections = [item for item in REQUIRED_AGENTS_SECTIONS if item not in text]
    if missing_sections:
        raise GateError("AGENTS.md 缺少以下 section:\n" + "\n".join(f"- {item}" for item in missing_sections))

    print("[docs-gate] 通过: 基础文档入口齐全")


def run_python_path_gate() -> None:
    """run_python_path_gate 查找写死的 Python 解释器路径。

    要做什么：
    - 阻止仓库文本或其他文件包含已禁止的解释器绝对路径。

    具体怎么做：
    - 按既有排除目录遍历文件，逐行比较原始字节与禁止片段。
    - 为命中行保存相对路径、行号和可替换解码的诊断文本。

    输出什么：
    - 无命中打印通过信息；存在命中时合并报告并抛出门禁错误。
    """
    violations: list[str] = []
    for current_root, dir_names, file_names in os.walk(ROOT_DIR):
        current_path = Path(current_root)
        if _should_skip_dir(current_path):
            dir_names[:] = []
            continue
        dir_names[:] = [name for name in dir_names if not _should_skip_dir(current_path / name)]
        for file_name in file_names:
            file_path = current_path / file_name
            if not file_path.is_file():
                continue
            content = file_path.read_bytes()
            for line_no, raw_line in enumerate(content.splitlines(), start=1):
                for forbidden in FORBIDDEN_PYTHON_PATHS:
                    if forbidden.encode("utf-8") not in raw_line:
                        continue
                    line = raw_line.decode("utf-8", errors="replace").strip()
                    rel = file_path.relative_to(ROOT_DIR).as_posix()
                    violations.append(f"{rel}:{line_no}: {line}")
                    break

    if violations:
        raise GateError(
            "禁止在仓库任意文件中写死 Python 解释器绝对路径；请改用 PATH 中的 python3 或 "
            "#!/usr/bin/env python3:\n" + "\n".join(violations)
        )

    print("[python-path-gate] 通过: 未发现写死 Python 解释器绝对路径")


def run_package_root_gate() -> None:
    """run_package_root_gate 检查受检总根不直属存放 Go 文件。

    要做什么：
    - 保留源码按子包组织的目录约束。

    具体怎么做：
    - 读取受检根设置，只收集已存在根的直属 Go 文件。

    输出什么：
    - 无违规打印通过信息；否则列出路径并抛出门禁错误。
    """
    violations: list[str] = []
    for root in _target_roots("PACKAGE_DEP_TARGET_ROOTS"):
        root_path = ROOT_DIR / root
        if not root_path.is_dir():
            continue
        violations.extend(
            file_path.relative_to(ROOT_DIR).as_posix()
            for file_path in sorted(root_path.glob("*.go"))
            if file_path.is_file()
        )

    if violations:
        raise GateError(
            "以下受检根目录下不应直接放置 .go 文件:\n" + "\n".join(f"- {item}" for item in violations)
        )

    print("[package-root-gate] 通过: 受检根目录未直接堆放 Go 文件")


def run_subpackage_gate() -> None:
    """run_subpackage_gate 核对父子包结构和包名。

    要做什么：
    - 拒绝含源码的包目录继续嵌套 Go 子包，并检查受检目录名一致性。

    具体怎么做：
    - 收集非排除 Go 目录，比较各目录是否存在含源码的后代。
    - 在配置根内逐文件读取包名，允许目录同名及其外部测试包名。

    输出什么：
    - 无源码沿用跳过；通过时打印结果，违规时合并诊断并抛出门禁错误。
    """
    go_dirs = _iter_go_dirs()
    if not go_dirs:
        print("[subpackage-gate] 跳过: 当前仓库尚无 Go 源码")
        return

    violations: list[str] = []
    for dir_path in go_dirs:
        direct_go_files = _go_files_in_dir(dir_path)
        if not direct_go_files:
            continue
        child_go_dirs = sorted(
            child
            for child in go_dirs
            if child != dir_path and child.is_relative_to(dir_path) and _go_files_in_dir(child)
        )
        if child_go_dirs:
            rel_dir = dir_path.relative_to(ROOT_DIR).as_posix()
            rel_children = ", ".join(child.relative_to(ROOT_DIR).as_posix() for child in child_go_dirs)
            violations.append(f"{rel_dir}: child_go_packages=[{rel_children}]")

    for root in _target_roots("SUBPACKAGE_GATE_TARGET_ROOTS"):
        root_path = ROOT_DIR / root
        if not root_path.is_dir():
            continue
        for file_path in sorted(root_path.rglob("*.go")):
            if _should_skip_dir(file_path.parent) or file_path.parent == root_path:
                continue
            package_name = _read_package_name(file_path)
            dir_name = file_path.parent.name
            allowed = {dir_name, f"{dir_name}_test"}
            if package_name not in allowed:
                rel_path = file_path.relative_to(ROOT_DIR).as_posix()
                violations.append(f"{rel_path}: package={package_name}, 期望={dir_name} 或 {dir_name}_test")

    if violations:
        raise GateError("分层子包门禁失败:\n" + "\n".join(f"- {item}" for item in violations))

    print("[subpackage-gate] 通过: 父子包结构正常")


def run_go_format_gate() -> None:
    """run_go_format_gate 检查既有范围内的 Go 格式。

    要做什么：
    - 要求非排除源码已经经过 gofmt。

    具体怎么做：
    - 收集相对路径并调用只列举模式，检查非空输出行。

    输出什么：
    - 无源码沿用跳过；格式正确打印通过，未格式化文件导致门禁错误。
    """
    go_files = [
        file_path.relative_to(ROOT_DIR).as_posix()
        for file_path in sorted(ROOT_DIR.rglob("*.go"))
        if file_path.is_file() and not _should_skip_dir(file_path.parent)
    ]
    if not go_files:
        print("[go-format-gate] 跳过: 当前仓库尚无 Go 源码")
        return

    proc = run_cmd(["gofmt", "-l", *go_files])
    unformatted = [line for line in proc.stdout.splitlines() if line.strip()]
    if unformatted:
        raise GateError("以下 Go 文件尚未 gofmt:\n" + "\n".join(f"- {item}" for item in unformatted))

    print("[go-format-gate] 通过: 所有 Go 文件已格式化")
    proc = run_cmd(["go", "run", "./scripts/agents/cmd/go_decl_layout"])
    print(proc.stdout, end="")


def run_go_test_gate() -> None:
    """run_go_test_gate 执行项目全部 Go 包测试。

    要做什么：
    - 将真实包测试结果纳入统一门禁。

    具体怎么做：
    - 在项目根调用 go test ./...，检查返回码并转发标准输出。

    输出什么：
    - 成功打印测试输出及通过信息；失败以两条标准流抛出门禁错误。
    """
    proc = run_cmd(["go", "test", "./..."], check=False)
    if proc.returncode != 0:
        raise GateError(proc.stdout + proc.stderr)
    print(proc.stdout, end="")
    print("[go-test-gate] 通过: go test ./... 成功")


def run_doc_comment_gate() -> None:
    """run_doc_comment_gate 汇集当前文档注释违规。

    要做什么：
    - 对统一入口暴露中文注释和段间空行检查结果。

    具体怎么做：
    - 调用 AST 注释检查包装器，将执行异常转换为门禁错误。
    - 对返回的违规逐项组成错误报告。

    输出什么：
    - 无违规打印通过；执行失败或格式违规抛出 GateError。
    """
    try:
        violations = find_violations()
    except RuntimeError as err:
        raise GateError(str(err)) from err
    if violations:
        raise GateError("中文文档注释门禁失败:\n" + "\n".join(violations))
    print("[doc-comment-gate] 通过: 中文文档注释与段间空行符合约束")


def run_script_tests_gate() -> None:
    """run_script_tests_gate 运行工程脚本回归。

    要做什么：
    - 保留中文注释门禁自身的回归验证。

    具体怎么做：
    - 使用当前解释器发现工程目录中约定后缀的 unittest 模块并同步执行。

    输出什么：
    - 成功转发标准输出并打印通过；失败由命令包装器抛出门禁错误。
    """
    proc = run_cmd([sys.executable, "-m", "unittest", "discover", "-s", "scripts/agents", "-p", "*_test.py"])
    print(proc.stdout, end="")
    print("[script-tests-gate] 通过: 注释门禁回归成功")


def main(argv: list[str] | None = None) -> int:
    """main 按固定顺序执行共享错误库门禁。

    要做什么：
    - 汇集文档、路径、包结构、注释、工程回归和 Go 测试。

    具体怎么做：
    - 保留当前未使用的参数接口，依次调用检查器，首个门禁异常即停止。
    - 将约定异常写入标准错误，其他异常按默认行为传播。

    输出什么：
    - 全部通过返回零；捕获门禁错误时返回一。
    """
    _ = argv
    try:
        run_docs_gate()
        run_python_path_gate()
        run_package_root_gate()
        run_subpackage_gate()
        run_go_format_gate()
        run_doc_comment_gate()
        run_script_tests_gate()
        run_go_test_gate()
    except GateError as err:
        print(f"[all-gates] {err}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
