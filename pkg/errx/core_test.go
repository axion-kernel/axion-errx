package errx_test

import (
	"testing"

	"github.com/axion-kernel/axion-errx/pkg/errx"
)

// TestCoreErrorCodes 验证全部共享错误名称与稳定码的对应关系。
//
// 要做什么：
// - 防止已发布的共享错误身份发生名称或数值漂移。
//
// 具体怎么做：
// - 为各共享类型列出明确的预期名称和编码，逐项运行并行子测试。
//
// 输出什么：
// - 每个类型的 Name 和 Code 均与固定预期相同。
func TestCoreErrorCodes(t *testing.T) {
	t.Parallel()

	// 测试准备：显式列出共享错误的发布名称与编码。
	cases := []struct {
		name string
		typ  *errx.Type
		code int
	}{
		{name: "internal_error", typ: errx.InternalError, code: 1000},
		{name: "illegal_argument", typ: errx.IllegalArgument, code: 1001},
		{name: "unauthorized", typ: errx.Unauthorized, code: 1002},
		{name: "forbidden", typ: errx.Forbidden, code: 1003},
		{name: "not_found", typ: errx.NotFound, code: 1004},
		{name: "conflict", typ: errx.Conflict, code: 1005},
		{name: "data_unavailable", typ: errx.DataUnavailable, code: 1006},
		{name: "rejected_operation", typ: errx.RejectedOperation, code: 1007},
		{name: "unsupported_operation", typ: errx.UnsupportedOperation, code: 1008},
		{name: "illegal_state", typ: errx.IllegalState, code: 1009},
		{name: "illegal_format", typ: errx.IllegalFormat, code: 1010},
		{name: "external_error", typ: errx.ExternalError, code: 1011},
		{name: "initialization_failed", typ: errx.InitializationFailed, code: 1012},
		{name: "method_not_allowed", typ: errx.MethodNotAllowed, code: 1013},
		{name: "bad_request_body", typ: errx.BadRequestBody, code: 1014},
		{name: "upstream_disconnected", typ: errx.UpstreamDisconnected, code: 1015},
		{name: "upstream_quota_exhausted", typ: errx.UpstreamQuotaExhausted, code: 1016},
		{name: "execution_skipped", typ: errx.ExecutionSkipped, code: 1017},
	}

	// 测试交互：逐个类型读取公开身份。
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			// 验证答案：名称与编码分别匹配本用例的固定预期。
			if tc.typ.Name() != tc.name {
				t.Fatalf("unexpected type name: got %q want %q", tc.typ.Name(), tc.name)
			}
			if tc.typ.Code() != tc.code {
				t.Fatalf("unexpected code: got %d want %d", tc.typ.Code(), tc.code)
			}
		})
	}

	// 测试收尾：仅查询进程级共享注册项，无需释放资源。
}

// TestCoreErrorsCrossMatchWithIsOfType 验证共享类型之间不会交叉匹配。
//
// 要做什么：
// - 确认错误类型判别只匹配产生该错误的注册类型。
//
// 具体怎么做：
// - 逐个共享类型构造错误，再与所有共享类型进行两两判别。
//
// 输出什么：
// - 只有来源类型与目标类型指针相同的组合返回 true。
func TestCoreErrorsCrossMatchWithIsOfType(t *testing.T) {
	t.Parallel()

	// 测试准备：收集需要两两比较的全部共享类型。
	cases := []struct {
		name string
		typ  *errx.Type
	}{
		{name: "internal_error", typ: errx.InternalError},
		{name: "illegal_argument", typ: errx.IllegalArgument},
		{name: "unauthorized", typ: errx.Unauthorized},
		{name: "forbidden", typ: errx.Forbidden},
		{name: "not_found", typ: errx.NotFound},
		{name: "conflict", typ: errx.Conflict},
		{name: "data_unavailable", typ: errx.DataUnavailable},
		{name: "rejected_operation", typ: errx.RejectedOperation},
		{name: "unsupported_operation", typ: errx.UnsupportedOperation},
		{name: "illegal_state", typ: errx.IllegalState},
		{name: "illegal_format", typ: errx.IllegalFormat},
		{name: "external_error", typ: errx.ExternalError},
		{name: "initialization_failed", typ: errx.InitializationFailed},
		{name: "method_not_allowed", typ: errx.MethodNotAllowed},
		{name: "bad_request_body", typ: errx.BadRequestBody},
		{name: "upstream_disconnected", typ: errx.UpstreamDisconnected},
		{name: "upstream_quota_exhausted", typ: errx.UpstreamQuotaExhausted},
		{name: "execution_skipped", typ: errx.ExecutionSkipped},
	}

	// 测试交互：为每个来源构造错误，再遍历所有判别目标。
	for _, source := range cases {
		source := source
		t.Run(source.name, func(t *testing.T) {
			t.Parallel()

			err := source.typ.New("test message")
			for _, target := range cases {
				target := target

				// 验证答案：判别结果必须与来源和目标的类型指针相等关系一致。
				got := errx.IsOfType(err, target.typ)
				want := source.typ == target.typ
				if got != want {
					t.Fatalf("unexpected IsOfType result: source=%s target=%s got=%t want=%t", source.name, target.name, got, want)
				}
			}
		})
	}

	// 测试收尾：构造的错误均为内存对象，没有外部资源。
}

// TestNotImplementedAliasesUnsupportedOperation 验证未实现类型保留别名身份。
//
// 要做什么：
// - 防止 NotImplemented 变成独立注册的错误类型。
//
// 具体怎么做：
// - 直接比较 NotImplemented 和 UnsupportedOperation 的类型指针。
//
// 输出什么：
// - 两个公开变量必须引用同一个类型对象。
func TestNotImplementedAliasesUnsupportedOperation(t *testing.T) {
	t.Parallel()

	// 测试准备：使用包初始化时注册的两个公开变量。
	//
	// 测试交互：比较两个变量持有的类型对象。
	//
	// 验证答案：别名保持相同的指针身份。
	if errx.NotImplemented != errx.UnsupportedOperation {
		t.Fatal("expected not implemented to alias unsupported operation")
	}

	// 测试收尾：仅比较共享对象，没有额外资源。
}

// TestNotImplementedMatchesUnsupportedOperationWithIsOfType 验证别名构造的错误可双向判别。
//
// 要做什么：
// - 确认通过未实现别名创建的错误也属于不支持操作类型。
//
// 具体怎么做：
// - 使用 NotImplemented 创建错误，再分别以原类型和别名判别。
//
// 输出什么：
// - 两次 IsOfType 查询都返回 true。
func TestNotImplementedMatchesUnsupportedOperationWithIsOfType(t *testing.T) {
	t.Parallel()

	// 测试准备：选择未实现别名作为错误来源。
	//
	// 测试交互：创建错误并分别查询原类型与别名。
	err := errx.NotImplemented.New("feature pending")

	// 验证答案：原类型和别名对该错误都成立。
	if !errx.IsOfType(err, errx.UnsupportedOperation) {
		t.Fatal("expected not implemented error to match unsupported operation")
	}
	if !errx.IsOfType(err, errx.NotImplemented) {
		t.Fatal("expected not implemented error to match not implemented alias")
	}

	// 测试收尾：错误对象只存在于内存中。
}
