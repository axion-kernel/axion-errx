package errx

import (
	"errors"
	"fmt"
	"testing"
)

// TestFindIfRetainsOrder 验证共享查询能够越过包装并保持多原因顺序。
//
// 要做什么：
// - 查询深层载荷时保留先遇到的分支，不被外层错误类型遮挡。
//
// 具体怎么做：
// - 构造标准包装、私有原因与多原因组合，按载荷选择目标节点。
//
// 输出什么：
// - 返回原节点身份，空输入和空条件均不命中。
func TestFindIfRetainsOrder(t *testing.T) {
	// 测试准备：两个有相同载荷的节点分别处于左右分支。
	first := ExternalError.NewWithOptions(Options{Message: "first", Payload: 42})
	second := ExternalError.NewWithOptions(Options{Message: "second", Payload: 42})
	source := IllegalState.WrapPrivate(errors.Join(fmt.Errorf("context: %w", first), second), "safe")
	// 测试交互：只匹配共享错误中指定载荷，不解释消息文本。
	match := func(err error) bool { node, ok := err.(*Error); return ok && node.Payload() == 42 }
	got := FindIf(source, match)
	// 验证答案：遍历保留分支次序和原始对象身份。
	if got != first || FindIf(nil, match) != nil || FindIf(source, nil) != nil || FindIf(second, func(error) bool { return false }) != nil {
		t.Fatalf("unexpected query result: %v", got)
	}
	// 测试收尾：仅使用内存对象，无外部资源。
}

// TestFindIfBoundsCycles 验证循环原因和类型化空值不会使共享查询失控。
//
// 要做什么：
// - 为异常原因图提供有限查询，并避免调用空接收者方法。
//
// 具体怎么做：
// - 用包内错误自引用构成循环，再混入类型化空值执行查询。
//
// 输出什么：
// - 循环在访问上限内结束，空值分支不进入匹配条件。
func TestFindIfBoundsCycles(t *testing.T) {
	// 测试准备：私有原因指向自身，并准备类型化空指针。
	loop := ExternalError.New("loop")
	loop.cause = loop
	var empty *Error
	count := 0
	// 测试交互：拒绝所有节点以验证终止边界。
	if FindIf(errors.Join(empty, loop), func(err error) bool { count++; return false }) != nil {
		t.Fatal("unexpected match")
	}
	// 验证答案：最多检查有限节点，类型化空值不传给调用方。
	if count < 1 || count > 4096 {
		t.Fatalf("unbounded traversal: %d", count)
	}
	// 测试收尾：解除循环引用便于局部状态检查。
	loop.cause = nil
}
