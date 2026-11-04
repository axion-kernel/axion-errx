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

type docEntry struct {
	Name     string   `json:"name"`
	Path     string   `json:"path"`
	Function bool     `json:"function"`
	Start    int      `json:"start"`
	End      int      `json:"end"`
	Comment  []string `json:"comment"`
}

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

func main() {
	root := flag.String("root", ".", "repository root")
	flag.Parse()
	entries := []docEntry{}
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
