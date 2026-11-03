# errx

Axion Kernel 项目群统一使用的公共错误库；后续所有 Go 项目使用本仓 module，保留原 `axion-box/axion-errx` 的提交历史与实现。

`errx` 是 `axion` 系列项目通用的结构化错误包，用来统一表达稳定错误码、错误类型、错误消息、原因链和创建时堆栈。

对外共享包路径：

- `github.com/axion-kernel/axion-errx/pkgs/errx`

## 基本用法

`ExecutionSkipped`（`execution_skipped` / `1017`）表示依赖服务明确要求跳过本次执行，
例如计划时间窗口没有预计算数据。消费端应生成正常的跳过结果，而非重试或记作执行失败。

### 创建一个新的 err

```go
package main

import (
	"fmt"

	"github.com/axion-kernel/axion-errx/pkgs/errx"
)

func main() {
	err := errx.IllegalArgument.New("invalid user id: %d", 42)

	fmt.Println(err.Error())       // illegal_argument: invalid user id: 42
	fmt.Println(err.Type().Name()) // illegal_argument
	fmt.Println(err.Code())        // 1001
	fmt.Println(err.Message())     // invalid user id: 42
	fmt.Println(err.Stacktrace() != "") // true
}
```

`Type.New(...)` 会创建一个 `*errx.Error`，并在创建时自动捕获堆栈。

### Wrap 一个旧 err

日常场景优先用 `errx.WrapError(...)`。它会自动区分普通 `error` 和已有的 `errx.Error`：

- 普通 `error`：按 `external_error` 包装，并生成 `new: old.Error()`
- 已有 `errx.Error`：沿用原来的 `Type` 再包装一层，并生成 `new: old.Message()`

```go
package main

import (
	"errors"
	"fmt"

	"github.com/axion-kernel/axion-errx/pkgs/errx"
)

func main() {
	base := errors.New("dial tcp timeout")
	err := errx.WrapError(base, "cloud request failed")

	fmt.Println(err.Error())          // external_error: cloud request failed: dial tcp timeout
	fmt.Println(err.Type().Name())    // external_error
	fmt.Println(err.Message())        // cloud request failed: dial tcp timeout
	fmt.Println(err.Cause() == base)  // true
	fmt.Println(err.Stacktrace() != "") // true
}
```

如果传入的已经是 `errx.Error`，会复用原类型并拼接消息：

```go
base := errx.NotFound.New("profile missing")
err := errx.WrapError(base, "while refreshing cache")

fmt.Println(err.Type().Name()) // not_found
fmt.Println(err.Message())     // while refreshing cache: profile missing
```

`Type.Wrap(...)` 仍然会保留原始错误作为 `cause`，同时为外层错误补充新的类型、消息和堆栈。它更适合“调用方明确知道要强制指定哪一种错误类型”的场景。

如果包装时不给新消息，会回退到原始错误消息：

```go
base := errors.New("parse body failed")
err := errx.WrapError(base, "")

fmt.Println(err.Message()) // parse body failed
```

### 从 error 链里读取 errx 信息

```go
package main

import (
	"errors"
	"fmt"

	"github.com/axion-kernel/axion-errx/pkgs/errx"
)

func main() {
	base := errors.New("record missing")
	err := errx.WrapError(errx.NotFound.Wrap(base, "load profile failed"), "while reading profile view")

	fmt.Println(errx.Code(err))                     // 1004
	fmt.Println(errx.IsOfType(err, errx.NotFound)) // true

	if ex := errx.Cast(err); ex != nil {
		fmt.Println(ex.Type().Name())     // not_found
		fmt.Println(ex.Message())         // while reading profile view: load profile failed: record missing
		fmt.Println(ex.Stacktrace() != "") // true
	}
}
```

常用 helper：

- `errx.WrapError(err, ...)`：日常包装错误的推荐入口
- `errx.Cast(err)`：从错误链里取出第一条 `*errx.Error`
- `errx.Code(err)`：读取稳定错误码；普通 `error` 会回退为 `internal_error`
- `errx.IsOfType(err, t)`：判断错误链里是否包含指定类型
- `errx.Attrs(err)`：导出结构化字段，便于日志或协议层继续封装

补充说明：

- `Message()` 返回不带类型前缀的消息体
- `Error()` 返回适合直接打印的 `type: message`

## 预定义错误

`errx` 内置了一组共享错误类型，供多个仓库复用。当前共享错误码段为 `1000~1999`，这里已经占用：

| Type | Code |
| --- | ---: |
| `internal_error` | `1000` |
| `illegal_argument` | `1001` |
| `unauthorized` | `1002` |
| `forbidden` | `1003` |
| `not_found` | `1004` |
| `conflict` | `1005` |
| `data_unavailable` | `1006` |
| `rejected_operation` | `1007` |
| `unsupported_operation` | `1008` |
| `illegal_state` | `1009` |
| `illegal_format` | `1010` |
| `external_error` | `1011` |
| `initialization_failed` | `1012` |
| `method_not_allowed` | `1013` |
| `bad_request_body` | `1014` |
| `upstream_disconnected` | `1015` |
| `upstream_quota_exhausted` | `1016` |

其中：

- `errx.NotImplemented` 是 `errx.UnsupportedOperation` 的别名
- `errx.UpstreamDisconnected` 只表示外部系统未发送可解释错误终态便断开；上游已经发送的错误应保留其原始分类
- `errx.UpstreamQuotaExhausted` 只表示可用上游账号池配额已经耗尽；它不表示用户积分不足、普通平台过载、未知限流或瞬时网络失败
- 这些类型适合放在共享层复用，不适合承载某个业务仓库私有语义

## 定义业务错误

在业务仓库里，可以基于 `errx.NewType(name, code)` 定义自己的稳定错误类型：

```go
package billing

import "github.com/axion-kernel/axion-errx/pkgs/errx"

var InvoiceClosed = errx.NewType("invoice_closed", 2100)
```

然后像预定义类型一样继续使用：

```go
func CloseInvoice() error {
	return InvoiceClosed.New("invoice %s has already been closed", "INV-001")
}
```

约定：

- `NewType` 统一使用 `errx.NewType("name", code)` 形式
- `name` 和 `code` 在同一进程内必须唯一，重复注册会 `panic`
- `1000~1999` 是共享层保留错误码段；业务私有错误码应在各自仓库维护

## 已知下游业务错误码范围

下游项目必须在自己的仓库内维护并自动校验业务错误码，同时在这里登记所占范围，避免不同项目复用同一段编号：

| Project | Repository | Business error code range |
| --- | --- | ---: |
| `axion-cloud-v2` | `github.com/axion-box/axion-cloud-v2` | `2000~2999` |
| `axion-agent-v2` | `github.com/axion-box/axion-agent-v2` | `3000~3999` |
| `glenclaw-upgrade` | `github.com/axion-box/glenclaw-upgrade` | `4000~4999` |

新增项目或调整范围时，必须先更新本表；已登记范围不得互相重叠，也不得占用共享层保留的 `1000~1999`。

`glenclaw-upgrade` 当前稳定领域错误如下；具体类型仍在业务仓库定义，避免共享库反向依赖设备升级实现：

| Type | Code | Meaning |
| --- | ---: | --- |
| `upgrade_current_plan_changed` | `4001` | 下载指令排队期间 current 已切换，调用方应刷新后重试。 |
| `upgrade_current_plan_locked` | `4002` | current 已进入下载或安装阶段，最新计划只能保留为 latest。 |
| `upgrade_no_current_update` | `4003` | 当前没有可供目标指令操作的更新。 |
| `upgrade_download_state_conflict` | `4004` | 下载指令与队列当前状态冲突。 |
| `upgrade_downloads_not_ready` | `4005` | 镜像尚未全部下载并验证，不能安装。 |
| `upgrade_installation_active` | `4006` | 安装已经激活，不允许在线取消。 |

## 开发

- 单元测试：`bash scripts/run_unit_tests.sh`
- 门禁：`bash scripts/run_gates.sh`

`Type.WrapPrivate(cause, message, args...)` 保留敏感原因供 errors.Is/As 和显式 Cause 检查，Error、Message、Attrs 不展示其文本。调用方负责提供安全摘要。


使用明确的安全诊断并保留真实原因时，调用 `Type.NewWithOptions(errx.Options{Message: safeText, Cause: cause, Metadata: jsonData})`。此入口不追加 cause 文本，公开 Attrs 也不输出其内容；`errors.Is/As` 仍可检查原因。`Metadata()` 返回 JSON 缓冲区副本。可选 `Payload` 只保留进程内原值身份，不自动进入日志或序列化，可变数据由绑定方隔离。原 `New`、`Wrap`、`WrapError` 语义保持不变。

## CI Runner

CI 统一使用组织 self-hosted runner `ccf-linux-01-kernel-docker`（Linux x64，runner group `Default`）。
`runs-on: [self-hosted, Linux, X64, axion-kernel, image-builder, ccf, amd64]`；配置、环境要求与排队排查见 [项目群 CI Runner 合同](https://github.com/axion-kernel/axion-agent-docs/blob/main/development/ci-runner.md)。
