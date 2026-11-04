package errx

var (
	// InternalError 表示未命中更具体分类的内部错误。
	InternalError = NewType("internal_error", 1000)

	// IllegalArgument 表示调用方传入的参数值、类型或组合不满足要求。
	IllegalArgument = NewType("illegal_argument", 1001)

	// Unauthorized 表示请求缺少合法身份，或鉴权校验未通过。
	Unauthorized = NewType("unauthorized", 1002)

	// Forbidden 表示身份已确认，但当前主体没有执行目标操作的权限。
	Forbidden = NewType("forbidden", 1003)

	// NotFound 表示请求引用的资源、对象或记录不存在。
	NotFound = NewType("not_found", 1004)

	// Conflict 表示当前操作与现有状态冲突，例如重复创建或版本冲突。
	Conflict = NewType("conflict", 1005)

	// DataUnavailable 表示依赖的数据当前不可用、尚未准备好或无法读取。
	DataUnavailable = NewType("data_unavailable", 1006)

	// RejectedOperation 表示请求语义本身合法，但被业务规则或前置条件拒绝执行。
	RejectedOperation = NewType("rejected_operation", 1007)

	// UnsupportedOperation 表示当前实现、运行模式或能力集合不支持该操作。
	UnsupportedOperation = NewType("unsupported_operation", 1008)

	// NotImplemented 是 UnsupportedOperation 的别名，用于表达功能尚未实现。
	NotImplemented = UnsupportedOperation

	// IllegalState 表示对象或系统当前状态不允许执行该操作。
	IllegalState = NewType("illegal_state", 1009)

	// IllegalFormat 表示输入内容的格式、编码或序列化结构不合法。
	IllegalFormat = NewType("illegal_format", 1010)

	// ExternalError 表示调用外部系统、依赖服务或第三方组件时发生错误。
	ExternalError = NewType("external_error", 1011)

	// InitializationFailed 表示组件、服务或运行环境在初始化阶段失败。
	InitializationFailed = NewType("initialization_failed", 1012)

	// MethodNotAllowed 表示入口层收到不支持的请求方法。
	MethodNotAllowed = NewType("method_not_allowed", 1013)

	// BadRequestBody 表示请求体解析失败，或请求体结构不符合预期。
	BadRequestBody = NewType("bad_request_body", 1014)

	// UpstreamDisconnected 表示外部系统没有发送可解释的错误终态便断开连接。
	UpstreamDisconnected = NewType("upstream_disconnected", 1015)

	// UpstreamQuotaExhausted 表示可用上游账号池配额已经耗尽。
	UpstreamQuotaExhausted = NewType("upstream_quota_exhausted", 1016)

	// ExecutionSkipped 表示上游明确要求跳过本次执行，不应作为暂时故障重试。
	ExecutionSkipped = NewType("execution_skipped", 1017)
)
