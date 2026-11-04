#!/usr/bin/env python3
"""统一门禁入口。"""

from __future__ import annotations

import os
import re
import subprocess
import sys
from pathlib import Path
from typing import Sequence

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
    """门禁执行失败。"""


def run_cmd(cmd: Sequence[str], *, check: bool = True) -> subprocess.CompletedProcess[str]:
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
    rel_parts = dir_path.relative_to(ROOT_DIR).parts if dir_path != ROOT_DIR else ()
    return any(part in SKIP_DIR_NAMES for part in rel_parts)


def _target_roots(env_name: str) -> tuple[str, ...]:
    raw = os.environ.get(env_name, "").strip()
    if not raw:
        return DEFAULT_PACKAGE_ROOTS
    return tuple(item.strip() for item in raw.split(",") if item.strip())


def _iter_go_dirs() -> list[Path]:
    dirs: set[Path] = set()
    for file_path in ROOT_DIR.rglob("*.go"):
        if _should_skip_dir(file_path.parent):
            continue
        dirs.add(file_path.parent)
    return sorted(dirs)


def _go_files_in_dir(dir_path: Path) -> list[Path]:
    return sorted(path for path in dir_path.glob("*.go") if path.is_file())


def _read_package_name(file_path: Path) -> str:
    for line in file_path.read_text(encoding="utf-8").splitlines():
        match = PACKAGE_PATTERN.match(line)
        if match is not None:
            return match.group(1)
    raise GateError(f"文件缺少 package 声明: {file_path.relative_to(ROOT_DIR)}")


def run_docs_gate() -> None:
    missing = [item for item in REQUIRED_DOC_PATHS if not (ROOT_DIR / item).exists()]
    if missing:
        raise GateError("缺少以下文档文件:\n" + "\n".join(f"- {item}" for item in missing))

    text = (ROOT_DIR / "AGENTS.md").read_text(encoding="utf-8")
    missing_sections = [item for item in REQUIRED_AGENTS_SECTIONS if item not in text]
    if missing_sections:
        raise GateError("AGENTS.md 缺少以下 section:\n" + "\n".join(f"- {item}" for item in missing_sections))

    print("[docs-gate] 通过: 基础文档入口齐全")


def run_python_path_gate() -> None:
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


def run_go_test_gate() -> None:
    proc = run_cmd(["go", "test", "./..."], check=False)
    if proc.returncode != 0:
        raise GateError(proc.stdout + proc.stderr)
    print(proc.stdout, end="")
    print("[go-test-gate] 通过: go test ./... 成功")


def main(argv: list[str] | None = None) -> int:
    _ = argv
    try:
        run_docs_gate()
        run_python_path_gate()
        run_package_root_gate()
        run_subpackage_gate()
        run_go_format_gate()
        run_go_test_gate()
    except GateError as err:
        print(f"[all-gates] {err}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
