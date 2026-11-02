package errx

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

// TestOptionsPreserveExactDiagnosticAndCause 验证共享错误同时保存准确诊断和真实原因。
func TestOptionsPreserveExactDiagnosticAndCause(t *testing.T) {
	metadata := json.RawMessage(`{"name":"UserQuestionError","code":"ASK_ABORTED"}`)
	payload := &struct{ Value string }{Value: "original"}
	err := RejectedOperation.NewWithOptions(Options{Message: "question aborted", Cause: context.Canceled, Metadata: metadata, Payload: payload})
	metadata[2] = 'X'
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
}

// TestOptionsEmptyCauseAndMetadata 验证无原因、空消息和空元数据均为明确值。
func TestOptionsEmptyCauseAndMetadata(t *testing.T) {
	err := IllegalArgument.NewWithOptions(Options{})
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
}
