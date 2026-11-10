# 仓库架构

当前统一 module 为 `github.com/axion-kernel/axion-errx`，公共包入口固定为 `pkg/errx`。
仓库由原 `axion-box/axion-errx` 直接迁入，保留原提交历史、实现、错误码和测试；仅迁移 Go module 与消费路径。
Axion Kernel 项目群后续所有 Go 项目统一依赖本仓，已有消费方在迁移时切换，禁止同时加载旧 module 和新 module。
错误模型仍保持纯 Go 基础库职责，不引入宿主、HTTP、数据库或 Cloud 运行时依赖。

当前仓库采用轻量结构：

- `pkg/errx/`: 对外共享的统一错误模型与通用错误码
- `scripts/agents/`: 统一门禁入口与最小必要检查
- `docs/`: agent 规则、需求基线与架构说明

设计原则：

- 只放共享错误能力，不放应用层运行时
- 共享包目录清晰稳定，便于被 `axion-agent` 和 `axion-bridge` 直接引用
- 脚本和文档规模与仓库复杂度匹配


## 显式诊断与可携带元数据

增加 Type.NewWithOptions：使用共享 *errx.Error 保存调用方指定的准确 Message、可选 Cause 和可选 JSON Metadata；不自动把 cause 文本追加到此入口的 Message。原 New/Wrap/WrapError 的行为和 1000–1999 共享类型保持不变。Metadata 在构造和读取时复制，结构化 Attrs 仅在存在 Metadata 时增加该字段。领域不再为保存原协议诊断或原生执行字段实现另一套 error 类型。此库不注册 Agent 私有码，不解释 Metadata 的产品内容。

Options 的 Payload 仅保留进程内原值身份，不写入 Attrs；通用库不解释该值，也不承诺深复制任意对象。调用方为协议 DTO 提供独立所有权，真正任意拒绝值可以有意保留对象 identity。

## 原因树查询

FindIf 按标准 Unwrap 的单原因或多原因树深度优先查找，先访问当前节点，再按分支声明顺序访问子节点。
调用方只提供当前节点匹配条件，协议码与载荷含义仍由其 owner 解释。查询跳过空值和类型化空值，
最多访问 4096 个节点；不渲染错误文本，也不复制错误。领域库复用此入口，不再维护递归查码器。

## 错误自带 HTTP 状态

`NewType(name, code, status)` 在注册错误身份时声明 400–599 的 HTTP 状态，原两参数调用默认 500。`Type.HttpStatus()` 与 `Error.HttpStatus()` 读取该状态；`Options.HttpStatus` 可显式覆盖当前实例。`errx.HttpStatus(err)` 通过统一的有界原因树查询支持标准包装和联合错误，未找到 Errx 实例时统一 500，不采信普通错误的同名方法。最外层有效声明优先；`WrapError` 保留状态，显式 `Type.Wrap/WrapPrivate` 重新分类并采用外层类型的状态。

共享错误在本仓声明状态，领域类型在 owner 仓声明，HTTP 宿主不再按 code 维护 switch。状态不新增到 Attrs、元数据或响应正文。HTTP handler、安全消息和序列化仍归各边界。
