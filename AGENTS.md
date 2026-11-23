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

### API 请求与响应结构

API 请求和响应的固定对象必须用 struct，优先复用现有类型，并禁止用 `map[string]any` 组装输出。详见[公共 API 类型规则](../axion-agent-docs/engineering/go/api-payloads.md)。

### 项目群公共勘误

先维护所属设计与测试合同，再改实现。领域拥有规则和默认值，组合根注入依赖。慢初始化在全局锁外，发布时复核身份和代次。持久事件使用稳定强类型；压缩上下文不能作为完整历史。验收执行本仓门禁并记录证据，凭据与运行产物保持零跟踪。详见[项目群易错勘误](../axion-agent-docs/engineering/agent-errata.md)。

Go 错误沿用项目群 Errx。稳定名称、数值码和状态由 owner 唯一注册，已有类型的原因链和数值码跨边界保留。响应只投影业务码与公开文案。涉及上游协议时由 provider adapter 归一化。详见[ErrorType 与传输合同](../axion-agent-docs/architecture/errors/error-contract.md#errortype-注册与使用)。

- 这是一个共享基础库仓库，不是应用仓库；不要引入与 HTTP 运行时、数据库、前端调试器等无关的骨架。
- 项目群统一 module 为 `github.com/axion-kernel/axion-errx`；不要重新引入旧 module 或改动已发布错误身份。
- 对外共享 Go 包固定放在 `pkg/errx`；不要恢复旧目录或改成 `internal/errx`。
- `NewType` 使用参数形式 `NewType("name", code[, status])`，第三个可选参数声明 HTTP 失败状态；不要回退到结构体参数形式。
- 共享错误码 `1000~1999` 需要保持跨仓库稳定，新增通用错误时先检查是否与现有定义重复。
- 门禁统一从 `scripts/agents/check_all_gates.py` 或 `bash scripts/run_gates.sh` 进入。
