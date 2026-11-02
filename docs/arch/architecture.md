# 仓库架构

当前统一 module 为 `github.com/axion-kernel/axion-errx`，公共包入口固定为 `pkgs/errx`。
仓库由原 `axion-box/axion-errx` 直接迁入，保留原提交历史、实现、错误码和测试；仅迁移 Go module 与消费路径。
Axion Kernel 项目群后续所有 Go 项目统一依赖本仓，已有消费方在迁移时切换，禁止同时加载旧 module 和新 module。
错误模型仍保持纯 Go 基础库职责，不引入宿主、HTTP、数据库或 Cloud 运行时依赖。

当前仓库采用轻量结构：

- `pkgs/errx/`: 对外共享的统一错误模型与通用错误码
- `scripts/agents/`: 统一门禁入口与最小必要检查
- `docs/`: agent 规则、需求基线与架构说明

设计原则：

- 只放共享错误能力，不放应用层运行时
- 共享包目录清晰稳定，便于被 `axion-agent` 和 `axion-bridge` 直接引用
- 脚本和文档规模与仓库复杂度匹配
