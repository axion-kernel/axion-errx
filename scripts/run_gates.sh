#!/usr/bin/env bash

# 从脚本位置确定项目根，使调用方可在任意工作目录启动统一门禁。
#
# 参数原样交给 Python 入口，exec 使退出状态和信号直接由门禁进程负责。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

exec python3 scripts/agents/check_all_gates.py "$@"
