// doc_comments 提取文档注释及源码范围，供 Python 门禁执行中文与段落格式检查。
package main

import (
	"encoding/json"
	"flag"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// docEntry 保存 Python 校验器需要的声明位置与原始注释。
type docEntry struct {
	// Name 标识声明、包或声明组的名称。
	Name string `json:"name"`

	// Path 是相对扫描根的斜杠分隔路径。
	Path string `json:"path"`

	// Function 表示此条目是否使用函数文档的完整格式。
	Function bool `json:"function"`

	// Start 是文档开头或无文档声明开头的一基行号。
	Start int `json:"start"`

	// End 是声明末尾的一基行号，供增量范围求交。
	End int `json:"end"`

	// Comment 保留各条原始注释文本，包括空行注释。
	Comment []string `json:"comment"`
}

// collectDocs 提取一个 Go 文件内的受检声明文档。
//
// 要做什么：
// - 为后续中文格式校验提供源码范围、符号种类和原始说明。
//
// 具体怎么做：
// - 使用带注释的 Go AST 解析文件，跳过 AST 认定的生成文件。
// - 收集函数、接口方法及选定类型、字段、声明组说明，保留现有筛选条件。
//
// 输出什么：
// - 返回文档条目；解析失败返回错误，生成文件返回空结果。
func collectDocs(path, relative string) ([]docEntry, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	if ast.IsGenerated(file) {
		return nil, nil
	}
	var entries []docEntry

	// 同时记录声明范围与可选的前置说明，缺少说明的受检符号仍保留条目。
	add := func(name string, group *ast.CommentGroup, node ast.Node, function bool) {
		start := node.Pos()
		var comments []string
		if group != nil {
			start = group.Pos()
			for _, comment := range group.List {
				comments = append(comments, comment.Text)
			}
		}
		entries = append(entries, docEntry{name, relative, function, fset.Position(start).Line, fset.Position(node.End()).Line, comments})
	}
	if file.Doc != nil {
		add(file.Name.Name, file.Doc, file.Doc, false)
	}

	// 按声明种类收集文档，不进行类型检查或包编译。
	for _, declaration := range file.Decls {
		switch decl := declaration.(type) {
		case *ast.FuncDecl:
			add(decl.Name.Name, decl.Doc, decl, true)
		case *ast.GenDecl:
			if decl.Tok != token.TYPE && decl.Doc != nil {
				add(decl.Tok.String(), decl.Doc, decl, false)
			}
			for _, spec := range decl.Specs {
				if value, ok := spec.(*ast.ValueSpec); ok && value.Doc != nil {
					add(value.Names[0].Name, value.Doc, value, false)
				}
				typed, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				group := typed.Doc
				if group == nil && len(decl.Specs) == 1 {
					group = decl.Doc
				}
				if typed.Name.IsExported() || group != nil {
					add(typed.Name.Name, group, typed, false)
				}

				// 接口的具名方法全部受检，结构体字段仅收集导出项或已有前置说明的项。
				switch value := typed.Type.(type) {
				case *ast.InterfaceType:
					for _, method := range value.Methods.List {
						for _, name := range method.Names {
							add(name.Name, method.Doc, method, true)
						}
					}
				case *ast.StructType:
					for _, field := range value.Fields.List {
						for _, name := range field.Names {
							if name.IsExported() || field.Doc != nil {
								add(name.Name, field.Doc, field, false)
							}
						}
					}
				}
			}
		}
	}
	return entries, nil
}

// main 遍历既有源码根并输出文档条目 JSON。
//
// 要做什么：
// - 为 Python 包装器提供稳定的 AST 提取命令入口。
//
// 具体怎么做：
// - 解析扫描根，遍历已存在的 pkg 与 internal 目录内所有 Go 文件。
// - 将各文件条目合并后编码到标准输出，遍历或编码失败立即报告。
//
// 输出什么：
// - 成功输出 JSON 数组；失败由标准日志报告并以非零状态退出。
func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	entries := []docEntry{}

	// 源码根与遍历策略保持原状，不扩展到工程工具目录。
	for _, name := range []string{"pkg", "internal"} {
		directory := filepath.Join(*root, name)
		if _, err := os.Stat(directory); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(directory, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			relative, err := filepath.Rel(*root, path)
			if err != nil {
				return err
			}
			docs, err := collectDocs(path, filepath.ToSlash(relative))
			entries = append(entries, docs...)
			return err
		})
		if err != nil {
			log.Fatal(err)
		}
	}
	if err := json.NewEncoder(os.Stdout).Encode(entries); err != nil {
		log.Fatal(err)
	}
}
