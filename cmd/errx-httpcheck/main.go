// Command errx-httpcheck 检查错误身份与 HTTP 状态的统一投影合同。
package main

import (
	"flag"
	"fmt"
	"os"
)

// main 执行当前仓库的 HTTP 错误合同检查。
//
// 要做什么：
// - 为各仓门禁提供同一份可发布的检查器。
//
// 具体怎么做：
// - 读取显式源码根，输出带路径和行号的违规。
//
// 输出什么：
// - 合规返回零，有违规或解析失败返回非零退出码。
func main() {
	root := flag.String("root", ".", "待检查的仓库源码根")
	flag.Parse()
	violations := checkRoot(*root)
	for _, violation := range violations {
		fmt.Fprintln(os.Stderr, violation)
	}
	if len(violations) != 0 {
		os.Exit(1)
	}
	fmt.Println("[http-error-contract] passed: carried Errx status; explicit framework status remains allowed")
}
