package errx

// httpStatusProvider 由错误实现声明自身的 HTTP 失败状态。
type httpStatusProvider interface {
	// HttpStatus 返回错误所属的 HTTP 失败状态。
	//
	// 要做什么：
	// - 将状态的定义权保留在错误所属仓。
	//
	// 具体怎么做：
	// - 返回错误类型或当前实例声明的状态。
	//
	// 输出什么：
	// - 返回 400–599 范围内的 HTTP 状态。
	HttpStatus() int
}
