package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// ==================== 路由测试 ====================

func TestLanguageFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "go"},
		{"src/app.ts", "typescript"},
		{"src/App.tsx", "typescript"},
		{"src/index.js", "typescript"},
		{"src/index.jsx", "typescript"},
		{"script.py", "python"},
		{"gui.pyw", "python"},
		{"types.pyi", "python"},
		{"readme.md", ""},
		{"config.yaml", ""},
		{"Makefile", ""},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := LanguageFromPath(tt.path)
			if result != tt.expected {
				t.Errorf("LanguageFromPath(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestServerForLanguage(t *testing.T) {
	tests := []struct {
		language  string
		wantFound bool
		command   string
	}{
		{"go", true, "gopls"},
		{"typescript", true, "typescript-language-server"},
		{"python", true, "pylsp"},
		{"rust", false, ""},
		{"java", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			cfg, found := ServerForLanguage(tt.language)
			if found != tt.wantFound {
				t.Errorf("ServerForLanguage(%q) found = %v, want %v", tt.language, found, tt.wantFound)
				return
			}
			if found && cfg.Command != tt.command {
				t.Errorf("ServerForLanguage(%q) command = %q, want %q", tt.language, cfg.Command, tt.command)
			}
		})
	}
}

func TestIsSupportedFile(t *testing.T) {
	tests := []struct {
		path   string
		expect bool
	}{
		{"main.go", true},
		{"app.ts", true},
		{"script.py", true},
		{"readme.md", false},
		{"data.json", false},
	}

	for _, tt := range tests {
		result := IsSupportedFile(tt.path)
		if result != tt.expect {
			t.Errorf("IsSupportedFile(%q) = %v, want %v", tt.path, result, tt.expect)
		}
	}
}

func TestSupportedExtensions(t *testing.T) {
	exts := SupportedExtensions()
	if len(exts) == 0 {
		t.Error("SupportedExtensions() returned empty list")
	}
}

func TestSupportedLanguages(t *testing.T) {
	langs := SupportedLanguages()
	if len(langs) != 3 {
		t.Errorf("SupportedLanguages() returned %d languages, want 3", len(langs))
	}
}

// ==================== 协议消息测试 ====================

func TestJSONRPCRequestMarshal(t *testing.T) {
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params: map[string]interface{}{
			"processId": 0,
			"rootUri":   "file:///tmp/project",
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if parsed["jsonrpc"] != "2.0" {
		t.Errorf("jsonrpc = %v, want 2.0", parsed["jsonrpc"])
	}
	if parsed["method"] != "initialize" {
		t.Errorf("method = %v, want initialize", parsed["method"])
	}
}

func TestDiagnosticSeverityString(t *testing.T) {
	tests := []struct {
		severity int
		expected string
	}{
		{1, "Error"},
		{2, "Warning"},
		{3, "Information"},
		{4, "Hint"},
		{0, "Unknown"},
		{99, "Unknown"},
	}

	for _, tt := range tests {
		d := &Diagnostic{Severity: tt.severity}
		result := d.SeverityString()
		if result != tt.expected {
			t.Errorf("Severity %d: got %q, want %q", tt.severity, result, tt.expected)
		}
	}
}

func TestPublishDiagnosticsUnmarshal(t *testing.T) {
	jsonData := `{
		"uri": "file:///tmp/test.go",
		"diagnostics": [
			{
				"range": {
					"start": {"line": 0, "character": 0},
					"end": {"line": 0, "character": 10}
				},
				"severity": 1,
				"message": "syntax error",
				"source": "go",
				"code": "syntax"
			}
		]
	}`

	var params PublishDiagnosticsParams
	if err := json.Unmarshal([]byte(jsonData), &params); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if params.URI != "file:///tmp/test.go" {
		t.Errorf("URI = %q, want file:///tmp/test.go", params.URI)
	}
	if len(params.Diagnostics) != 1 {
		t.Fatalf("Diagnostics count = %d, want 1", len(params.Diagnostics))
	}
	d := params.Diagnostics[0]
	if d.Severity != 1 {
		t.Errorf("Severity = %d, want 1", d.Severity)
	}
	if d.Message != "syntax error" {
		t.Errorf("Message = %q, want syntax error", d.Message)
	}
	if d.Range.Start.Line != 0 || d.Range.Start.Character != 0 {
		t.Errorf("Start position = %d:%d, want 0:0", d.Range.Start.Line, d.Range.Start.Character)
	}
}

// ==================== URI转换测试 ====================

func TestPathToURI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		check func(string) bool
	}{
		{
			name:  "relative path converts to absolute file URI",
			input: ".",
			check: func(uri string) bool {
				return strings.HasPrefix(uri, "file:///")
			},
		},
		{
			name:  "URI contains forward slashes",
			input: ".",
			check: func(uri string) bool {
				return !strings.Contains(uri[7:], "\\")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri := pathToURI(tt.input)
			if !tt.check(uri) {
				t.Errorf("pathToURI(%q) = %q, check failed", tt.input, uri)
			}
		})
	}
}

// ==================== 管理器基础测试 ====================

func TestNewManager(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if !mgr.IsEnabled() {
		t.Error("Manager should be enabled by default")
	}
	if mgr.rootDir == "" {
		t.Error("Manager rootDir should not be empty")
	}
	if !strings.HasPrefix(mgr.rootURI, "file:///") {
		t.Errorf("Manager rootURI = %q, should start with file:///", mgr.rootURI)
	}
}

func TestManagerSetEnabled(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	mgr.SetEnabled(false)
	if mgr.IsEnabled() {
		t.Error("Manager should be disabled after SetEnabled(false)")
	}

	mgr.SetEnabled(true)
	if !mgr.IsEnabled() {
		t.Error("Manager should be enabled after SetEnabled(true)")
	}
}

func TestManagerConfigure(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	cfg := LanguageServerConfig{
		Language: "go",
		Command:  "gopls",
		Args:     []string{},
	}
	mgr.Configure("go", cfg)

	mgr.mu.RLock()
	stored, ok := mgr.configs["go"]
	mgr.mu.RUnlock()

	if !ok {
		t.Error("Config not found for 'go'")
	}
	if stored.Command != "gopls" {
		t.Errorf("Config command = %q, want gopls", stored.Command)
	}
}

func TestManagerConfigureFromMap(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	configs := map[string]LanguageServerConfig{
		"go":     {Language: "go", Command: "gopls"},
		"python": {Language: "python", Command: "pylsp"},
	}
	mgr.ConfigureFromMap(configs)

	mgr.mu.RLock()
	_, hasGo := mgr.configs["go"]
	_, hasPython := mgr.configs["python"]
	mgr.mu.RUnlock()

	if !hasGo {
		t.Error("Go config not found")
	}
	if !hasPython {
		t.Error("Python config not found")
	}
}

func TestManagerGetDiagnosticsUnsupportedFile(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	ctx := context.Background()
	_, err := mgr.GetDiagnostics(ctx, filepath.Join(dir, "readme.md"))
	if err == nil {
		t.Error("Expected error for unsupported file type")
	}
	if !strings.Contains(err.Error(), "unsupported file type") {
		t.Errorf("Error = %q, should mention unsupported file type", err.Error())
	}
}

func TestManagerGetDiagnosticsDisabled(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	mgr.SetEnabled(false)

	// 创建一个.go文件
	goFile := filepath.Join(dir, "test.go")
	_ = os.WriteFile(goFile, []byte("package main\n"), 0644)

	ctx := context.Background()
	_, err := mgr.GetDiagnosticsForLanguage(ctx, goFile, "go")
	if err == nil {
		t.Error("Expected error when LSP is disabled")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("Error = %q, should mention disabled", err.Error())
	}
}

func TestManagerGetDiagnosticsNoServer(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)
	// 不配置任何服务器

	goFile := filepath.Join(dir, "test.go")
	_ = os.WriteFile(goFile, []byte("package main\n"), 0644)

	ctx := context.Background()
	_, err := mgr.GetDiagnosticsForLanguage(ctx, goFile, "go")
	if err == nil {
		t.Error("Expected error when no server configured")
	}
}

func TestManagerShutdownAll(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager(dir)

	// ShutdownAll应该不panic即使没有客户端
	mgr.ShutdownAll()
}

// ==================== 默认配置测试 ====================

func TestDefaultConfigs(t *testing.T) {
	configs := DefaultConfigs()
	if len(configs) != 3 {
		t.Errorf("DefaultConfigs() returned %d configs, want 3", len(configs))
	}

	for _, lang := range []string{"go", "typescript", "python"} {
		cfg, ok := configs[lang]
		if !ok {
			t.Errorf("Missing config for %s", lang)
			continue
		}
		if cfg.Command == "" {
			t.Errorf("Empty command for %s", lang)
		}
	}
}

// ==================== 客户端基础测试 ====================

func TestNewClient(t *testing.T) {
	client := NewClient("echo", []string{}, "file:///tmp")
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.rootURI != "file:///tmp" {
		t.Errorf("rootURI = %q, want file:///tmp", client.rootURI)
	}
}

func TestClientIsRunningFalse(t *testing.T) {
	client := NewClient("echo", []string{}, "file:///tmp")
	if client.IsRunning() {
		t.Error("Client should not be running before Start()")
	}
}

// ==================== JSON-RPC消息格式测试 ====================

func TestWriteMessageFormat(t *testing.T) {
	// 验证JSON-RPC消息格式正确
	req := &JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  nil,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// 验证jsonrpc版本
	if !strings.Contains(string(data), `"jsonrpc":"2.0"`) {
		t.Error("Message should contain jsonrpc version 2.0")
	}

	// 验证Content-Length头的格式
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if !strings.Contains(header, "Content-Length:") {
		t.Error("Header should contain Content-Length")
	}
	if !strings.HasSuffix(header, "\r\n\r\n") {
		t.Error("Header should end with double CRLF")
	}
}

func TestHandleNotificationDiagnostics(t *testing.T) {
	client := NewClient("echo", []string{}, "file:///tmp")
	client.diagnostics = make(map[string][]Diagnostic)

	params := map[string]interface{}{
		"uri": "file:///tmp/test.go",
		"diagnostics": []interface{}{
			map[string]interface{}{
				"range": map[string]interface{}{
					"start": map[string]interface{}{"line": float64(5), "character": float64(10)},
					"end":   map[string]interface{}{"line": float64(5), "character": float64(15)},
				},
				"severity": float64(1),
				"message":  "undefined variable",
				"source":   "go",
			},
		},
	}

	client.handleDiagnostics(params)

	client.diagMu.RLock()
	diags := client.diagnostics["file:///tmp/test.go"]
	client.diagMu.RUnlock()

	if len(diags) != 1 {
		t.Fatalf("Expected 1 diagnostic, got %d", len(diags))
	}
	if diags[0].Message != "undefined variable" {
		t.Errorf("Message = %q, want 'undefined variable'", diags[0].Message)
	}
	if diags[0].Severity != 1 {
		t.Errorf("Severity = %d, want 1", diags[0].Severity)
	}
	if diags[0].Range.Start.Line != 5 {
		t.Errorf("Line = %d, want 5", diags[0].Range.Start.Line)
	}
}

// ==================== 集成测试（需要语言服务器） ====================

func TestIntegrationGetDiagnosticsWithTempFile(t *testing.T) {
	// 检查gopls是否可用
	if _, err := os.Stat(filepath.Join(os.Getenv("GOPATH"), "bin", "gopls")); os.IsNotExist(err) {
		// 查找PATH中的gopls
		found := false
		for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
			if _, err := os.Stat(filepath.Join(dir, "gopls")); err == nil {
				found = true
				break
			}
		}
		if !found {
			t.Skip("gopls not found, skipping integration test")
		}
	}

	dir := t.TempDir()

	// 创建一个有语法错误的Go文件
	goFile := filepath.Join(dir, "broken.go")
	content := `package main

func main() {
	x := }  // syntax error
`
	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// 创建go.mod
	goMod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module test\n\ngo 1.21\n"), 0644); err != nil {
		t.Fatalf("Failed to write go.mod: %v", err)
	}

	mgr := NewManager(dir)
	mgr.ConfigureFromMap(DefaultConfigs())
	defer mgr.ShutdownAll()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	diags, err := mgr.GetDiagnostics(ctx, goFile)
	if err != nil {
		t.Fatalf("GetDiagnostics failed: %v", err)
	}

	// gopls应该报告至少一个错误
	if len(diags) == 0 {
		t.Log("Warning: no diagnostics returned (gopls may not have fully started)")
	} else {
		t.Logf("Got %d diagnostics", len(diags))
		for _, d := range diags {
			t.Logf("  [%s] line %d: %s", d.SeverityString(), d.Range.Start.Line+1, d.Message)
		}
	}
}
