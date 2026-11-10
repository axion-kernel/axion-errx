package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// checker 持有单个函数的错误身份传播和违规诊断。
type checker struct {
	fset       *token.FileSet
	file       string
	errors     map[string]bool
	statuses   map[string]bool
	violations []string
}

// checkRoot 检查仓库内的生产 Go 源码。
//
// 要做什么：
// - 覆盖公开实现和调试宿主，不扫描第三方或生成物。
//
// 具体怎么做：
// - 遍历 pkg、cmd、internal，解析每份非测试 Go 文件。
//
// 输出什么：
// - 返回排序遍历产生的全部诊断，读取失败同样阻断门禁。
func checkRoot(root string) []string {
	var violations []string
	if info, err := os.Stat(root); err != nil || !info.IsDir() {
		return []string{"http-error-contract: 源码根必须是存在的目录"}
	}
	for _, name := range []string{"pkg", "cmd", "internal"} {
		base := filepath.Join(root, name)
		_ = filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if path != base {
					violations = append(violations, err.Error())
				}
				return nil
			}
			if entry.IsDir() {
				if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "node_modules" || entry.Name() == "vendor" || entry.Name() == "thirdparty" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			file, parseErr := parser.ParseFile(fset, path, nil, parser.ParseComments)
			if parseErr != nil {
				violations = append(violations, parseErr.Error())
				return nil
			}
			if ast.IsGenerated(file) {
				return nil
			}
			relative, _ := filepath.Rel(root, path)
			violations = append(violations, checkFile(fset, filepath.ToSlash(relative), file)...)
			return nil
		})
	}
	return violations
}

// checkFile 对每个函数建立局部错误与状态依赖。
//
// 要做什么：
// - 拒绝传输层重新解释错误身份，同时允许协议 owner 声明自身状态。
//
// 具体怎么做：
// - 识别 error 参数、查码表达式及 HTTP 状态出口，再检查控制与赋值。
//
// 输出什么：
// - 返回当前文件中的合同违规。
func checkFile(fset *token.FileSet, path string, file *ast.File) []string {
	var violations []string
	for _, declaration := range file.Decls {
		fn, ok := declaration.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			continue
		}
		if fn.Name.Name == "WriteError" && len(fn.Type.Params.List) == 4 {
			if first, ok := fn.Type.Params.List[0].Type.(*ast.SelectorExpr); ok && first.Sel.Name == "ResponseWriter" {
				violations = append(violations, fmt.Sprintf("%s:%d: HTTP WriteError 不得声明宿主 status 参数", path, fset.Position(fn.Pos()).Line))
			}
		}
		// 状态的真实 owner 是 Errx 查询和各错误本体的 HttpStatus 方法。
		if fn.Name.Name == "HttpStatus" && (fn.Recv != nil || path == "pkg/errx/http_status.go") {
			continue
		}
		c := &checker{fset: fset, file: path, errors: map[string]bool{}, statuses: map[string]bool{}}
		for _, field := range fn.Type.Params.List {
			if id, ok := field.Type.(*ast.Ident); ok && id.Name == "error" {
				for _, name := range field.Names {
					c.errors[name.Name] = true
				}
			}
		}
		ast.Inspect(fn.Body, func(node ast.Node) bool {
			if assignment, ok := node.(*ast.AssignStmt); ok {
				for _, left := range assignment.Lhs {
					if id, ok := left.(*ast.Ident); ok && (id.Name == "err" || strings.HasSuffix(id.Name, "Err") || id.Name == "failure") {
						c.errors[id.Name] = true
					}
				}
			}
			if call, ok := node.(*ast.CallExpr); ok {
				if value := statusArgument(call); value != nil {
					if id, ok := value.(*ast.Ident); ok {
						c.statuses[id.Name] = true
					}
				}
			}
			return true
		})
		// 简单赋值的闭包覆盖先查码再 switch、先取状态再写出的常见写法。
		for pass := 0; pass < 8; pass++ {
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if assignment, ok := node.(*ast.AssignStmt); ok {
					for index, right := range assignment.Rhs {
						if index < len(assignment.Lhs) && c.tainted(right) && !carriedStatus(right) {
							if id, ok := assignment.Lhs[index].(*ast.Ident); ok {
								c.errors[id.Name] = true
							}
						}
					}
				}
				return true
			})
		}
		c.walk(fn.Body, false)
		returnsStatus := false
		if fn.Type.Results != nil && len(fn.Type.Results.List) == 1 {
			if value, ok := fn.Type.Results.List[0].Type.(*ast.Ident); ok {
				returnsStatus = value.Name == "int"
			}
		}
		if returnsStatus && strings.Contains(strings.ToLower(fn.Name.Name), "status") && fn.Recv == nil && len(c.errors) != 0 {
			ast.Inspect(fn.Body, func(node ast.Node) bool {
				if ret, ok := node.(*ast.ReturnStmt); ok {
					for _, value := range ret.Results {
						if !carriedStatus(value) {
							c.report(ret.Pos(), "错误状态辅助函数不得重建映射，请使用 Errx 状态")
						}
					}
				}
				return true
			})
		}
		violations = append(violations, c.violations...)
	}
	return violations
}

// report 添加包含文件和行号的可定位诊断。
//
// 要做什么：
// - 让本地和 CI 能直接定位违规源码。
//
// 具体怎么做：
// - 将 AST 位置转成源码行号并附上合同说明。
//
// 输出什么：
// - 将诊断追加到当前函数结果。
func (c *checker) report(position token.Pos, message string) {
	c.violations = append(c.violations, fmt.Sprintf("%s:%d: %s", c.file, c.fset.Position(position).Line, message))
}

// carriedStatus 判断表达式是否直接读取统一的 Errx 状态。
//
// 要做什么：
// - 将被授权的公共状态读取与本地查表区别开。
//
// 具体怎么做：
// - 仅接受一个参数的 Errx HttpStatus 调用。
//
// 输出什么：
// - 返回表达式是否为公共状态入口。
func carriedStatus(expression ast.Expr) bool {
	call, ok := expression.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return false
	}
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "HttpStatus" {
		return false
	}
	owner, ok := selector.X.(*ast.Ident)
	return ok && owner.Name == "errx"
}

// tainted 判断表达式是否取决于错误身份或查码结果。
//
// 要做什么：
// - 捕获按错误类型、错误码和错误文本选择状态的路径。
//
// 具体怎么做：
// - 遍历标识符和错误查询调用，传播局部赋值依赖。
//
// 输出什么：
// - 返回表达式是否依赖错误。
func (c *checker) tainted(expression ast.Expr) bool {
	if expression == nil || carriedStatus(expression) {
		return false
	}
	found := false
	ast.Inspect(expression, func(node ast.Node) bool {
		if id, ok := node.(*ast.Ident); ok && c.errors[id.Name] {
			found = true
		}
		if call, ok := node.(*ast.CallExpr); ok {
			if selector, ok := call.Fun.(*ast.SelectorExpr); ok && (selector.Sel.Name == "IsOfType" || selector.Sel.Name == "Cast" || selector.Sel.Name == "Code") {
				found = true
			}
		}
		return !found
	})
	return found
}

// statusArgument 识别标准 HTTP 框架和常用响应封装的状态参数。
//
// 要做什么：
// - 保留直接使用框架声明状态的权利，同时定位错误驱动的状态输出。
//
// 具体怎么做：
// - 读取 WriteHeader、http.Error/Redirect 及带状态值的 JSON 辅助入口。
//
// 输出什么：
// - 返回状态表达式；非状态出口返回 nil。
func statusArgument(call *ast.CallExpr) ast.Expr {
	if selector, ok := call.Fun.(*ast.SelectorExpr); ok {
		if selector.Sel.Name == "WriteHeader" && len(call.Args) == 1 {
			return call.Args[0]
		}
		if owner, ok := selector.X.(*ast.Ident); ok && owner.Name == "http" {
			if selector.Sel.Name == "Error" && len(call.Args) == 3 {
				return call.Args[2]
			}
			if selector.Sel.Name == "Redirect" && len(call.Args) == 4 {
				return call.Args[3]
			}
		}
	}
	if name, ok := call.Fun.(*ast.Ident); ok && len(call.Args) >= 3 {
		if name.Name == "writeJSON" || name.Name == "respond" || name.Name == "writeErrorWithCode" || name.Name == "writeFailure" || name.Name == "writeError" {
			value := call.Args[1]
			if looksStatus(value) {
				return value
			}
		}
	}
	return nil
}

// looksStatus 判断表达式是否明确表示 HTTP 状态。
//
// 要做什么：
// - 识别状态变量和标准 HTTP 常量，避免将普通业务字段视为状态。
//
// 具体怎么做：
// - 接受状态命名、HTTP 状态常量及有效范围内的整数字面量。
//
// 输出什么：
// - 返回是否存在明确的状态语义。
func looksStatus(value ast.Expr) bool {
	switch expression := value.(type) {
	case *ast.Ident:
		return strings.Contains(strings.ToLower(expression.Name), "status")
	case *ast.SelectorExpr:
		return strings.HasPrefix(expression.Sel.Name, "Status")
	case *ast.BasicLit:
		number, err := strconv.Atoi(expression.Value)
		return err == nil && number >= 100 && number <= 599
	}
	return false
}

// successfulStatus 判断明确写出的成功或重定向状态。
//
// 要做什么：
// - 区分错误失败映射与领域将预期缺失投影为正常业务值的行为。
//
// 具体怎么做：
// - 识别低于 400 的状态字面量和当前标准库的成功、重定向常量。
//
// 输出什么：
// - 返回是否为明确的非失败状态。
func successfulStatus(value ast.Expr) bool {
	if literal, ok := value.(*ast.BasicLit); ok {
		number, err := strconv.Atoi(literal.Value)
		return err == nil && number >= 100 && number < 400
	}
	if selector, ok := value.(*ast.SelectorExpr); ok {
		name := selector.Sel.Name
		return name == "StatusOK" || name == "StatusAccepted" || name == "StatusCreated" || name == "StatusNoContent" || name == "StatusNotModified" || name == "StatusMovedPermanently" || name == "StatusFound" || name == "StatusSeeOther" || name == "StatusTemporaryRedirect" || name == "StatusPermanentRedirect"
	}
	return false
}

// walk 检查状态选择的控制流与响应字段。
//
// 要做什么：
// - 拒绝错误到状态的重复决策以及状态冒充业务码。
//
// 具体怎么做：
// - 递归检查错误分类控制块，允许直接读取公共状态；普通存在性判断不重新分类。
//
// 输出什么：
// - 将不合规状态赋值、出口和错误正文登记为诊断。
func (c *checker) walk(node ast.Node, classified bool) {
	if node == nil {
		return
	}
	switch current := node.(type) {
	case *ast.IfStmt:
		c.walk(current.Init, classified)
		control := false
		ast.Inspect(current.Cond, func(child ast.Node) bool {
			if binary, ok := child.(*ast.BinaryExpr); ok && (binary.Op == token.EQL || binary.Op == token.NEQ) {
				if id, ok := binary.Y.(*ast.Ident); ok && id.Name == "nil" {
					return false
				}
				if id, ok := binary.X.(*ast.Ident); ok && id.Name == "nil" {
					return false
				}
			}
			if id, ok := child.(*ast.Ident); ok && c.errors[id.Name] {
				control = true
			}
			return true
		})
		c.walk(current.Body, classified || control)
		c.walk(current.Else, classified || control)
		return
	case *ast.SwitchStmt:
		c.walk(current.Init, classified)
		control := c.tainted(current.Tag)
		for _, item := range current.Body.List {
			branch := item.(*ast.CaseClause)
			for _, condition := range branch.List {
				control = control || c.tainted(condition)
			}
		}
		c.walk(current.Body, classified || control)
		return
	case *ast.TypeSwitchStmt:
		c.walk(current.Body, true)
		return
	case *ast.AssignStmt:
		for index, left := range current.Lhs {
			id, ok := left.(*ast.Ident)
			if !ok || index >= len(current.Rhs) {
				continue
			}
			value := current.Rhs[index]
			if c.statuses[id.Name] && (classified || c.tainted(value)) && !carriedStatus(value) {
				c.report(current.Pos(), "禁止根据错误身份选择 HTTP status，请读取错误自带状态")
			}
		}
	case *ast.CallExpr:
		if selector, ok := current.Fun.(*ast.SelectorExpr); ok && selector.Sel.Name == "WriteError" && len(current.Args) == 4 {
			c.report(current.Pos(), "WriteError 不得接受宿主 status，请使用三参数入口")
		}
		if name, ok := current.Fun.(*ast.Ident); ok && len(current.Args) == 4 {
			if name.Name == "WriteError" {
				c.report(current.Pos(), "WriteError 不得接受宿主 status，请使用三参数入口")
			}
			if name.Name == "writeErrorWithCode" {
				status, leftOK := current.Args[1].(*ast.Ident)
				code, rightOK := current.Args[2].(*ast.Ident)
				if leftOK && rightOK && status.Name == code.Name && looksStatus(status) {
					c.report(current.Pos(), "HTTP status 不得同时作为业务 code 传入")
				}
			}
		}
		if value := statusArgument(current); value != nil && !carriedStatus(value) {
			if c.tainted(value) || classified && looksStatus(value) && !successfulStatus(value) {
				c.report(current.Pos(), "禁止根据错误分类驱动 HTTP 状态出口")
			}
		}
	case *ast.KeyValueExpr:
		key := ""
		switch field := current.Key.(type) {
		case *ast.BasicLit:
			key, _ = strconv.Unquote(field.Value)
		case *ast.Ident:
			key = field.Name
		}
		_, literal := current.Value.(*ast.BasicLit)
		if strings.EqualFold(key, "code") && !literal && (looksStatus(current.Value) || carriedStatus(current.Value)) {
			c.report(current.Pos(), "HTTP status 不得冒充业务 code")
		}
	}
	ast.Inspect(node, func(child ast.Node) bool {
		if child == node {
			return true
		}
		if child != nil {
			c.walk(child, classified)
		}
		return false
	})
}
