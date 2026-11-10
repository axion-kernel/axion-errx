package errx

import "context"

// HttpStatus 返回错误类型声明的 HTTP 失败状态。
//
// 要做什么：
// - 让类型和创建的错误共用所属仓声明的默认状态。
//
// 具体怎么做：
// - 读取注册值，空类型和零值类型使用内部错误状态。
//
// 输出什么：
// - 返回有效状态，未声明时返回 500。
func (t *Type) HttpStatus() int {
	if t == nil || t.httpStatus == 0 {
		return 500
	}
	return t.httpStatus
}

// HttpStatus 返回当前错误实例的 HTTP 失败状态。
//
// 要做什么：
// - 让错误携带自己的传输状态，不依赖宿主查码。
//
// 具体怎么做：
// - 优先读取实例状态，否则读取所属类型；不从私有原因推翻外层分类。
//
// 输出什么：
// - 返回有效状态，空接收者返回 500。
func (e *Error) HttpStatus() int {
	if e == nil {
		return 500
	}
	if e.httpStatus != 0 {
		return e.httpStatus
	}
	return e.typ.HttpStatus()
}

// HttpStatus 读取任意错误自身携带的 HTTP 失败状态。
//
// 要做什么：
// - 为传输边界统一读取错误状态，支持标准包装与多原因错误。
//
// 具体怎么做：
// - 复用有界原因树查询，采用第一个有效的状态声明；显式外层声明优先。
// - 将标准取消与超时视为 408；空值、普通错误和无效状态回退 500。
//
// 输出什么：
// - 返回 400–599 范围内的状态，不读取消息、错误码或诊断载荷。
func HttpStatus(err error) int {
	status := 500
	FindIf(err, func(current error) bool {
		if provider, ok := current.(httpStatusProvider); ok {
			if value := provider.HttpStatus(); value >= 400 && value <= 599 {
				status = value
				return true
			}
		}
		if current == context.Canceled || current == context.DeadlineExceeded {
			status = 408
			return true
		}
		return false
	})
	return status
}

// validateHttpStatus 拒绝非法的 HTTP 失败状态。
//
// 要做什么：
// - 防止错误定义意外产生成功响应或无效响应。
//
// 具体怎么做：
// - 检查状态是否处于 400–599 闭区间。
//
// 输出什么：
// - 合法状态正常返回，非法状态触发 panic。
func validateHttpStatus(status int) {
	if status < 400 || status > 599 {
		panic("errx: HTTP status must be between 400 and 599")
	}
}
