package errx_test

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"

	"github.com/axion-kernel/axion-errx/pkg/errx"
)

// TestNewErrorCarriesTypeMessageCodeAndStacktrace 验证新错误携带完整诊断身份。
//
// 要做什么：
// - 检查构造器保存类型、格式化消息、编码和调用栈，且不凭空产生原因。
//
// 具体怎么做：
// - 以带数值参数的消息创建 IllegalArgument，逐项读取公开属性。
//
// 输出什么：
// - 消息参数正确展开，类型编码为 1001，原因为空且调用栈非空。
func TestNewErrorCarriesTypeMessageCodeAndStacktrace(t *testing.T) {
	t.Parallel()

	// 测试准备：选择参数错误类型与数值消息参数。
	//
	// 测试交互：通过类型构造带调用栈的新错误。
	err := errx.IllegalArgument.New("invalid user id: %d", 42)

	// 验证答案：逐项检查诊断身份、消息、原因和调用栈。
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != "illegal_argument: invalid user id: 42" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	if err.Type() != errx.IllegalArgument {
		t.Fatalf("unexpected type: got %v", err.Type())
	}
	if err.Code() != 1001 {
		t.Fatalf("unexpected code: got %d", err.Code())
	}
	if err.Message() != "invalid user id: 42" {
		t.Fatalf("unexpected message: %q", err.Message())
	}
	if err.Cause() != nil {
		t.Fatalf("unexpected cause: %v", err.Cause())
	}
	if err.Unwrap() != nil {
		t.Fatalf("unexpected unwrap value: %v", err.Unwrap())
	}
	if err.Stacktrace() == "" {
		t.Fatal("expected stacktrace")
	}

	// 测试收尾：错误对象仅占用内存。
}

// TestWrapPreservesCauseAndSupportsHelpers 验证普通包装保留原因链并兼容辅助查询。
//
// 要做什么：
// - 确认类型包装同时保留标准原因身份与结构化查询能力。
//
// 具体怎么做：
// - 包装普通超时文本错误，检查标准错误链、类型判别、编码与消息。
//
// 输出什么：
// - 原因对象不变，消息由新上下文与原因文本依次组成。
func TestWrapPreservesCauseAndSupportsHelpers(t *testing.T) {
	t.Parallel()

	// 测试准备：创建具有独立身份的普通错误。
	base := errors.New("dial tcp timeout")

	// 测试交互：使用外部错误类型包装原因。
	err := errx.ExternalError.Wrap(base, "cloud request failed")

	// 验证答案：标准原因链与结构化辅助查询都保持预期语义。
	if !errors.Is(err, base) {
		t.Fatal("expected wrapped cause in error chain")
	}
	if err.Cause() != base {
		t.Fatal("expected cause to be preserved")
	}
	if err.Unwrap() != base {
		t.Fatal("expected unwrap to expose cause")
	}
	if errx.Cast(err) != err {
		t.Fatal("expected cast to return wrapped errx error")
	}
	if !errx.IsOfType(err, errx.ExternalError) {
		t.Fatal("expected errx.IsOfType to match")
	}
	if errx.Code(err) != errx.ExternalError.Code() {
		t.Fatalf("unexpected code from helper: got %d", errx.Code(err))
	}
	if err.Message() != "cloud request failed: dial tcp timeout" {
		t.Fatalf("unexpected message: %q", err.Message())
	}
	if err.Error() != "external_error: cloud request failed: dial tcp timeout" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}

	// 测试收尾：没有建立实际网络连接。
}

// TestWrapFallsBackToCauseMessageWhenMessageEmpty 验证空上下文保留原因消息。
//
// 要做什么：
// - 防止调用方未提供消息时丢失原始诊断。
//
// 具体怎么做：
// - 用空消息包装普通解析错误，再读取包装后的消息本体。
//
// 输出什么：
// - 返回原因消息，不增加多余分隔符。
func TestWrapFallsBackToCauseMessageWhenMessageEmpty(t *testing.T) {
	t.Parallel()

	// 测试准备：创建解析失败原因。
	base := errors.New("parse body failed")

	// 测试交互：使用空上下文包装原因。
	err := errx.BadRequestBody.Wrap(base, "")

	// 验证答案：消息回退到原原因文本。
	if err.Message() != "parse body failed" {
		t.Fatalf("unexpected fallback message: %q", err.Message())
	}

	// 测试收尾：没有创建解析器或外部资源。
}

// TestWrapUsesStructuredMessageWhenCauseIsErrxError 验证结构化原因不重复带入类型名称。
//
// 要做什么：
// - 确认显式更换外层类型时，消息只引用内层错误的消息本体。
//
// 具体怎么做：
// - 创建 NotFound 原因，再以 Conflict 包装并读取消息、显示文本和原因。
//
// 输出什么：
// - 显示文本仅使用 Conflict 类型名称，原因仍为原 NotFound 对象。
func TestWrapUsesStructuredMessageWhenCauseIsErrxError(t *testing.T) {
	t.Parallel()

	// 测试准备：创建具有独立类型的结构化原因。
	base := errx.NotFound.New("record missing")

	// 测试交互：显式改用冲突类型包装原因。
	err := errx.Conflict.Wrap(base, "save profile failed")

	// 验证答案：消息没有重复类型名称，原因身份保持原样。
	if err.Message() != "save profile failed: record missing" {
		t.Fatalf("unexpected message: %q", err.Message())
	}
	if err.Error() != "conflict: save profile failed: record missing" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	if err.Cause() != base {
		t.Fatalf("unexpected cause: %v", err.Cause())
	}

	// 测试收尾：两个错误均为内存对象。
}

// TestWrapReturnsNilForNilCause 验证类型包装不会把空原因变成错误。
//
// 要做什么：
// - 保持没有原始错误时的成功语义。
//
// 具体怎么做：
// - 向 ExternalError.Wrap 传入 nil 原因和非空消息。
//
// 输出什么：
// - 包装结果必须为 nil。
func TestWrapReturnsNilForNilCause(t *testing.T) {
	t.Parallel()

	// 测试准备：提供空原因与应被忽略的消息。
	//
	// 测试交互：调用类型级包装方法。
	//
	// 验证答案：非空消息不能为成功结果制造错误。
	if got := errx.ExternalError.Wrap(nil, "ignored"); got != nil {
		t.Fatalf("expected nil wrap result, got %#v", got)
	}

	// 测试收尾：包装过程没有创建错误对象。
}

// TestWrapErrorReturnsNilForNilCause 验证通用包装保留空原因。
//
// 要做什么：
// - 确认通用错误包装入口可以直接处理成功结果。
//
// 具体怎么做：
// - 向 WrapError 传入 nil 与非空诊断上下文。
//
// 输出什么：
// - 结果仍为 nil，不创建 ExternalError。
func TestWrapErrorReturnsNilForNilCause(t *testing.T) {
	t.Parallel()

	// 测试准备：选择空错误输入。
	//
	// 测试交互：调用通用包装入口。
	//
	// 验证答案：忽略诊断上下文并返回 nil。
	if got := errx.WrapError(nil, "ignored"); got != nil {
		t.Fatalf("expected nil wrap result, got %#v", got)
	}

	// 测试收尾：没有创建可释放资源。
}

// TestWrapErrorWrapsPlainErrorAsExternalError 验证普通错误使用统一外部错误类型。
//
// 要做什么：
// - 确认通用包装为普通错误补充结构化身份并保留原因链。
//
// 具体怎么做：
// - 创建普通连接超时错误，再通过 WrapError 增加请求上下文。
//
// 输出什么：
// - 返回 ExternalError，消息按上下文在前的顺序组合，原始原因可由 errors.Is 找到。
func TestWrapErrorWrapsPlainErrorAsExternalError(t *testing.T) {
	t.Parallel()

	// 测试准备：以普通错误模拟连接失败。
	base := errors.New("dial tcp timeout")

	// 测试交互：通过通用入口包装请求上下文。
	err := errx.WrapError(base, "cloud request failed")

	// 验证答案：检查默认类型、组合文本与原始原因身份。
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if err.Type() != errx.ExternalError {
		t.Fatalf("unexpected type: got %v", err.Type())
	}
	if err.Message() != "cloud request failed: dial tcp timeout" {
		t.Fatalf("unexpected message: %q", err.Message())
	}
	if err.Error() != "external_error: cloud request failed: dial tcp timeout" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	if err.Cause() != base {
		t.Fatalf("unexpected cause: %v", err.Cause())
	}
	if !errors.Is(err, base) {
		t.Fatal("expected wrapped cause in error chain")
	}

	// 测试收尾：本用例不发起网络请求。
}

// TestWrapErrorReusesExistingErrxTypeAndAppendsMessage 验证通用包装沿用类型并补充外层上下文。
//
// 要做什么：
// - 确认已有结构化错误再次包装后仍保留类型与完整原因链。
//
// 具体怎么做：
// - 先将普通原因包装为 NotFound，再通过 WrapError 增加缓存刷新上下文。
//
// 输出什么：
// - 返回新的 NotFound 对象，消息从外向内连接，两个原始原因层级均可查找且调用栈非空。
func TestWrapErrorReusesExistingErrxTypeAndAppendsMessage(t *testing.T) {
	t.Parallel()

	// 测试准备：构造普通原因与带加载上下文的结构化错误。
	baseCause := errors.New("record missing")
	base := errx.NotFound.Wrap(baseCause, "load profile failed")

	// 测试交互：以缓存刷新上下文再次包装。
	err := errx.WrapError(base, "while refreshing cache")

	// 验证答案：新对象沿用原类型，保留全部原因层级并捕获新的调用栈。
	if err == nil {
		t.Fatal("expected wrapped error")
	}
	if err == base {
		t.Fatal("expected a newly created errx error")
	}
	if err.Type() != errx.NotFound {
		t.Fatalf("unexpected type: got %v", err.Type())
	}
	if err.Message() != "while refreshing cache: load profile failed: record missing" {
		t.Fatalf("unexpected message: %q", err.Message())
	}
	if err.Error() != "not_found: while refreshing cache: load profile failed: record missing" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}
	if err.Cause() != base {
		t.Fatalf("unexpected cause: %v", err.Cause())
	}
	if !errors.Is(err, base) {
		t.Fatal("expected original errx error in error chain")
	}
	if !errors.Is(err, baseCause) {
		t.Fatal("expected original root cause in error chain")
	}
	if err.Stacktrace() == "" {
		t.Fatal("expected stacktrace")
	}

	// 测试收尾：包装链只保存在内存中。
}

// TestWrapErrorKeepsOriginalMessageWhenNewMessageEmpty 验证空的新消息不覆盖原始诊断。
//
// 要做什么：
// - 保持结构化错误在无新增上下文时的类型与消息。
//
// 具体怎么做：
// - 创建 IllegalState，再使用空字符串调用通用包装入口。
//
// 输出什么：
// - 新错误仍为 IllegalState，消息与显示文本不增加分隔符。
func TestWrapErrorKeepsOriginalMessageWhenNewMessageEmpty(t *testing.T) {
	t.Parallel()

	// 测试准备：创建已有状态诊断。
	base := errx.IllegalState.New("worker already started")

	// 测试交互：不提供额外上下文进行包装。
	err := errx.WrapError(base, "")

	// 验证答案：原消息和类型在新包装中保留。
	if err.Message() != "worker already started" {
		t.Fatalf("unexpected message: %q", err.Message())
	}
	if err.Type() != errx.IllegalState {
		t.Fatalf("unexpected type: got %v", err.Type())
	}
	if err.Error() != "illegal_state: worker already started" {
		t.Fatalf("unexpected error string: %q", err.Error())
	}

	// 测试收尾：本用例没有启动实际 worker。
}

// TestAttrsForStructuredPlainAndNilErrors 验证三类错误输入的诊断字段投影。
//
// 要做什么：
// - 确认结构化错误、普通错误和空输入均能提供固定基础字段。
//
// 具体怎么做：
// - 对结构化错误分别调用包函数和接收者方法，再检查普通错误与 nil 的回退字段。
//
// 输出什么：
// - 结构化身份与调用栈保留；普通错误和 nil 使用 InternalError，且不凭空补充原因或调用栈。
func TestAttrsForStructuredPlainAndNilErrors(t *testing.T) {
	t.Parallel()

	// 测试准备：创建带固定消息的结构化错误。
	structured := errx.NotFound.New("record missing")

	// 测试交互：先从包级入口读取诊断字段。
	structuredAttrs := errx.Attrs(structured)

	// 验证答案：结构化字段完整保留类型、编码、消息和调用栈。
	if len(structuredAttrs) != 5 {
		t.Fatalf("unexpected attr count: got %d", len(structuredAttrs))
	}
	if structuredAttrs[0].Key != "error_type" || structuredAttrs[0].Value != "not_found" {
		t.Fatalf("unexpected type attr: %+v", structuredAttrs[0])
	}
	if structuredAttrs[1].Key != "error_code" || structuredAttrs[1].Value != 1004 {
		t.Fatalf("unexpected code attr: %+v", structuredAttrs[1])
	}
	if structuredAttrs[2].Key != "error_message" || structuredAttrs[2].Value != "record missing" {
		t.Fatalf("unexpected message attr: %+v", structuredAttrs[2])
	}
	if structuredAttrs[3].Key != "error_cause" || structuredAttrs[3].Value != "" {
		t.Fatalf("unexpected cause attr: %+v", structuredAttrs[3])
	}
	if structuredAttrs[4].Key != "error_stacktrace" || structuredAttrs[4].Value == "" {
		t.Fatalf("unexpected stacktrace attr: %+v", structuredAttrs[4])
	}

	// 对照接收者入口，确认两个入口提供相同的基础身份字段。
	receiverAttrs := structured.Attrs()
	if len(receiverAttrs) != 5 {
		t.Fatalf("unexpected receiver attr count: got %d", len(receiverAttrs))
	}
	if receiverAttrs[0] != structuredAttrs[0] || receiverAttrs[1] != structuredAttrs[1] || receiverAttrs[2] != structuredAttrs[2] {
		t.Fatalf("expected receiver attrs to match helper attrs: got %+v want %+v", receiverAttrs, structuredAttrs)
	}

	// 普通错误仅投影其消息，并采用统一内部错误身份。
	plain := errors.New("plain failure")
	plainAttrs := errx.Attrs(plain)
	if plainAttrs[0].Value != "internal_error" {
		t.Fatalf("unexpected fallback type: %+v", plainAttrs[0])
	}
	if plainAttrs[1].Value != 1000 {
		t.Fatalf("unexpected fallback code: %+v", plainAttrs[1])
	}
	if plainAttrs[2].Value != "plain failure" {
		t.Fatalf("unexpected fallback message: %+v", plainAttrs[2])
	}
	if plainAttrs[3].Value != "" {
		t.Fatalf("unexpected fallback cause: %+v", plainAttrs[3])
	}
	if plainAttrs[4].Value != "" {
		t.Fatalf("unexpected fallback stacktrace: %+v", plainAttrs[4])
	}

	// 空输入保留内部错误身份，其余诊断载荷均为空。
	nilAttrs := errx.Attrs(nil)
	if nilAttrs[0].Value != "internal_error" {
		t.Fatalf("unexpected nil type: %+v", nilAttrs[0])
	}
	if nilAttrs[1].Value != 1000 {
		t.Fatalf("unexpected nil code: %+v", nilAttrs[1])
	}
	if nilAttrs[2].Value != "" || nilAttrs[3].Value != "" || nilAttrs[4].Value != "" {
		t.Fatalf("unexpected nil attrs payload: %+v", nilAttrs)
	}

	// 测试收尾：字段切片均归当前用例持有。
}

// TestErrorHelpersHandleNilAndPlainErrors 验证访问方法与辅助函数的空值边界。
//
// 要做什么：
// - 确认空实例、空类型和普通错误不会被错误地解释为有效结构化类型。
//
// 具体怎么做：
// - 查询空 Error 的各属性，使用空 Type 创建错误，再检查 Cast、Code 与 IsOfType 的回退结果。
//
// 输出什么：
// - 访问方法返回约定零值，普通错误使用内部错误码，空目标类型不匹配。
func TestErrorHelpersHandleNilAndPlainErrors(t *testing.T) {
	t.Parallel()

	// 测试准备：声明空结构化错误指针。
	var err *errx.Error

	// 测试交互：逐项调用空接收者方法和包级辅助函数。
	//
	// 验证答案：每个访问入口遵守明确的零值或回退约定。
	if err.Error() != "" {
		t.Fatalf("unexpected nil Error() result: %q", err.Error())
	}
	if err.Message() != "" {
		t.Fatalf("unexpected nil Message() result: %q", err.Message())
	}
	if err.Cause() != nil {
		t.Fatalf("unexpected nil Cause() result: %v", err.Cause())
	}
	if err.Type() != nil {
		t.Fatalf("unexpected nil Type() result: %v", err.Type())
	}
	if err.Code() != 0 {
		t.Fatalf("unexpected nil Code() result: %d", err.Code())
	}
	if err.Stacktrace() != "" {
		t.Fatalf("unexpected nil Stacktrace() result: %q", err.Stacktrace())
	}
	if err.Unwrap() != nil {
		t.Fatalf("unexpected nil Unwrap() result: %v", err.Unwrap())
	}
	if got := err.Attrs(); len(got) != 5 {
		t.Fatalf("unexpected nil receiver attrs count: %d", len(got))
	}

	// 空类型仍可创建消息对象，但不会获得有效类型编码。
	var nilType *errx.Type
	errWithNilType := nilType.New("detached")
	if errWithNilType.Type() != nil {
		t.Fatalf("expected nil type, got %v", errWithNilType.Type())
	}
	if errWithNilType.Code() != 0 {
		t.Fatalf("expected zero code for nil type, got %d", errWithNilType.Code())
	}

	// 普通错误和空输入无法转换为结构化错误，类型判别也不能误匹配。
	if errx.Cast(nil) != nil {
		t.Fatalf("expected nil cast result, got %#v", errx.Cast(nil))
	}
	if errx.Cast(errors.New("plain")) != nil {
		t.Fatalf("expected plain error cast to be nil, got %#v", errx.Cast(errors.New("plain")))
	}
	if errx.Code(nil) != 0 {
		t.Fatalf("unexpected nil helper code: %d", errx.Code(nil))
	}
	if errx.Code(errors.New("plain")) != errx.InternalError.Code() {
		t.Fatalf("unexpected plain helper code: %d", errx.Code(errors.New("plain")))
	}
	if errx.IsOfType(errors.New("plain"), errx.NotFound) {
		t.Fatal("did not expect plain error to match type")
	}
	if errx.IsOfType(errx.NotFound.New("missing"), nil) {
		t.Fatal("did not expect nil type matcher to succeed")
	}

	// 测试收尾：没有申请外部资源或额外注册类型。
}

// TestWrapPrivatePreservesCauseWithoutRendering 验证私有原因可判别但不进入公开诊断。
//
// 要做什么：
// - 同时验证原因身份、结构化错误身份与敏感文本隔离。
//
// 具体怎么做：
// - 用含模拟令牌的 URL 错误包装超时原因，再以安全消息创建私有包装。
// - 检查标准错误链、消息、调用栈、字段与再次包装后的显示文本。
//
// 输出什么：
// - 原因链保持原对象，公开诊断不含模拟敏感内容，空原因仍返回 nil。
func TestWrapPrivatePreservesCauseWithoutRendering(t *testing.T) {
	// 测试准备：构造带模拟敏感地址的超时原因。
	cause := &url.Error{Op: "Get", URL: "https://example.test/?token=secret", Err: context.DeadlineExceeded}

	// 测试交互：创建只公开安全消息的包装错误。
	failure := errx.ExternalError.WrapPrivate(cause, "request %d failed", 7)
	var original *url.Error

	// 验证答案：原因身份和错误类型保持稳定，调用栈在创建时已捕获。
	if !errors.Is(failure, context.DeadlineExceeded) || !errors.As(failure, &original) || original != cause || failure.Cause() != cause {
		t.Fatal("private cause identity lost")
	}
	if !errx.IsOfType(failure, errx.ExternalError) || failure.Message() != "request 7 failed" || failure.Stacktrace() == "" {
		t.Fatalf("structured identity lost: %v", failure)
	}
	// 逐一检查所有公开渲染入口，包含再次包装后的显示文本。
	for _, display := range []string{failure.Error(), failure.Message(), fmt.Sprint(failure.Attrs()), fmt.Sprint(errx.Attrs(failure)), errx.WrapError(failure, "outer").Error()} {
		if strings.Contains(display, "secret") || strings.Contains(display, "example.test") {
			t.Fatalf("private cause rendered: %s", display)
		}
	}
	if errx.ExternalError.WrapPrivate(nil, "unused") != nil {
		t.Fatal("nil cause must remain nil")
	}

	// 测试收尾：样本仅使用内存错误，不发起网络请求。
}
