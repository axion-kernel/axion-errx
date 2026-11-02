package errx

import "encoding/json"

// Options 构造具有准确消息、原因链和可选协议元数据的共享错误。
type Options struct {
	// Message 是准确的诊断文本，不自动拼接 Cause 的消息。
	Message string

	// Cause 保留原始错误供 errors.Is/As 检查，不加入公开诊断字段。
	Cause error

	// Metadata 是由调用方解释的 JSON 数据；构造后与调用方缓冲区独立。
	Metadata json.RawMessage

	// Payload 保存进程内原值身份，不自动序列化；其可变数据由调用方负责隔离。
	Payload any
}

// NewWithOptions 创建共享错误，独立捕获准确消息、私有原因、堆栈及元数据。
func (t *Type) NewWithOptions(options Options) *Error {
	return &Error{
		typ:          t,
		message:      options.Message,
		cause:        options.Cause,
		stacktrace:   captureStacktrace(),
		privateCause: true,
		metadata:     append(json.RawMessage(nil), options.Metadata...),
		payload:      options.Payload,
	}
}

// Payload 返回调用方明确绑定的进程内值，不渲染、不复制、不跨协议传输。
func (e *Error) Payload() any {
	if e == nil {
		return nil
	}
	return e.payload
}

// Metadata 返回调用方协议元数据的独立副本；没有元数据时返回 nil。
func (e *Error) Metadata() json.RawMessage {
	if e == nil {
		return nil
	}
	return append(json.RawMessage(nil), e.metadata...)
}
