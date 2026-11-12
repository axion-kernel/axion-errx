# 文档目录

- `agents/`: AI agent 执行约定与仓库勘误
- `arch/`: 仓库架构与目录说明
- `code-style/`: 代码风格
- `spec/`: 需求基线与包职责

## CI Runner

`.github/workflows/ci.yml` 使用 `ccf-linux-01-kernel-docker`，配置为
`runs-on: [self-hosted, Linux, X64, axion-kernel, image-builder, ccf, amd64]`，执行完整门禁与 race 测试。
公共仓库还需管理员允许访问 `Default` runner group，详见 [项目群合同](https://github.com/axion-kernel/axion-agent-docs/blob/main/engineering/ci/runner/runner-configuration.md)。
