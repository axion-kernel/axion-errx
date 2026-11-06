package main

import (
	"os"
	"path/filepath"
	"testing"
)

// TestDeclarationLayout 覆盖包级声明的顺序、分组格式和适用边界。
//
// 要做什么：
// - 接受可变长度的多行标题、可省略的单组标题和跨行条目。
// - 拒绝错误顺序、遗漏说明和错误间距。
//
// 具体怎么做：
// - 为每个场景在临时目录写入 Go 文件，执行门禁的 AST 检查。
// - 对比检查结果和预期，由临时目录负责清理。
//
// 输出什么：
// - 检查结果与预期不一致时报告对应场景。
func TestDeclarationLayout(t *testing.T) {
	cases := []struct {
		name string
		body string
		want bool
	}{
		{
			name: "多行标题及不同宽度",
			body: `
const (
	// ---
	// 首组说明。
	// 补充背景。
	// ---

	// first 是第一个枚举值。
	first = iota

	// ----------
	// 第二组说明。
	// ----------

	// second 是后续枚举值。
	second
)

var (
	// values 是用于测试的值。
	// 初始化表达式可以跨行。
	values = []int{
		first,
		second,
	}
)

func use() {
	const local = 1
}
`,
			want: true,
		},
		{name: "允许空文件区段", body: "type Item int\n", want: true},
		{name: "拒绝后置变量", body: "type Item int\nvar (\n// 变量说明。\nitem Item\n)\n"},
		{name: "拒绝先变量后常量", body: "var (\n// 变量说明。\nv = 1\n)\nconst (\n// 常量说明。\nc = 1\n)\n"},
		{name: "拒绝重复声明块", body: "const (\n// 常量说明。\na = 1\n)\nconst (\n// 常量说明。\nb = 2\n)\n"},
		{name: "拒绝单行包级声明", body: "// 常量说明。\nconst a = 1\n"},
		{name: "拒绝缺少条目说明", body: "const (\na = 1\n)\n"},
		{name: "拒绝缺少条目间距", body: "const (\n// 首条说明。\na = 1\n// 次条说明。\nb = 2\n)\n"},
		{name: "拒绝缺少标题间距", body: "const (\n// ---\n// 分组说明。\n// ---\n// 条目说明。\na = 1\n)\n"},
		{name: "拒绝标题横线不等长", body: "const (\n// ---\n// 分组说明。\n// ----\n\n// 条目说明。\na = 1\n)\n"},
		{name: "拒绝多组省略首组标题", body: "const (\n// 首条说明。\na = 1\n\n// ---\n// 分组说明。\n// ---\n\n// 次条说明。\nb = 2\n)\n"},
		{name: "拒绝块尾空行", body: "const (\n// 条目说明。\na = 1\n\n)\n"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "input.go")
			if err := os.WriteFile(path, []byte("package example\n\n"+tc.body), 0600); err != nil {
				t.Fatal(err)
			}
			if got := checkFile(path); got != tc.want {
				t.Fatalf("检查结果为 %t，期望 %t", got, tc.want)
			}
		})
	}
}
