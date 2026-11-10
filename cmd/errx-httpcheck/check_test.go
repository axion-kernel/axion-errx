package main

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
)

// TestHTTPErrorContractRejectsMappingsAndAllowsFrameworkStatus 验证错误映射门禁的拒绝与许可边界。
//
// 要做什么：
// - 防止 switch、if、查表、旧签名及状态冒充业务码回流，同时保留标准框架直接写状态。
//
// 具体怎么做：
// - 解析真实 Go 语法夹具，检查控制流、赋值别名、公开统一出口和业务安全文案。
//
// 输出什么：
// - 每种违规至少产生一条诊断，明确许可的框架写法不产生诊断。
func TestHTTPErrorContractRejectsMappingsAndAllowsFrameworkStatus(t *testing.T) {
	// 测试准备：夹具无需解析依赖，只检查当前仓库的源码合同。
	cases := []struct {
		name   string
		source string
		reject bool
	}{
		{"code switch", `func serve(w Writer, err error) { status := 500; switch errx.Code(err) { case 1001: status = 400 }; w.WriteHeader(status) }`, true},
		{"classification switch", `func serve(w Writer, err error) { status := 500; switch { case errx.IsOfType(err, errx.NotFound): status = 404 }; w.WriteHeader(status) }`, true},
		{"if mapping", `func serve(w Writer, problem error) { if errx.IsOfType(problem, errx.Forbidden) { w.WriteHeader(403) } }`, true},
		{"nil error fixed status", `func serve(w Writer, err error) { if err != nil { http.Error(w,"failed",400) } }`, true},
		{"nil error carried status", `func serve(w Writer, err error) { if err != nil { status := errx.HttpStatus(err); w.WriteHeader(status) } }`, false},
		{"method fixed status", `func serve(w Writer, err error) { if err != nil { app.writeError(w,400,err) } }`, true},
		{"method carried status", `func serve(w Writer, err error) { if err != nil { app.writeJSON(w,errx.HttpStatus(err),value) } }`, false},
		{"error wrapper signature", `func writeHTTPFailure(w http.ResponseWriter, status int, err error) { w.WriteHeader(status) }`, true},
		{"reversed error wrapper signature", `func writeError(w http.ResponseWriter, err error, status int) { w.WriteHeader(status) }`, true},
		{"declared alias mapping", `func serve(w Writer, err error) { var code = errx.Code(err); var status = table[code]; w.WriteHeader(status) }`, true},
		{"declared carried status", `func serve(w Writer, err error) { if err != nil { var status = errx.HttpStatus(err); w.WriteHeader(status) } }`, false},
		{"lookup table", `func serve(w Writer, err error) { status := table[errx.Code(err)]; w.WriteHeader(status) }`, true},
		{"lookup alias", `func serve(w Writer, err error) { code := errx.Code(err); next := code; status := table[next]; w.WriteHeader(status) }`, true},
		{"raw code status", `func serve(w Writer, err error) { w.WriteHeader(errx.Code(err)) }`, true},
		{"old writer", `func serve(w Writer, err error) { httpx.WriteError(w, 400, err, "safe") }`, true},
		{"old base signature", `func WriteError(w http.ResponseWriter, status int, err error, message string) { encode(message) }`, true},
		{"old local writer", `func serve(w Writer, err error) { WriteError(w, 400, err, "safe") }`, true},
		{"marketdata status as code", `func serve(w Writer) { status := 400; writeErrorWithCode(w, status, status, "safe") }`, true},
		{"carried status as code", `func serve(w Writer, err error) { encode(map[string]any{"code":errx.HttpStatus(err)}) }`, true},
		{"status helper", `func errorStatus(err error) int { switch errx.Code(err) { case 1001: return 400 }; return 500 }`, true},
		{"status as business code", `func serve(w Writer) { status := 400; encode(map[string]any{"code":status}) }`, true},
		{"constant as business code", `func serve(w Writer) { encode(ErrorBody{Code:http.StatusBadRequest}) }`, true},
		{"framework constant", `func serve(w Writer) { w.WriteHeader(http.StatusNoContent); http.Error(w,"method",405); http.Redirect(w,r,"/next",303) }`, false},
		{"route status", `func serve(w Writer) { if r.Method != "POST" { w.WriteHeader(405) }; status := 202; writeJSON(w,status,value) }`, false},
		{"unified writer", `func serve(w Writer, err error) { httpx.WriteError(w,err,"safe") }`, false},
		{"carried status", `func serve(w Writer, err error) { status := errx.HttpStatus(err); w.WriteHeader(status) }`, false},
		{"message classification", `func serve(w Writer, err error) { message := "failed"; switch { case errx.IsOfType(err,errx.NotFound): message = "missing" }; httpx.WriteError(w,err,message) }`, false},
		{"owner method", `func (e *Error) HttpStatus() int { if e.code == 1 { return 400 }; return 500 }`, false},
		{"business code", `func serve(w Writer, err error) { encode(ErrorBody{Code:errx.Code(err)}) }`, false},
	}
	// 测试交互与验证答案：逐条夹具确认门禁只拦截错误驱动的重复映射。
	for _, test := range cases {
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "fixture.go", "package fixture\n"+test.source, 0)
		if err != nil {
			t.Fatal(err)
		}
		violations := checkFile(fset, "pkg/fixture/serve.go", file)
		if (len(violations) != 0) != test.reject {
			t.Errorf("%s: reject=%v diagnostics=%v", test.name, test.reject, violations)
		}
	}
	// 测试收尾：所有夹具均在内存解析，不修改用户源码。
}

// TestHTTPErrorContractCoversProductionRoots 验证门禁扫描范围和不存在根目录的失败。
//
// 要做什么：
// - 确保正式实现和调试宿主都受检，测试与生成物不成为误报。
//
// 具体怎么做：
// - 在临时仓库放置违规生产文件及同名测试、生成文件后运行完整检查。
//
// 输出什么：
// - 三个生产根都报告违规，排除文件不产生诊断，非法根不被视为通过。
func TestHTTPErrorContractCoversProductionRoots(t *testing.T) {
	// 测试准备：每种目录包含相同的违规状态查码。
	root := t.TempDir()
	bad := "package fixture\nfunc serve(w Writer, err error) { w.WriteHeader(errx.Code(err)) }\n"
	for _, name := range []string{"pkg", "cmd", "internal"} {
		dir := filepath.Join(root, name, "fixture")
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		for path, source := range map[string]string{"serve.go": bad, "serve_test.go": bad, "generated.go": "// Code generated by fixture. DO NOT EDIT.\n" + bad} {
			if err := os.WriteFile(filepath.Join(dir, path), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
		}
	}
	// 测试交互与验证答案：检查目录递归及错误根目录。
	if violations := checkRoot(root); len(violations) != 3 {
		t.Fatalf("diagnostics=%v", violations)
	}
	if len(checkRoot(filepath.Join(root, "missing"))) == 0 {
		t.Fatal("missing root accepted")
	}
	// 测试收尾：临时仓库由测试框架自动删除。
}
