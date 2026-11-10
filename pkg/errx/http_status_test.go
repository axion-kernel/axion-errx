package errx_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/axion-kernel/axion-errx/pkg/errx"
)

// statusFailure 模拟由外部协议拥有的状态声明。
type statusFailure int

// Error 返回测试协议的固定诊断。
//
// 要做什么：
// - 让状态夹具满足标准错误接口。
//
// 具体怎么做：
// - 返回固定文本，状态不依赖文本解析。
//
// 输出什么：
// - 返回用于包装的诊断文本。
func (statusFailure) Error() string { return "private upstream diagnostic" }

// HttpStatus 返回协议夹具自身的状态。
//
// 要做什么：
// - 模拟远端错误与 Errx 共享状态读取合同。
//
// 具体怎么做：
// - 将夹具值直接作为状态返回。
//
// 输出什么：
// - 返回调用方给定的状态，包含用于验证回退的无效值。
func (s statusFailure) HttpStatus() int { return int(s) }

// TestSharedHttpStatus 验证每种共享错误自带固定状态。
//
// 要做什么：
// - 防止宿主删除映射后发生公共错误语义漂移。
//
// 具体怎么做：
// - 遍历全部公共类型，核对类型、实例和通用查询结果。
//
// 输出什么：
// - 公共错误的三种读取路径得到相同的预期状态。
func TestSharedHttpStatus(t *testing.T) {
	// 测试准备：列出全部稳定公共错误的状态合同。
	cases := []struct {
		typ  *errx.Type
		want int
	}{
		{errx.InternalError, 500}, {errx.IllegalArgument, 400}, {errx.Unauthorized, 401},
		{errx.Forbidden, 403}, {errx.NotFound, 404}, {errx.Conflict, 409},
		{errx.DataUnavailable, 503}, {errx.RejectedOperation, 409}, {errx.UnsupportedOperation, 501},
		{errx.IllegalState, 409}, {errx.IllegalFormat, 400}, {errx.ExternalError, 502},
		{errx.InitializationFailed, 500}, {errx.MethodNotAllowed, 405}, {errx.BadRequestBody, 400},
		{errx.UpstreamDisconnected, 502}, {errx.UpstreamQuotaExhausted, 429}, {errx.ExecutionSkipped, 409},
	}
	// 测试交互与验证答案：每个类型创建实例后核对所有读取入口。
	for _, test := range cases {
		failure := test.typ.New("private message")
		if test.typ.HttpStatus() != test.want || failure.HttpStatus() != test.want || errx.HttpStatus(failure) != test.want {
			t.Fatalf("%s: type=%d instance=%d query=%d want=%d", test.typ.Name(), test.typ.HttpStatus(), failure.HttpStatus(), errx.HttpStatus(failure), test.want)
		}
	}
	// 测试收尾：仅使用进程内错误，无需释放资源。
}

// TestHttpStatusPreservesWrappingAndOwner 验证组合宿主不丢失错误所属方的状态。
//
// 要做什么：
// - 覆盖领域类型、实例覆盖、透明包装、显式改分类和多原因优先级。
//
// 具体怎么做：
// - 构造领域错误与远端错误，经过标准包装和 Errx 包装后查询。
//
// 输出什么：
// - 状态按最外层声明确定，错误码和原因链保持原有合同。
func TestHttpStatusPreservesWrappingAndOwner(t *testing.T) {
	// 测试准备：领域类型声明超时，远端实例声明限流。
	owner := errx.NewType("http_status_owner_test", 991201, 504)
	legacy := errx.NewType("http_status_legacy_test", 991202)
	remote := errx.ExternalError.NewWithOptions(errx.Options{Message: "safe", Cause: context.Canceled, HttpStatus: 429})
	wrapped := errx.WrapError(fmt.Errorf("context: %w", remote), "more context")
	// 测试交互：按实际产品组合方式构造原因树。
	cases := []struct {
		failure error
		want    int
	}{
		{owner.New("timeout"), 504}, {legacy.New("unknown"), 500},
		{wrapped, 429}, {errx.Forbidden.Wrap(remote, "denied"), 403},
		{errx.InternalError.WrapPrivate(remote, "hidden"), 500},
		{fmt.Errorf("wrapped: %w", statusFailure(418)), 418},
		{errx.WrapError(statusFailure(418), "protocol"), 418},
		{errors.Join(errors.New("plain"), owner.New("timeout")), 504},
		{errors.Join(remote, owner.New("timeout")), 429},
		{fmt.Errorf("cancel: %w", context.Canceled), 408},
		{errx.WrapError(context.DeadlineExceeded, "deadline"), 408},
		{nil, 500}, {(*errx.Error)(nil), 500}, {errors.New("plain"), 500},
		{statusFailure(200), 500}, {statusFailure(600), 500}, {new(errx.Error), 500},
	}
	// 验证答案：状态不依赖错误码查表，透明包装保留原身份和原因。
	for index, test := range cases {
		if got := errx.HttpStatus(test.failure); got != test.want {
			t.Fatalf("case %d: got=%d want=%d", index, got, test.want)
		}
	}
	if wrapped.Code() != remote.Code() || !errors.Is(wrapped, remote) || wrapped.HttpStatus() != 429 || (*errx.Type)(nil).HttpStatus() != 500 || new(errx.Type).HttpStatus() != 500 {
		t.Fatal("wrapping changed identity, status or cause")
	}
	// 测试收尾：注册项随进程结束，无外部资源。
}

// TestHttpStatusRejectsInvalidDeclarations 验证非法状态在注册和构造时被拒绝。
//
// 要做什么：
// - 防止错误产生成功状态或无效响应。
//
// 具体怎么做：
// - 对越界状态及多个状态执行构造，捕获预期 panic。
//
// 输出什么：
// - 所有非法声明都被拒绝，合法边界可以读取。
func TestHttpStatusRejectsInvalidDeclarations(t *testing.T) {
	// 测试准备：保存需要拒绝的注册和实例构造操作。
	invalid := []func(){
		func() { errx.NewType("invalid_status", 991203, 399) },
		func() { errx.NewType("invalid_status", 991203, 600) },
		func() { errx.NewType("invalid_status", 991203, 0) },
		func() { errx.NewType("invalid_status", 991203, 400, 500) },
		func() { errx.ExternalError.NewWithOptions(errx.Options{HttpStatus: 200}) },
		func() { errx.ExternalError.NewWithOptions(errx.Options{HttpStatus: 600}) },
	}
	// 测试交互与验证答案：每个非法构造都必须触发拒绝。
	for index, create := range invalid {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("case %d accepted invalid status", index)
				}
			}()
			create()
		}()
	}
	for _, status := range []int{400, 599} {
		if got := errx.ExternalError.NewWithOptions(errx.Options{HttpStatus: status}).HttpStatus(); got != status {
			t.Fatalf("boundary got=%d want=%d", got, status)
		}
	}
	// 测试收尾：非法注册未污染注册表，可继续注册同名同码。
	if got := errx.NewType("invalid_status", 991203, 500).HttpStatus(); got != 500 {
		t.Fatal(got)
	}
}
