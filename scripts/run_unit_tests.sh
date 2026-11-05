#!/usr/bin/env bash

# 从项目根发现 Go 包，再执行当前测试入口既有的包测试流程。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

# 按行保存非空包路径；发现命令失败仍沿用既有的空结果处理，不新增失败策略。
packages=()
while IFS= read -r line; do
  [[ -n "${line}" ]] || continue
  packages+=("${line}")
done < <(go list ./... || true)

# 没有可执行包时保留原来的成功跳过行为。
if [[ ${#packages[@]} -eq 0 ]]; then
  echo "[run_unit_tests] 跳过: 当前仓库没有可执行 Go 包"
  exit 0
fi

# 对已发现的全部包同步执行测试，返回测试命令的最终状态。
echo "[run_unit_tests] running unit packages"
go test "${packages[@]}"
