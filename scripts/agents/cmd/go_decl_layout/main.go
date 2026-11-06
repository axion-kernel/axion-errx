package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// main 从仓库根扫描自有 Go 文件，输出文件、行号和违规原因。
func main() {
	failed := false
	err := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path != "." && excludedDir(path, entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Type().IsRegular() && filepath.Ext(path) == ".go" {
			failed = !checkFile(filepath.ToSlash(path)) || failed
		}
		return nil
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		failed = true
	}
	if failed {
		os.Exit(1)
	}
	fmt.Println("[go-decl-layout-gate] 通过：包级常量与变量布局符合规范")
}

// excludedDir 跳过外部依赖、上游镜像、本地产物和嵌套仓库。
func excludedDir(path, name string) bool {
	switch strings.ToLower(name) {
	case "vendor", "thirdparty", "third_party", "node_modules", "upstream", "dist", "bin", "output", "__pycache__":
		return true
	}
	if strings.HasPrefix(name, ".") {
		return true
	}
	_, err := os.Lstat(filepath.Join(path, ".git"))
	return err == nil
}

// checkFile 用 Go AST 识别包级声明并保留物理行号。
func checkFile(path string) bool {
	src, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, src, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return false
	}

	lines := strings.Split(string(src), "\n")
	line := func(pos token.Pos) int { return fset.PositionFor(pos, false).Line }
	ok := true
	report := func(at int, message string) {
		fmt.Fprintf(os.Stderr, "%s:%d: %s\n", path, at, message)
		ok = false
	}

	// 声明序位只随文件推进。类型和函数共同占据文件末尾的区段。
	stage := 0
	seen := make(map[token.Token]bool)
	for _, decl := range file.Decls {
		group, isGroup := decl.(*ast.GenDecl)
		if !isGroup || group.Tok == token.TYPE {
			stage = 3
			continue
		}
		if group.Tok == token.IMPORT {
			if stage != 0 {
				report(line(group.Pos()), "import 必须位于包级 const、var 和其他声明之前")
			}
			continue
		}

		next := 1
		if group.Tok == token.VAR {
			next = 2
		}
		if stage > next {
			report(line(group.Pos()), "声明顺序应为 import、const、var、其余声明")
		} else {
			stage = next
		}
		if seen[group.Tok] {
			report(line(group.Pos()), "每个文件只能有一个包级 "+group.Tok.String()+" 块")
		}
		seen[group.Tok] = true
		checkBlock(group, lines, line, report)
	}
	return ok
}

// checkBlock 检查条目文档、分组标题和相邻内容之间的物理空行。
//
// 要做什么：
// - 逐条核对括号块中的文档和间距。
//
// 具体怎么做：
// - 以 AST 的条目和文档位置划定间隔，在间隔中识别分组标题。
// - 记录首组是否有标题，并用条目末行核对闭括号的位置。
//
// 输出什么：
// - 将发现的违规交给报告函数，并保留实际源码行号。
func checkBlock(group *ast.GenDecl, lines []string, line func(token.Pos) int, report func(int, string)) {
	start := line(group.Pos())
	if !group.Lparen.IsValid() || len(group.Specs) == 0 {
		report(start, "包级 "+group.Tok.String()+" 必须使用非空括号块")
		return
	}
	if line(group.Lparen) != start || !strings.HasSuffix(strings.TrimSpace(lines[start-1]), "(") {
		report(start, "左括号应与声明关键字同一行")
	}

	previousEnd := line(group.Lparen)
	firstHasHeading := false
	for index, node := range group.Specs {
		spec := node.(*ast.ValueSpec)
		docStart := line(spec.Pos())
		if spec.Doc == nil || !containsChinese(spec.Doc.Text()) {
			report(docStart, "每条包级声明前应紧贴中文说明")
		}
		if spec.Doc != nil {
			docStart = line(spec.Doc.Pos())
			for _, comment := range spec.Doc.List {
				if isHeadingRule(strings.TrimSpace(comment.Text)) {
					report(line(comment.Pos()), "分组标题和条目说明之间必须空一行")
					break
				}
			}
		}

		// 区间只包含前一条结束到当前说明之间的内容，排除多行初始化和注释内部。
		if docStart <= previousEnd {
			report(docStart, "声明及其说明应从独立行开始")
		} else {
			gap := lines[previousEnd : docStart-1]
			if index > 0 {
				if len(gap) == 0 || strings.TrimSpace(gap[0]) != "" {
					report(docStart, "相邻声明或分组之间必须空一行")
				} else {
					gap = gap[1:]
				}
			}
			if len(gap) > 0 {
				if !validHeading(gap) {
					report(docStart, "分组说明须由等长的 // --- 包围，并与条目说明隔一个空行")
				}
				if index == 0 {
					firstHasHeading = true
				} else if !firstHasHeading {
					report(docStart, "块内有多个分组时，第一组也须写分组说明")
				}
			}
		}

		previousEnd = line(spec.End())
		if spec.Comment != nil {
			previousEnd = line(spec.Comment.End())
		}
	}
	if line(group.Rparen) != previousEnd+1 {
		report(line(group.Rparen), "最后一条声明后应直接闭合括号块")
	}
}

// validHeading 识别横线可变长、首尾匹配的多行分组标题。
func validHeading(gap []string) bool {
	if len(gap) < 4 || strings.TrimSpace(gap[len(gap)-1]) != "" {
		return false
	}
	first := strings.TrimSpace(gap[0])
	last := strings.TrimSpace(gap[len(gap)-2])
	if !isHeadingRule(first) || first != last {
		return false
	}
	hasDescription := false
	for _, row := range gap[1 : len(gap)-2] {
		text, isComment := strings.CutPrefix(strings.TrimSpace(row), "//")
		if !isComment || isHeadingRule(strings.TrimSpace(row)) {
			return false
		}
		hasDescription = hasDescription || containsChinese(text)
	}
	return hasDescription
}

// isHeadingRule 判断独立注释行是否为分组横线。
func isHeadingRule(row string) bool {
	dashes, found := strings.CutPrefix(row, "// ")
	return found && len(dashes) >= 3 && strings.Trim(dashes, "-") == ""
}

// containsChinese 判断文档正文中是否包含中文说明。
func containsChinese(text string) bool {
	return strings.ContainsFunc(text, func(r rune) bool { return unicode.Is(unicode.Han, r) })
}
