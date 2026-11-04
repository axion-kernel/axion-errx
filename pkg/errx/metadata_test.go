package errx

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// TestOptionsPreserveExactDiagnosticAndCause 验证共享错误同时保存准确诊断和真实原因。
//
// 要做什么：
// - 验证公开诊断与私有原因分离，元数据按值隔离而载荷保留身份。
//
// 具体怎么做：
// - 使用取消原因、JSON 元数据与指针载荷构造错误，再修改输入和读取到的元数据。
// - 检查消息、类型、标准错误链、调用栈、字段与载荷。
//
// 输出什么：
// - 消息保持准确，原因可判别但不进入字段，所有元数据副本互不影响，载荷指针不变。
func TestOptionsPreserveExactDiagnosticAndCause(t *testing.T) {
	// 测试准备：创建可修改的元数据与可判别身份的指针载荷。
	metadata := json.RawMessage(`{"name":"UserQuestionError","code":"ASK_ABORTED"}`)
	payload := &struct{ Value string }{Value: "original"}

	// 测试交互：构造带私有原因的错误，再修改调用方原始元数据。
	err := RejectedOperation.NewWithOptions(Options{Message: "question aborted", Cause: context.Canceled, Metadata: metadata, Payload: payload})
	metadata[2] = 'X'

	// 验证答案：类型、消息、原因身份与调用栈保持构造时的语义。
	if err.Type() != RejectedOperation || err.Code() != RejectedOperation.Code() || err.Message() != "question aborted" || err.Error() != "rejected_operation: question aborted" {
		t.Fatal(err)
	}
	var shared *Error
	if !errors.Is(err, context.Canceled) || !errors.As(err, &shared) || shared != err || err.Stacktrace() == "" {
		t.Fatal("shared cause or stack lost", err)
	}
	if string(err.Metadata()) != `{"name":"UserQuestionError","code":"ASK_ABORTED"}` {
		t.Fatal("constructor retained mutable caller bytes")
	}

	// 修改访问方法和结构化字段返回的副本，验证两者都不暴露内部缓冲区。
	copy := err.Metadata()
	copy[2] = 'Y'
	attrs := err.Attrs()
	attrs[len(attrs)-1].Value.(json.RawMessage)[2] = 'Z'
	if string(err.Metadata()) != `{"name":"UserQuestionError","code":"ASK_ABORTED"}` {
		t.Fatal("metadata read exposed shared bytes")
	}
	if attrs[3].Value != "" {
		t.Fatal("private cause leaked into attributes", attrs[3])
	}
	if err.Payload() != payload {
		t.Fatal("in-process value identity lost")
	}

	// 测试收尾：所有数据均在内存中，无需释放外部资源。
}

// TestOptionsEmptyCauseAndMetadata 验证无原因、空消息和空元数据均为明确值。
//
// 要做什么：
// - 确认空选项和空接收者可用，并保留传统包装方法的消息组合行为。
//
// 具体怎么做：
// - 用零值选项创建错误，查询空指针访问方法，再用传统 Wrap 包装超时原因。
//
// 输出什么：
// - 零值选项不补写消息或元数据；传统包装仍连接新消息与原因消息。
func TestOptionsEmptyCauseAndMetadata(t *testing.T) {
	// 测试准备：使用空选项，不提供原因、元数据或载荷。
	//
	// 测试交互：创建错误并读取空值对应的公开属性。
	err := IllegalArgument.NewWithOptions(Options{})

	// 验证答案：基础字段仍为五项，没有附加元数据。
	if err == nil || err.Message() != "" || err.Cause() != nil || err.Metadata() != nil || len(err.Attrs()) != 5 {
		t.Fatal(err)
	}
	var empty *Error
	if empty.Metadata() != nil || empty.Payload() != nil {
		t.Fatal("nil metadata must be absent")
	}
	legacy := IllegalArgument.Wrap(context.DeadlineExceeded, "legacy")
	if legacy.Message() != "legacy: context deadline exceeded" || legacy.Metadata() != nil {
		t.Fatal("legacy wrapping changed", legacy)
	}

	// 测试收尾：本用例仅创建内存错误对象。
}
