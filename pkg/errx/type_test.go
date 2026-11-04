package errx_test

import (
	"testing"

	"github.com/axion-kernel/axion-errx/pkg/errx"
)

// TestNewTypeCarriesNameAndCode 验证注册后的类型保留名称与编码。
//
// 要做什么：
// - 确认类型访问方法返回注册时给定的稳定身份。
//
// 具体怎么做：
// - 使用专属测试名称和编码注册类型，再分别读取名称和编码。
//
// 输出什么：
// - 两个访问结果与注册参数完全相同。
func TestNewTypeCarriesNameAndCode(t *testing.T) {
	t.Parallel()

	// 测试准备：使用当前用例专属的名称和编码，避免与其他注册项冲突。
	//
	// 测试交互：注册类型，再通过公开方法读取身份。
	typ := errx.NewType("type_test_new", 990101)

	// 验证答案：名称和编码均保持原值。
	if typ.Name() != "type_test_new" {
		t.Fatalf("unexpected type name: got %q", typ.Name())
	}
	if typ.Code() != 990101 {
		t.Fatalf("unexpected type code: got %d", typ.Code())
	}

	// 测试收尾：注册表具有进程级生命周期，无需单独释放类型。
}

// TestTypeAccessorsHandleNilReceiver 验证空类型指针的读取结果。
//
// 要做什么：
// - 确认调用方可安全查询空类型指针。
//
// 具体怎么做：
// - 声明空类型指针，分别调用 Name 和 Code。
//
// 输出什么：
// - 名称为空字符串，编码为零，访问过程不发生 panic。
func TestTypeAccessorsHandleNilReceiver(t *testing.T) {
	t.Parallel()

	// 测试准备：保留未初始化的类型指针。
	var typ *errx.Type

	// 测试交互：读取空接收者的名称与编码。
	//
	// 验证答案：两个访问方法均返回各自的零值。
	if typ.Name() != "" {
		t.Fatalf("unexpected nil type name: %q", typ.Name())
	}
	if typ.Code() != 0 {
		t.Fatalf("unexpected nil type code: %d", typ.Code())
	}

	// 测试收尾：本用例没有创建注册项或外部资源。
}
