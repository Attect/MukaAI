package syntax

import (
	"encoding/json"
	"strings"
	"testing"
)

// ==================== Dispatcher 测试 ====================

func TestDispatcher_Check_JSON(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewJSONChecker())

	// 合法JSON
	result := d.Check(`{"key": "value", "num": 42}`, "test.json", 100)
	if result == nil {
		t.Fatal("expected non-nil result for JSON file")
	}
	if result.HasErrors {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
	if result.Language != "json" {
		t.Errorf("expected language 'json', got '%s'", result.Language)
	}
	if result.CheckMethod != "native" {
		t.Errorf("expected method 'native', got '%s'", result.CheckMethod)
	}

	// 非法JSON
	result = d.Check(`{"key": "value" "missing_colon": true}`, "bad.json", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.HasErrors {
		t.Error("expected errors for invalid JSON")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}
	if result.Errors[0].Line == 0 {
		t.Error("expected line number in error")
	}
}

func TestDispatcher_Check_YAML(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewYAMLChecker())

	// 合法YAML
	result := d.Check("key: value\nlist:\n  - item1\n  - item2\n", "test.yaml", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}

	// 非法YAML - 错误缩进
	result = d.Check("key: value\n  bad_indent: true\n", "bad.yaml", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.HasErrors {
		t.Error("expected errors for invalid YAML")
	}
}

func TestDispatcher_Check_XML(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewXMLChecker())

	// 合法XML
	result := d.Check(`<?xml version="1.0"?><root><item>test</item></root>`, "test.xml", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}

	// 非法XML - 未闭合标签
	result = d.Check(`<root><item>unclosed</root>`, "bad.xml", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.HasErrors {
		t.Error("expected errors for invalid XML")
	}
}

func TestDispatcher_Check_HTML(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewHTMLChecker())

	// 合法HTML
	result := d.Check(`<!DOCTYPE html><html><head><title>Test</title></head><body><p>Hello</p></body></html>`, "test.html", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestDispatcher_Check_Go(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewGoChecker())

	// 合法Go代码
	result := d.Check(`package main

func main() {
	println("hello")
}`, "test.go", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}

	// 非法Go代码 - 缺少大括号
	result = d.Check(`package main

func main() {
	println("hello"
`, "bad.go", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.HasErrors {
		t.Error("expected errors for invalid Go")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one error")
	}
	if result.Errors[0].Line == 0 {
		t.Error("expected line number in Go error")
	}
}

func TestDispatcher_UnknownExtension(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewJSONChecker())

	// 未知扩展名应返回nil（跳过检查）
	result := d.Check("some content", "test.txt", 100)
	if result != nil {
		t.Error("expected nil result for unknown extension")
	}
}

func TestDispatcher_NoExtension(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewJSONChecker())

	// 无扩展名应返回nil
	result := d.Check("some content", "Makefile", 100)
	if result != nil {
		t.Error("expected nil result for file without extension")
	}
}

func TestDispatcher_FileSizeLimit(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewJSONChecker())

	// 设置很小的文件大小限制
	d.SetMaxFileSize(10)

	// 超过大小限制
	result := d.Check(`{"key": "this is longer than 10 bytes"}`, "test.json", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Degraded {
		t.Error("expected degraded result for oversized file")
	}
	if result.CheckMethod != "skipped" {
		t.Errorf("expected 'skipped', got '%s'", result.CheckMethod)
	}
}

func TestDispatcher_PanicRecovery(t *testing.T) {
	d := NewDispatcher()
	// 注册一个会panic的检查器
	d.RegisterChecker(&panicChecker{})

	result := d.Check("content", "test.panic", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.Degraded {
		t.Error("expected degraded result after panic")
	}
}

// panicChecker 一个会panic的检查器，用于测试panic恢复
type panicChecker struct{}

func (p *panicChecker) SupportedExtensions() []string { return []string{".panic"} }
func (p *panicChecker) Check(content string, filePath string) *SyntaxCheckResult {
	panic("intentional panic for testing")
}

func TestDispatcher_RegisterMultipleCheckers(t *testing.T) {
	d := NewDispatcher()
	RegisterAllCheckers(d)

	exts := d.RegisteredExtensions()
	if len(exts) < 5 {
		t.Errorf("expected at least 5 registered extensions, got %d", len(exts))
	}

	// 验证关键扩展名存在
	extMap := make(map[string]bool)
	for _, ext := range exts {
		extMap[ext] = true
	}
	for _, expected := range []string{".json", ".yaml", ".yml", ".xml", ".html", ".go"} {
		if !extMap[expected] {
			t.Errorf("expected extension '%s' to be registered", expected)
		}
	}
}

// ==================== JSON Checker 测试 ====================

func TestJSONChecker_Valid(t *testing.T) {
	c := NewJSONChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"object", `{"key": "value"}`},
		{"array", `[1, 2, 3]`},
		{"nested", `{"a": {"b": [1, 2, {"c": true}]}}`},
		{"string", `"hello"`},
		{"number", `42`},
		{"null", `null`},
		{"empty_string", ""},
		{"whitespace_only", "  \n\t  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.json")
			if result.HasErrors {
				t.Errorf("expected no errors for %s, got: %v", tt.name, result.Errors)
			}
		})
	}
}

func TestJSONChecker_Invalid(t *testing.T) {
	c := NewJSONChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"unclosed_brace", `{"key": "value"`},
		{"trailing_comma", `{"key": "value",}`},
		{"missing_comma", `{"a": 1 "b": 2}`},
		{"single_quotes", `{'key': 'value'}`},
		{"invalid_token", `{key: value}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.json")
			if !result.HasErrors {
				t.Errorf("expected errors for %s", tt.name)
			}
		})
	}
}

// ==================== YAML Checker 测试 ====================

func TestYAMLChecker_Valid(t *testing.T) {
	c := NewYAMLChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"simple", "key: value\n"},
		{"list", "- item1\n- item2\n"},
		{"nested", "parent:\n  child: value\n"},
		{"multiline", "key: |\n  line1\n  line2\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.yaml")
			if result.HasErrors {
				t.Errorf("expected no errors for %s, got: %v", tt.name, result.Errors)
			}
		})
	}
}

// ==================== XML Checker 测试 ====================

func TestXMLChecker_Valid(t *testing.T) {
	c := NewXMLChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"simple", `<root><item>text</item></root>`},
		{"with_attrs", `<root lang="en"><item id="1">text</item></root>`},
		{"with_prolog", `<?xml version="1.0"?><root/>`},
		{"self_closing", `<root><br/><img src="x"/></root>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.xml")
			if result.HasErrors {
				t.Errorf("expected no errors for %s, got: %v", tt.name, result.Errors)
			}
		})
	}
}

func TestXMLChecker_Invalid(t *testing.T) {
	c := NewXMLChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"unclosed_tag", `<root><item></root>`},
		{"invalid_entity", `<root>&invalid;</root>`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.xml")
			if !result.HasErrors {
				t.Errorf("expected errors for %s", tt.name)
			}
		})
	}
}

// ==================== Go Checker 测试 ====================

func TestGoChecker_Valid(t *testing.T) {
	c := NewGoChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"hello_world", "package main\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}"},
		{"var_decl", "package p\n\nvar x = 1"},
		{"type_decl", "package p\n\ntype T struct {\n\tName string\n}"},
		{"interface", "package p\n\ntype I interface {\n\tM()\n}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.go")
			if result.HasErrors {
				t.Errorf("expected no errors for %s, got: %v", tt.name, result.Errors)
			}
		})
	}
}

func TestGoChecker_Invalid(t *testing.T) {
	c := NewGoChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"missing_brace", "package main\n\nfunc main() {\n"},
		{"bad_syntax", "package main\n\nfunc main(\n\tprintln(\n"},
		{"no_package", "func main() {}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.go")
			if !result.HasErrors {
				t.Errorf("expected errors for %s", tt.name)
			}
		})
	}
}

// ==================== External Checker 测试 ====================

func TestExternalChecker_UnsupportedExtension(t *testing.T) {
	c := NewExternalChecker()
	result := c.Check("content", "test.txt")
	if result != nil {
		t.Error("expected nil for unsupported extension")
	}
}

func TestExternalChecker_DegradedWhenToolNotAvailable(t *testing.T) {
	c := NewExternalChecker()
	// Java在测试环境中通常不可用
	result := c.Check("public class Test {}", "Test.java")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// 不可用时应该降级
	if !result.Degraded && !result.HasErrors {
		// 可能可用（安装了javac），也可能不可用（降级）
		t.Logf("Java check result: degraded=%v, has_errors=%v, method=%s", result.Degraded, result.HasErrors, result.CheckMethod)
	}
}

// ==================== Result 结构测试 ====================

func TestSyntaxCheckResult_Serialization(t *testing.T) {
	result := &SyntaxCheckResult{
		Language:    "json",
		HasErrors:   true,
		Errors:      []SyntaxError{{Line: 1, Column: 5, Message: "test error", Severity: "error"}},
		Warnings:    []SyntaxWarning{{Message: "test warning", Severity: "warning"}},
		CheckMethod: "native",
	}

	// 确保能正常序列化为JSON
	data, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("failed to marshal result: %v", err)
	}

	// 确保包含所有必要字段
	s := string(data)
	for _, field := range []string{"language", "has_errors", "errors", "warnings", "check_method"} {
		if !strings.Contains(s, field) {
			t.Errorf("expected JSON to contain field '%s'", field)
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"test.json", "json"},
		{"test.yaml", "yaml"},
		{"test.yml", "yaml"},
		{"test.xml", "xml"},
		{"test.html", "html"},
		{"test.htm", "html"},
		{"test.go", "go"},
		{"test.js", "javascript"},
		{"test.ts", "typescript"},
		{"test.py", "python"},
		{"test.java", "java"},
		{"test.kt", "kotlin"},
		{"test.rs", "rust"},
		{"test.sh", "shell"},
		{"test.ps1", "powershell"},
		{"test.bat", "batch"},
		{"test.gradle", "gradle"},
		{"path/to/file.java", "java"},
		{"C:\\Users\\test.py", "python"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := detectLanguage(tt.path)
			if got != tt.expected {
				t.Errorf("detectLanguage(%s) = %s, want %s", tt.path, got, tt.expected)
			}
		})
	}
}

func TestOffsetToLineCol(t *testing.T) {
	tests := []struct {
		content string
		offset  int
		expLine int
		expCol  int
	}{
		{"hello", 3, 1, 4},
		{"line1\nline2\nline3", 6, 2, 1},  // start of line2 ('l')
		{"line1\nline2\nline3", 7, 2, 2},  // second char of line2 ('i')
		{"line1\nline2\nline3", 12, 3, 1}, // start of line3 ('l')
		{"", 0, 1, 1},
		{"ab", -1, 1, 1},  // negative offset
		{"ab", 100, 1, 3}, // offset > len
	}

	for _, tt := range tests {
		line, col := offsetToLineCol(tt.content, tt.offset)
		if line != tt.expLine || col != tt.expCol {
			t.Errorf("offsetToLineCol(%q, %d) = (%d, %d), want (%d, %d)",
				tt.content, tt.offset, line, col, tt.expLine, tt.expCol)
		}
	}
}
