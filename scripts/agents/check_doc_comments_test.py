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
    """DocCommentsTest 核对 AST 文档提取、Git 增量范围与中文格式。"""

    def test_ast_checks_functions_methods_tests_and_comment_only_changes(self):
        """test_ast_checks_functions_methods_tests_and_comment_only_changes 核对真实工具调用与差异。

        要做什么：
        - 检查合法文档、仅删注释空行及方法和测试缺少文档的样本。

        具体怎么做：
        - 在临时目录准备源码，分别在无基线及真实 Git 提交后调用提取检查。
        - 删除章节前空行检查失败，再写入缺少方法和测试说明的源码检查诊断。

        输出什么：
        - 产生合法输入、空行和缺少说明断言；上下文退出后清理临时仓库。
        """
        # 测试准备：创建只有合法函数文档的隔离 Go 样本。
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            package = root / "pkg/sample"
            package.mkdir(parents=True)
            path = package / "sample.go"
            path.write_text("package sample\n\n" + DOC + "func Read() {}\n", encoding="utf-8")
            # 测试交互与验证：先全量检查，再建立真实 Git 基线确认无变更结果。
            self.assertEqual(find_violations(root), [])
            for args in (("init",), ("config", "user.email", "codex@example.com"), ("config", "user.name", "Codex"), ("add", "."), ("commit", "-m", "baseline")):
                subprocess.run(["git", *args], cwd=root, check=True, capture_output=True)
            self.assertEqual(find_violations(root), [])
            # 测试交互与验证：只删除空注释行，仍须检测到段落格式问题。
            path.write_text(path.read_text(encoding="utf-8").replace("//\n// 具体怎么做：", "// 具体怎么做："), encoding="utf-8")
            self.assertTrue(any("空一行" in item for item in find_violations(root)))
            # 测试交互：替换为缺少接口方法、接收者方法和测试说明的源码文本。
            path.write_text("package sample\n\n// Reader 保存读取协议。\ntype Reader interface {\nRead()\n}\n\n// Source 保存来源。\ntype Source struct {}\nfunc (s Source) Read() {}\n", encoding="utf-8")
            (package / "sample_test.go").write_text("package sample\nfunc TestRead(t *testing.T) {}\n", encoding="utf-8")
            violations = find_violations(root)
            # 验证答案：诊断包括测试和至少两个 Read 声明，不要求样本通过 Go 编译。
            self.assertTrue(any("TestRead" in item for item in violations), violations)
            self.assertGreaterEqual(sum("Read: 缺少" in item for item in violations), 2, violations)
        # 测试收尾：目录上下文已移除样本及 Git 元数据。

    def test_short_type_docs_require_chinese_and_separated_paragraphs(self):
        """test_short_type_docs_require_chinese_and_separated_paragraphs 核对简短类型文档。

        要做什么：
        - 检查中文说明与标题段落空行约束。

        具体怎么做：
        - 分别提交中文单行、英文单行、缺少空行和含空行的标题样本。
        - 比较返回问题列表是否为空。

        输出什么：
        - 产生四种格式断言；样本均在内存中，不需资源清理。
        """
        # 测试准备、交互与验证：将固定正反例直接传给简短注释校验器。
        self.assertEqual(validate_short_comment(["// Source 保存来源。"]), [])
        self.assertTrue(validate_short_comment(["// Source stores data."]))
        self.assertTrue(validate_short_comment(["// Source 保存来源。", "// 约束：", "// - 来源必须存在。 "]))
        self.assertEqual(validate_short_comment(["// Source 保存来源。", "//", "// 约束：", "// - 来源必须存在。"]), [])
        # 测试收尾：没有创建文件或子进程。


if __name__ == "__main__":
    unittest.main()
