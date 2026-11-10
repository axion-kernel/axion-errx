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

	// HttpStatus 为当前实例声明 HTTP 失败状态，零值沿用错误类型的状态。
	HttpStatus int
}

// NewWithOptions 创建共享错误，独立捕获准确消息、私有原因、堆栈及元数据。
//
// 要做什么：
// - 使用显式选项构造错误，将公开诊断文本与私有原因链分开保存。
//
// 具体怎么做：
// - 保存当前类型、消息和原始原因，并在构造位置捕获调用栈。
// - 复制元数据字节，保留载荷原值，并标记原因不参与公开诊断渲染。
//
// 输出什么：
// - 返回新错误；其元数据与调用方缓冲区隔离，载荷仍保留原值身份。
func (t *Type) NewWithOptions(options Options) *Error {
	if options.HttpStatus != 0 {
		validateHttpStatus(options.HttpStatus)
	}
	return &Error{
		typ:          t,
		message:      options.Message,
		cause:        options.Cause,
		stacktrace:   captureStacktrace(),
		privateCause: true,
		metadata:     append(json.RawMessage(nil), options.Metadata...),
		payload:      options.Payload,
		httpStatus:   options.HttpStatus,
	}
}

// Payload 返回调用方明确绑定的进程内值，不渲染、不复制、不跨协议传输。
//
// 要做什么：
// - 获取构造错误时附带的进程内载荷。
//
// 具体怎么做：
// - 空接收者直接返回 nil，否则原样返回保存的载荷，不复制其中的可变数据。
//
// 输出什么：
// - 返回载荷原值；空接收者或未设置载荷时返回 nil。
func (e *Error) Payload() any {
	if e == nil {
		return nil
	}
	return e.payload
}

// Metadata 返回调用方协议元数据的独立副本；没有元数据时返回 nil。
//
// 要做什么：
// - 读取错误的协议元数据，避免调用方修改错误内部保存的字节。
//
// 具体怎么做：
// - 空接收者返回 nil，否则将保存的元数据复制到独立字节切片。
//
// 输出什么：
// - 返回可由调用方独立修改的 JSON 字节副本；没有数据时返回 nil。
func (e *Error) Metadata() json.RawMessage {
	if e == nil {
		return nil
	}
	return append(json.RawMessage(nil), e.metadata...)
}
