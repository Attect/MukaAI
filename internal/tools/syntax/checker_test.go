package syntax

import (
	"encoding/json"
	"strings"
	"testing"

	"golang.org/x/net/html"
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
	for _, expected := range []string{".json", ".yaml", ".yml", ".xml", ".html", ".go", ".toml", ".css", ".sql", ".properties"} {
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

// ==================== TOML Checker 测试 ====================

func TestTOMLChecker_Valid(t *testing.T) {
	c := NewTOMLChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"simple", `key = "value"`},
		{"section", `[database]
host = "localhost"
port = 5432`},
		{"array", `fruits = ["apple", "banana", "cherry"]`},
		{"nested", `[owner]
name = "Tom"
[owner.bio]
age = 30`},
		{"empty", ""},
		{"whitespace_only", "  \n\t  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.toml")
			if result.HasErrors {
				t.Errorf("expected no errors for %s, got: %v", tt.name, result.Errors)
			}
			if result.Language != "toml" {
				t.Errorf("expected language 'toml', got '%s'", result.Language)
			}
		})
	}
}

func TestTOMLChecker_Invalid(t *testing.T) {
	c := NewTOMLChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"missing_equals", `key "value"`},
		{"invalid_section", `[section`},
		{"duplicate_key", `key = "1"
key = "2"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.toml")
			if !result.HasErrors {
				t.Errorf("expected errors for %s", tt.name)
			}
			if len(result.Errors) == 0 {
				t.Error("expected at least one error")
			}
		})
	}
}

// ==================== CSS Checker 测试 ====================

func TestCSSChecker_Valid(t *testing.T) {
	c := NewCSSChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"simple_rule", `body { color: red; }`},
		{"nested", `.container { display: flex; }
.container .item { flex: 1; }`},
		{"at_rule", `@media (max-width: 600px) { body { font-size: 14px; } }`},
		{"empty", ""},
		{"whitespace_only", "  \n\t  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.css")
			if result.HasErrors {
				t.Errorf("expected no errors for %s, got: %v", tt.name, result.Errors)
			}
			if result.Language != "css" {
				t.Errorf("expected language 'css', got '%s'", result.Language)
			}
		})
	}
}

func TestCSSChecker_Invalid(t *testing.T) {
	c := NewCSSChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"unclosed_brace", `body { color: red;`},
		{"extra_closing_brace", `}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.css")
			if !result.HasErrors {
				t.Errorf("expected errors for %s", tt.name)
			}
		})
	}
}

// ==================== SQL Checker 测试 ====================

func TestSQLChecker_Valid(t *testing.T) {
	c := NewSQLChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"simple_select", `SELECT * FROM users WHERE id = 1;`},
		{"multi_statement", `CREATE TABLE users (id INT, name VARCHAR(100));
INSERT INTO users VALUES (1, 'Alice');
SELECT * FROM users;`},
		{"with_comments", `-- This is a comment
SELECT /* inline */ name FROM users;`},
		{"string_with_quote", `SELECT 'it''s fine' AS val;`},
		{"bracket_ident", `SELECT [name] FROM [my-table];`},
		{"empty", ""},
		{"whitespace_only", "  \n\t  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.sql")
			if result.HasErrors {
				t.Errorf("expected no errors for %s, got: %v", tt.name, result.Errors)
			}
			if result.Language != "sql" {
				t.Errorf("expected language 'sql', got '%s'", result.Language)
			}
		})
	}
}

func TestSQLChecker_Invalid(t *testing.T) {
	c := NewSQLChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"unclosed_string", `SELECT 'unclosed FROM users;`},
		{"unclosed_paren", `SELECT * FROM users WHERE id IN (1, 2;`},
		{"extra_close_paren", `SELECT *) FROM users;`},
		{"unclosed_comment", `/* this is never closed`},
		{"unclosed_ident", `SELECT "name FROM users;`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.sql")
			if !result.HasErrors {
				t.Errorf("expected errors for %s", tt.name)
			}
			if len(result.Errors) == 0 {
				t.Error("expected at least one error")
			}
		})
	}
}

// ==================== Properties Checker 测试 ====================

func TestPropertiesChecker_Valid(t *testing.T) {
	c := NewPropertiesChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"simple", "key=value"},
		{"colon_sep", "key:value"},
		{"space_sep", "key value"},
		{"comment", "# This is a comment\nkey=value"},
		{"bang_comment", "! Another comment\nkey=value"},
		{"unicode_escape", `name=\u0041lice`},
		{"escape_sequences", `path=C:\\Users\\test
newline=line1\nline2
tab=col1\tcol2`},
		{"continuation", `key=line1 \
line2`},
		{"empty", ""},
		{"whitespace_only", "  \n\t  "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.properties")
			if result.HasErrors {
				t.Errorf("expected no errors for %s, got: %v", tt.name, result.Errors)
			}
			if result.Language != "properties" {
				t.Errorf("expected language 'properties', got '%s'", result.Language)
			}
		})
	}
}

func TestPropertiesChecker_Invalid(t *testing.T) {
	c := NewPropertiesChecker()
	tests := []struct {
		name    string
		content string
	}{
		{"invalid_unicode_short", `name=\u00`},
		{"invalid_unicode_char", `name=\u00GG`},
		{"trailing_continuation", `key=value\`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.Check(tt.content, "test.properties")
			if !result.HasErrors {
				t.Errorf("expected errors for %s", tt.name)
			}
		})
	}
}

// ==================== Dispatcher 注册测试 ====================

func TestDispatcher_Check_TOML(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewTOMLChecker())

	result := d.Check(`key = "value"`, "test.toml", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestDispatcher_Check_CSS(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewCSSChecker())

	result := d.Check("body { color: red; }", "test.css", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestDispatcher_Check_SQL(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewSQLChecker())

	result := d.Check("SELECT * FROM users;", "test.sql", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestDispatcher_Check_Properties(t *testing.T) {
	d := NewDispatcher()
	d.RegisterChecker(NewPropertiesChecker())

	result := d.Check("key=value", "test.properties", 100)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors, got: %v", result.Errors)
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
		{"test.toml", "toml"},
		{"test.css", "css"},
		{"test.sql", "sql"},
		{"test.properties", "properties"},
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

// ==================== BAT/GradleKTS 扩展名注册测试 ====================

func TestExternalChecker_BATExtension(t *testing.T) {
	c := NewExternalChecker()

	// .bat 和 .cmd 应该被ExternalChecker支持
	exts := c.SupportedExtensions()
	extMap := make(map[string]bool)
	for _, ext := range exts {
		extMap[ext] = true
	}
	if !extMap[".bat"] {
		t.Error("expected .bat to be supported by ExternalChecker")
	}
	if !extMap[".cmd"] {
		t.Error("expected .cmd to be supported by ExternalChecker")
	}
}

func TestExternalChecker_GradleKTSExtension(t *testing.T) {
	c := NewExternalChecker()

	// .gradle.kts 应该被ExternalChecker支持
	exts := c.SupportedExtensions()
	extMap := make(map[string]bool)
	for _, ext := range exts {
		extMap[ext] = true
	}
	if !extMap[".gradle.kts"] {
		t.Error("expected .gradle.kts to be supported by ExternalChecker")
	}
}

func TestExternalChecker_BATCheck(t *testing.T) {
	c := NewExternalChecker()

	// 测试.bat文件能触发检查（降级或实际检查都是合理的）
	result := c.Check("@echo off\necho hello\n", "test.bat")
	// 在非Windows平台期望降级，Windows平台可能成功或降级
	if result == nil {
		t.Error("expected non-nil result for .bat file")
	}
	if result != nil {
		t.Logf("BAT check: language=%s, degraded=%v, method=%s", result.Language, result.Degraded, result.CheckMethod)
	}
}

func TestExternalChecker_GradleKTSCheck(t *testing.T) {
	c := NewExternalChecker()

	// 测试.gradle.kts文件（双扩展名）能触发Gradle检查
	result := c.Check(`plugins { java }`, "build.gradle.kts")
	if result == nil {
		t.Error("expected non-nil result for .gradle.kts file")
	}
	if result != nil {
		t.Logf("Gradle KTS check: language=%s, degraded=%v, method=%s", result.Language, result.Degraded, result.CheckMethod)
	}
}

func TestDetectLanguage_GradleKTS(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"build.gradle.kts", "gradle"},
		{"settings.gradle.kts", "gradle"},
		{"path/to/build.gradle.kts", "gradle"},
		{"test.kts", "kotlin"},
		{"build.gradle", "gradle"},
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

func TestDispatcher_RegisterAllIncludesBATAndGradleKTS(t *testing.T) {
	d := NewDispatcher()
	RegisterAllCheckers(d)

	exts := d.RegisteredExtensions()
	extMap := make(map[string]bool)
	for _, ext := range exts {
		extMap[ext] = true
	}

	// BAT扩展名
	if !extMap[".bat"] {
		t.Error("expected .bat to be registered")
	}
	if !extMap[".cmd"] {
		t.Error("expected .cmd to be registered")
	}
	// GradleKTS扩展名
	if !extMap[".gradle.kts"] {
		t.Error("expected .gradle.kts to be registered")
	}
}

// ==================== HTML+JS 嵌入式JavaScript检查测试 ====================

func TestHTMLChecker_NoScript(t *testing.T) {
	c := NewHTMLChecker()
	result := c.Check(`<!DOCTYPE html><html><body><p>Hello</p></body></html>`, "test.html")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors, got: %v", result.Errors)
	}
}

func TestHTMLChecker_ValidScript(t *testing.T) {
	c := NewHTMLChecker()
	html := `<!DOCTYPE html><html><body>
<script>console.log("hello");</script>
</body></html>`
	result := c.Check(html, "test.html")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	// node可用时检查JS，不可用时HTML结构检查仍通过
	t.Logf("HTML+JS check: has_errors=%v, degraded=%v, warnings=%d", result.HasErrors, result.Degraded, len(result.Warnings))
}

func TestHTMLChecker_InvalidScript(t *testing.T) {
	c := NewHTMLChecker()
	html := `<!DOCTYPE html><html><body>
<script>function foo( { }</script>
</body></html>`
	result := c.Check(html, "test.html")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	t.Logf("HTML+invalid JS check: has_errors=%v, warnings=%d", result.HasErrors, len(result.Warnings))
}

func TestHTMLChecker_MultipleScripts(t *testing.T) {
	c := NewHTMLChecker()
	html := `<!DOCTYPE html><html><body>
<script>var x = 1;</script>
<script>var y = 2;</script>
</body></html>`
	result := c.Check(html, "test.html")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	t.Logf("HTML+multiple scripts: has_errors=%v, warnings=%d", result.HasErrors, len(result.Warnings))
}

func TestHTMLChecker_ImportMapSkipped(t *testing.T) {
	c := NewHTMLChecker()
	// importmap类型的script不应被作为JS检查
	html := `<!DOCTYPE html><html><body>
<script type="importmap">{ "imports": {} }</script>
</body></html>`
	result := c.Check(html, "test.html")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors for importmap script, got: %v", result.Errors)
	}
	// importmap中的JSON内容不应产生JS语法错误
	t.Logf("HTML+importmap: has_errors=%v, warnings=%d", result.HasErrors, len(result.Warnings))
}

func TestHTMLChecker_NonJSScriptTypes(t *testing.T) {
	c := NewHTMLChecker()
	// 各种非JS类型的script标签应被跳过
	html := `<!DOCTYPE html><html><body>
<script type="application/json">{ "key": "value" }</script>
<script type="text/template"><div>{{name}}</div></script>
<script type="importmap">{ "imports": {} }</script>
</body></html>`
	result := c.Check(html, "test.html")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors for non-JS scripts, got: %v", result.Errors)
	}
}

func TestHTMLChecker_EmptyScript(t *testing.T) {
	c := NewHTMLChecker()
	html := `<!DOCTYPE html><html><body>
<script></script>
<script>    </script>
</body></html>`
	result := c.Check(html, "test.html")
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasErrors {
		t.Errorf("expected no errors for empty scripts, got: %v", result.Errors)
	}
}

func TestIsJSScriptType(t *testing.T) {
	tests := []struct {
		scriptType string
		expected   bool
	}{
		{"", true},
		{"text/javascript", true},
		{"module", true},
		{"application/javascript", true},
		{"TEXT/JAVASCRIPT", true},
		{"Module", true},
		{"importmap", false},
		{"application/json", false},
		{"text/template", false},
		{"text/x-custom", false},
	}

	for _, tt := range tests {
		t.Run(tt.scriptType, func(t *testing.T) {
			got := isJSScriptType(tt.scriptType)
			if got != tt.expected {
				t.Errorf("isJSScriptType(%q) = %v, want %v", tt.scriptType, got, tt.expected)
			}
		})
	}
}

func TestExtractScriptContents(t *testing.T) {
	htmlContent := `<!DOCTYPE html><html><body>
<script>var a = 1;</script>
<script type="module">import { x } from 'y';</script>
<script type="importmap">{ "imports": {} }</script>
<script type="text/javascript">var b = 2;</script>
</body></html>`

	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		t.Fatalf("failed to parse HTML: %v", err)
	}

	scripts := extractScriptContents(doc)
	if len(scripts) != 4 {
		t.Fatalf("expected 4 script elements, got %d", len(scripts))
	}

	// 验证各script内容
	expectedContents := []string{
		"var a = 1;",
		"import { x } from 'y';",
		`{ "imports": {} }`,
		"var b = 2;",
	}
	for i, script := range scripts {
		if !strings.Contains(script.Content, strings.TrimSpace(expectedContents[i])) {
			t.Errorf("script[%d]: expected content containing %q, got %q", i, expectedContents[i], script.Content)
		}
	}

	// 验证type属性
	expectedTypes := []string{"", "module", "importmap", "text/javascript"}
	for i, script := range scripts {
		if script.Type != expectedTypes[i] {
			t.Errorf("script[%d]: expected type %q, got %q", i, expectedTypes[i], script.Type)
		}
	}

	// 验证索引
	for i, script := range scripts {
		if script.Index != i {
			t.Errorf("script[%d]: expected index %d, got %d", i, i, script.Index)
		}
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
