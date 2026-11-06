package errx

import "reflect"

// FindIf 查找标准原因树中满足调用方条件的第一个错误。
//
// 要做什么：
// - 为调用方提供统一的原因查询，避免各业务库复制遍历实现。
//
// 具体怎么做：
// - 按当前节点、单原因或多原因的声明顺序深度优先遍历，不渲染错误文本。
// - 跳过空值与类型化空值，最多访问 4096 个节点，约束循环原因和异常大树。
// - 条件只读检查当前节点；业务码、协议载荷和匹配策略仍由调用方拥有。
//
// 输出什么：
// - 返回首个命中错误；空条件、没有命中或达到访问上限时返回 nil。
func FindIf(source error, match func(error) bool) error {
	if match == nil {
		return nil
	}
	pending := []error{source}

	// 用有界栈按原因声明顺序深度优先访问，空值和类型化空值不交给匹配器。
	for visited := 0; len(pending) > 0 && visited < 4096; visited++ {
		last := len(pending) - 1
		current := pending[last]
		pending = pending[:last]
		if current == nil {
			continue
		}
		value := reflect.ValueOf(current)
		switch value.Kind() {
		case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
			if value.IsNil() {
				continue
			}
		}
		if match(current) {
			return current
		}

		// 多原因按逆序压栈以保留声明顺序，单原因直接压栈等待下一次访问。
		switch cause := current.(type) {
		case interface {
			// Unwrap 提供当前错误的多个直接原因。
			//
			// 要做什么：
			// - 为原因树遍历暴露按声明顺序排列的子错误。
			//
			// 具体怎么做：
			// - 由错误实现返回子错误切片，查询方按剩余访问额度截取并压栈。
			//
			// 输出什么：
			// - 返回直接原因列表，空列表表示没有可继续遍历的原因。
			Unwrap() []error
		}:
			children := cause.Unwrap()
			remaining := 4096 - visited - 1 - len(pending)
			if len(children) > remaining {
				children = children[:remaining]
			}
			for index := len(children) - 1; index >= 0; index-- {
				pending = append(pending, children[index])
			}
		case interface {
			// Unwrap 提供当前错误的单个直接原因。
			//
			// 要做什么：
			// - 让遍历继续进入单原因错误链。
			//
			// 具体怎么做：
			// - 由错误实现返回其包装的原因，查询方将该原因压栈。
			//
			// 输出什么：
			// - 返回直接原因，nil 表示没有下一层原因。
			Unwrap() error
		}:
			pending = append(pending, cause.Unwrap())
		}
	}
	return nil
}
