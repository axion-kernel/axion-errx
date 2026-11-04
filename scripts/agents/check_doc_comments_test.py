"""使用真实 Go AST 与 Git 基线验证错误库注释门禁。"""

from pathlib import Path
import subprocess
import tempfile
import unittest

from check_doc_comments import find_violations, validate_short_comment


DOC = """// Read 读取指定数据。
//
// 要做什么：
// - 读取指定来源。
//
// 具体怎么做：
// - 调用来源接口。
//
// 输出什么：
// - 返回读取结果。
"""


class DocCommentsTest(unittest.TestCase):
    def test_ast_checks_functions_methods_tests_and_comment_only_changes(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            package = root / "pkg/sample"
            package.mkdir(parents=True)
            path = package / "sample.go"
            path.write_text("package sample\n\n" + DOC + "func Read() {}\n", encoding="utf-8")
            self.assertEqual(find_violations(root), [])
            for args in (("init",), ("config", "user.email", "codex@example.com"), ("config", "user.name", "Codex"), ("add", "."), ("commit", "-m", "baseline")):
                subprocess.run(["git", *args], cwd=root, check=True, capture_output=True)
            self.assertEqual(find_violations(root), [])
            path.write_text(path.read_text(encoding="utf-8").replace("//\n// 具体怎么做：", "// 具体怎么做："), encoding="utf-8")
            self.assertTrue(any("空一行" in item for item in find_violations(root)))
            path.write_text("package sample\n\n// Reader 保存读取协议。\ntype Reader interface {\nRead()\n}\n\n// Source 保存来源。\ntype Source struct {}\nfunc (s Source) Read() {}\n", encoding="utf-8")
            (package / "sample_test.go").write_text("package sample\nfunc TestRead(t *testing.T) {}\n", encoding="utf-8")
            violations = find_violations(root)
            self.assertTrue(any("TestRead" in item for item in violations), violations)
            self.assertGreaterEqual(sum("Read: 缺少" in item for item in violations), 2, violations)

    def test_short_type_docs_require_chinese_and_separated_paragraphs(self):
        self.assertEqual(validate_short_comment(["// Source 保存来源。"]), [])
        self.assertTrue(validate_short_comment(["// Source stores data."]))
        self.assertTrue(validate_short_comment(["// Source 保存来源。", "// 约束：", "// - 来源必须存在。 "]))
        self.assertEqual(validate_short_comment(["// Source 保存来源。", "//", "// 约束：", "// - 来源必须存在。"]), [])


if __name__ == "__main__":
    unittest.main()
