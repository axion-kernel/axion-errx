# AI Agent 导航

> `AGENTS.md` 只放“文档指引目录”和“AI 易错勘误”两个 Section；具体规则统一沉淀到 `docs/`。

## 目录

- Agent 规则总入口：见 [docs/agents/index.md](docs/agents/index.md)。
- AI 易错勘误：见 [docs/agents/errata.md](docs/agents/errata.md)。
- Go 代码风格：见 [docs/code-style/go.md](docs/code-style/go.md)。
- 仓库架构说明：见 [docs/arch/architecture.md](docs/arch/architecture.md)。
- 需求基线：见 [docs/spec/requirements-baseline.md](docs/spec/requirements-baseline.md)。
- 文档目录总览：见 [docs/README.md](docs/README.md)。
- 统一门禁入口：`scripts/agents/check_all_gates.py`
- Shell 门禁入口：`bash scripts/run_gates.sh`

## AI 易错勘误

- 这是一个共享基础库仓库，不是应用仓库；不要引入与 HTTP 运行时、数据库、前端调试器等无关的骨架。
- 项目群统一 module 为 `github.com/axion-kernel/axion-errx`；不要重新引入旧 module 或改动已发布错误身份。
- 对外共享 Go 包固定放在 `pkg/errx`；不要恢复旧目录或改成 `internal/errx`。
- `NewType` 统一使用参数形式 `NewType("name", code)`，不要回退到结构体参数形式。
- 共享错误码 `1000~1999` 需要保持跨仓库稳定，新增通用错误时先检查是否与现有定义重复。
- 门禁统一从 `scripts/agents/check_all_gates.py` 或 `bash scripts/run_gates.sh` 进入。
