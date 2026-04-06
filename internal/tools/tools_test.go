package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// ==================== Types Tests ====================

func TestToolResult_ToJSON(t *testing.T) {
	tests := []struct {
		name   string
		result *ToolResult
		want   string
	}{
		{
			name:   "success result",
			result: NewSuccessResult(map[string]string{"key": "value"}),
			want:   `"success": true`,
		},
		{
			name:   "error result",
			result: NewErrorResult("something went wrong"),
			want:   `"success": false`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.result.ToJSON()
			if !contains(got, tt.want) {
				t.Errorf("ToJSON() = %v, want to contain %v", got, tt.want)
			}
		})
	}
}

func TestToolParameter_ToMap(t *testing.T) {
	param := &ToolParameter{
		Type:        "string",
		Description: "test parameter",
		Enum:        []string{"a", "b"},
	}

	m := param.ToMap()

	if m["type"] != "string" {
		t.Errorf("expected type string, got %v", m["type"])
	}

	if m["description"] != "test parameter" {
		t.Errorf("expected description, got %v", m["description"])
	}
}

func TestBuildSchema(t *testing.T) {
	properties := map[string]*ToolParameter{
		"path": {
			Type:        "string",
			Description: "file path",
		},
	}

	schema := BuildSchema(properties, []string{"path"})

	if schema["type"] != "object" {
		t.Errorf("expected type object, got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found")
	}

	if _, ok := props["path"]; !ok {
		t.Error("path property not found")
	}

	required, ok := schema["required"].([]string)
	if !ok || len(required) != 1 || required[0] != "path" {
		t.Error("required field not correct")
	}
}

// ==================== Registry Tests ====================

func TestToolRegistry_RegisterTool(t *testing.T) {
	registry := NewToolRegistry()

	tool := NewReadFileTool()
	err := registry.RegisterTool(tool)
	if err != nil {
		t.Fatalf("failed to register tool: %v", err)
	}

	// 重复注册应该失败
	err = registry.RegisterTool(tool)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestToolRegistry_GetTool(t *testing.T) {
	registry := NewToolRegistry()
	tool := NewReadFileTool()
	registry.RegisterTool(tool)

	got, exists := registry.GetTool("read_file")
	if !exists {
		t.Fatal("tool not found")
	}

	if got.Name() != "read_file" {
		t.Errorf("expected read_file, got %v", got.Name())
	}

	// 不存在的工具
	_, exists = registry.GetTool("nonexistent")
	if exists {
		t.Error("expected nonexistent tool to not exist")
	}
}

func TestToolRegistry_GetAllToolSchemas(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterTool(NewReadFileTool())
	registry.RegisterTool(NewWriteFileTool())

	schemas := registry.GetAllToolSchemas()
	if len(schemas) != 2 {
		t.Errorf("expected 2 schemas, got %d", len(schemas))
	}

	for _, schema := range schemas {
		if schema.Type != "function" {
			t.Errorf("expected type function, got %v", schema.Type)
		}
		if schema.Function.Name == "" {
			t.Error("function name should not be empty")
		}
	}
}

func TestToolRegistry_ExecuteTool(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterTool(NewReadFileTool())

	// 创建临时文件
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("hello world"), 0644)

	// 执行工具
	result, err := registry.ExecuteTool(context.Background(), "read_file", map[string]interface{}{
		"path": tmpFile,
	})
	if err != nil {
		t.Fatalf("failed to execute tool: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

// ==================== Filesystem Tools Tests ====================

func TestReadFileTool(t *testing.T) {
	tool := NewReadFileTool()

	// 创建临时文件
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world"
	os.WriteFile(tmpFile, []byte(content), 0644)

	// 测试读取
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": tmpFile,
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data, ok := result.Data.(map[string]interface{})
	if !ok {
		t.Fatal("data is not a map")
	}

	if data["content"] != content {
		t.Errorf("expected content %v, got %v", content, data["content"])
	}

	// 测试相对路径（应该失败）
	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"path": "relative/path.txt",
	})
	if result.Success {
		t.Error("expected error for relative path")
	}

	// 测试不存在的文件
	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"path": filepath.Join(tmpDir, "nonexistent.txt"),
	})
	if result.Success {
		t.Error("expected error for nonexistent file")
	}
}

func TestWriteFileTool(t *testing.T) {
	tool := NewWriteFileTool()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	content := "hello world"

	// 测试写入
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path":    tmpFile,
		"content": content,
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// 验证文件内容
	readContent, _ := os.ReadFile(tmpFile)
	if string(readContent) != content {
		t.Errorf("expected content %v, got %v", content, string(readContent))
	}

	// 测试创建父目录
	nestedFile := filepath.Join(tmpDir, "a", "b", "c", "test.txt")
	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"path":    nestedFile,
		"content": content,
	})
	if !result.Success {
		t.Errorf("expected success for nested path, got error: %s", result.Error)
	}
}

func TestListDirectoryTool(t *testing.T) {
	tool := NewListDirectoryTool()

	tmpDir := t.TempDir()

	// 创建一些文件和目录
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755)
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644)

	// 测试非递归列出
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": tmpDir,
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data := result.Data.(map[string]interface{})
	count := data["count"].(int)
	if count != 3 { // 1 subdir + 2 files
		t.Errorf("expected 3 entries, got %d", count)
	}

	// 测试递归列出
	os.WriteFile(filepath.Join(tmpDir, "subdir", "file3.txt"), []byte("content3"), 0644)

	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"path":      tmpDir,
		"recursive": true,
	})

	data = result.Data.(map[string]interface{})
	count = data["count"].(int)
	if count != 4 { // 1 subdir + 2 files + 1 nested file
		t.Errorf("expected 4 entries, got %d", count)
	}
}

func TestDeleteFileTool(t *testing.T) {
	tool := NewDeleteFileTool()

	tmpDir := t.TempDir()

	// 测试删除文件
	tmpFile := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(tmpFile, []byte("content"), 0644)

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": tmpFile,
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// 验证文件已删除
	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("file should be deleted")
	}

	// 测试删除目录
	tmpSubDir := filepath.Join(tmpDir, "subdir")
	os.Mkdir(tmpSubDir, 0755)
	os.WriteFile(filepath.Join(tmpSubDir, "file.txt"), []byte("content"), 0644)

	// 非递归删除非空目录应该失败
	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"path": tmpSubDir,
	})
	if result.Success {
		t.Error("expected error for non-empty directory without recursive")
	}

	// 递归删除
	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"path":      tmpSubDir,
		"recursive": true,
	})
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

func TestCreateDirectoryTool(t *testing.T) {
	tool := NewCreateDirectoryTool()

	tmpDir := t.TempDir()

	// 测试创建目录
	newDir := filepath.Join(tmpDir, "newdir")
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"path": newDir,
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	// 验证目录存在
	info, err := os.Stat(newDir)
	if err != nil || !info.IsDir() {
		t.Error("directory should exist")
	}

	// 测试创建多级目录
	nestedDir := filepath.Join(tmpDir, "a", "b", "c")
	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"path": nestedDir,
	})
	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	// 测试已存在的目录
	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"path": newDir,
	})
	if !result.Success {
		t.Errorf("expected success for existing directory, got error: %s", result.Error)
	}
}

// ==================== Command Tools Tests ====================

func TestExecuteCommandTool(t *testing.T) {
	tool := NewExecuteCommandTool()

	var result *ToolResult
	var err error

	if runtime.GOOS == "windows" {
		result, err = tool.Execute(context.Background(), map[string]interface{}{
			"command": "echo",
			"args":    []interface{}{"hello"},
		})
	} else {
		result, err = tool.Execute(context.Background(), map[string]interface{}{
			"command": "echo hello",
		})
	}

	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data := result.Data.(*CommandResult)
	if data.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", data.ExitCode)
	}
}

func TestShellExecuteTool(t *testing.T) {
	tool := NewShellExecuteTool()

	cmd := "echo hello"

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": cmd,
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data := result.Data.(*CommandResult)
	if data.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", data.ExitCode)
	}
}

func TestCommandTimeout(t *testing.T) {
	tool := NewExecuteCommandTool()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "ping -n 10 127.0.0.1"
	} else {
		cmd = "sleep 10"
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": cmd,
		"timeout": 1, // 1 second timeout
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	data := result.Data.(*CommandResult)
	if !data.TimedOut {
		t.Error("expected timeout")
	}
}

func TestCommandWorkingDir(t *testing.T) {
	tool := NewShellExecuteTool()

	tmpDir := t.TempDir()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cd"
	} else {
		cmd = "pwd"
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command":     cmd,
		"working_dir": tmpDir,
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	data := result.Data.(*CommandResult)
	// 输出应该包含工作目录路径
	if !contains(data.Stdout, tmpDir) {
		t.Errorf("expected output to contain %s, got %s", tmpDir, data.Stdout)
	}
}

// ==================== Helper Functions ====================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ==================== Integration Tests ====================

func TestRegisterAllTools(t *testing.T) {
	registry := NewToolRegistry()

	// 注册所有文件系统工具
	if err := RegisterFilesystemTools(registry); err != nil {
		t.Fatalf("failed to register filesystem tools: %v", err)
	}

	// 注册所有命令工具
	if err := RegisterCommandTools(registry); err != nil {
		t.Fatalf("failed to register command tools: %v", err)
	}

	// 验证工具数量
	count := registry.ToolCount()
	if count != 7 { // 5 filesystem + 2 command
		t.Errorf("expected 7 tools, got %d", count)
	}

	// 验证所有工具都有有效的Schema
	schemas := registry.GetAllToolSchemas()
	for _, schema := range schemas {
		if schema.Function.Name == "" {
			t.Error("tool name should not be empty")
		}
		if schema.Function.Description == "" {
			t.Error("tool description should not be empty")
		}
		if schema.Function.Parameters == nil {
			t.Error("tool parameters should not be nil")
		}
	}
}

func TestGlobalRegistry(t *testing.T) {
	// 清理全局注册中心
	defaultRegistry = NewToolRegistry()

	// 注册工具
	MustRegisterTool(NewReadFileTool())

	// 验证可以通过全局函数访问
	_, exists := GetTool("read_file")
	if !exists {
		t.Error("tool should exist in global registry")
	}

	// 验证全局执行
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("content"), 0644)

	result, err := ExecuteTool(context.Background(), "read_file", map[string]interface{}{
		"path": tmpFile,
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

// ==================== Benchmark Tests ====================

func BenchmarkToolRegistry_ExecuteTool(b *testing.B) {
	registry := NewToolRegistry()
	registry.RegisterTool(NewReadFileTool())

	tmpFile := filepath.Join(b.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("content"), 0644)

	params := map[string]interface{}{
		"path": tmpFile,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.ExecuteTool(context.Background(), "read_file", params)
	}
}

func BenchmarkToolRegistry_GetAllToolSchemas(b *testing.B) {
	registry := NewToolRegistry()
	RegisterFilesystemTools(registry)
	RegisterCommandTools(registry)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.GetAllToolSchemas()
	}
}

// ==================== Edge Cases ====================

func TestReadFileTool_MissingPath(t *testing.T) {
	tool := NewReadFileTool()

	result, _ := tool.Execute(context.Background(), map[string]interface{}{})
	if result.Success {
		t.Error("expected error for missing path")
	}
}

func TestWriteFileTool_MissingContent(t *testing.T) {
	tool := NewWriteFileTool()

	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"path": filepath.Join(t.TempDir(), "test.txt"),
	})
	if result.Success {
		t.Error("expected error for missing content")
	}
}

func TestToolRegistry_RegisterNilTool(t *testing.T) {
	registry := NewToolRegistry()

	err := registry.RegisterTool(nil)
	if err == nil {
		t.Error("expected error for nil tool")
	}
}

func TestToolRegistry_ExecuteNonexistentTool(t *testing.T) {
	registry := NewToolRegistry()

	_, err := registry.ExecuteTool(context.Background(), "nonexistent", nil)
	if err == nil {
		t.Error("expected error for nonexistent tool")
	}
}

func TestExecuteCommandTool_EmptyCommand(t *testing.T) {
	tool := NewExecuteCommandTool()

	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"command": "",
	})
	if result.Success {
		t.Error("expected error for empty command")
	}
}

func TestToolResult_Metadata(t *testing.T) {
	result := &ToolResult{
		Success: false,
		Error:   "test error",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	}

	json := result.ToJSON()
	if !contains(json, "metadata") {
		t.Error("metadata should be in JSON output")
	}
}

func TestCommandResult_Fields(t *testing.T) {
	tool := NewShellExecuteTool()

	tmpDir := t.TempDir()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo test_output && echo test_error 1>&2"
	} else {
		cmd = "echo test_output && echo test_error >&2"
	}

	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command":     cmd,
		"working_dir": tmpDir,
		"timeout":     10,
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	data := result.Data.(*CommandResult)

	if data.Command != cmd {
		t.Errorf("command mismatch")
	}

	if data.WorkingDir != tmpDir {
		t.Errorf("working dir mismatch")
	}

	if data.Duration == "" {
		t.Error("duration should not be empty")
	}

	if !contains(data.Stdout, "test_output") {
		t.Errorf("stdout should contain 'test_output', got: %s", data.Stdout)
	}
}

func TestToolParameter_NestedProperties(t *testing.T) {
	param := &ToolParameter{
		Type: "object",
		Properties: map[string]*ToolParameter{
			"nested": {
				Type: "string",
			},
		},
	}

	m := param.ToMap()
	props, ok := m["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found")
	}

	if _, ok := props["nested"]; !ok {
		t.Error("nested property not found")
	}
}

func TestToolParameter_ArrayType(t *testing.T) {
	param := &ToolParameter{
		Type: "array",
		Items: &ToolParameter{
			Type: "string",
		},
	}

	m := param.ToMap()
	items, ok := m["items"].(map[string]interface{})
	if !ok {
		t.Fatal("items not found")
	}

	if items["type"] != "string" {
		t.Error("items type mismatch")
	}
}

func TestToolParameter_MinMaxConstraints(t *testing.T) {
	minLen := 1
	maxLen := 100
	minVal := 0.0
	maxVal := 100.0

	param := &ToolParameter{
		Type:      "string",
		MinLength: &minLen,
		MaxLength: &maxLen,
		Minimum:   &minVal,
		Maximum:   &maxVal,
	}

	m := param.ToMap()

	if m["minLength"] != minLen {
		t.Error("minLength mismatch")
	}

	if m["maxLength"] != maxLen {
		t.Error("maxLength mismatch")
	}

	if m["minimum"] != minVal {
		t.Error("minimum mismatch")
	}

	if m["maximum"] != maxVal {
		t.Error("maximum mismatch")
	}
}

func TestNewErrorResultWithError(t *testing.T) {
	result := NewErrorResultWithError("test error", nil)
	if result.Metadata != nil {
		t.Error("metadata should be nil when originalErr is nil")
	}

	result = NewErrorResultWithError("test error", context.DeadlineExceeded)
	if result.Metadata == nil {
		t.Error("metadata should not be nil when originalErr is not nil")
	}

	if result.Metadata["original_error"] == nil {
		t.Error("original_error should be in metadata")
	}
}

func TestExecuteToolWithDifferentParamTypes(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterTool(NewReadFileTool())

	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("content"), 0644)

	// Test with JSON string params (use forward slashes for JSON compatibility)
	jsonPath := strings.ReplaceAll(tmpFile, "\\", "/")
	_, err := registry.ExecuteTool(context.Background(), "read_file", `{"path":"`+jsonPath+`"}`)
	if err != nil {
		t.Errorf("execute with string params failed: %v", err)
	}

	// Test with nil params - should return error result, not error
	result, err := registry.ExecuteTool(context.Background(), "read_file", nil)
	if err != nil {
		t.Fatalf("execute should not return error: %v", err)
	}
	if result.Success {
		t.Error("expected error result for nil params")
	}
}

func TestListDirectoryTool_InvalidPath(t *testing.T) {
	tool := NewListDirectoryTool()

	// Test with relative path
	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"path": "relative/path",
	})
	if result.Success {
		t.Error("expected error for relative path")
	}

	// Test with file path instead of directory
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("content"), 0644)

	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"path": tmpFile,
	})
	if result.Success {
		t.Error("expected error for file path")
	}
}

func TestDeleteFileTool_InvalidPath(t *testing.T) {
	tool := NewDeleteFileTool()

	// Test with relative path
	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"path": "relative/path",
	})
	if result.Success {
		t.Error("expected error for relative path")
	}

	// Test with nonexistent path
	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"path": filepath.Join(t.TempDir(), "nonexistent"),
	})
	if result.Success {
		t.Error("expected error for nonexistent path")
	}
}

func TestCreateDirectoryTool_InvalidPath(t *testing.T) {
	tool := NewCreateDirectoryTool()

	// Test with relative path
	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"path": "relative/path",
	})
	if result.Success {
		t.Error("expected error for relative path")
	}
}

func TestWriteFileTool_InvalidPath(t *testing.T) {
	tool := NewWriteFileTool()

	// Test with relative path
	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"path":    "relative/path",
		"content": "test",
	})
	if result.Success {
		t.Error("expected error for relative path")
	}
}

func TestShellExecuteTool_TimeoutTypes(t *testing.T) {
	tool := NewShellExecuteTool()

	cmd := "echo test"

	// Test with float64 timeout
	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"command": cmd,
		"timeout": float64(10),
	})
	if !result.Success {
		t.Errorf("expected success with float64 timeout, got error: %s", result.Error)
	}

	// Test with int timeout
	result, _ = tool.Execute(context.Background(), map[string]interface{}{
		"command": cmd,
		"timeout": 10,
	})
	if !result.Success {
		t.Errorf("expected success with int timeout, got error: %s", result.Error)
	}
}

func TestExecuteCommandTool_WithEnv(t *testing.T) {
	tool := NewExecuteCommandTool()

	// Use shell_execute for environment variable expansion
	result, err := tool.Execute(context.Background(), map[string]interface{}{
		"command": "echo",
		"args":    []interface{}{"test_value"},
		"env": map[string]interface{}{
			"TEST_VAR": "test_value",
		},
	})
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	data := result.Data.(*CommandResult)
	if !contains(data.Stdout, "test_value") {
		t.Errorf("expected output to contain 'test_value', got: %s", data.Stdout)
	}
}

func TestGetAllToolSchemasJSON(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterTool(NewReadFileTool())

	jsonStr, err := registry.GetAllToolSchemasJSON()
	if err != nil {
		t.Fatalf("failed to get JSON: %v", err)
	}

	if jsonStr == "" {
		t.Error("JSON should not be empty")
	}

	if !contains(jsonStr, "read_file") {
		t.Error("JSON should contain tool name")
	}
}

func TestListToolNames(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterTool(NewReadFileTool())
	registry.RegisterTool(NewWriteFileTool())

	names := registry.ListToolNames()
	if len(names) != 2 {
		t.Errorf("expected 2 names, got %d", len(names))
	}
}

func TestDefaultRegistry(t *testing.T) {
	// Reset default registry
	defaultRegistry = NewToolRegistry()

	// Test DefaultRegistry function
	reg := DefaultRegistry()
	if reg == nil {
		t.Error("DefaultRegistry should not return nil")
	}

	// Test that it's the same instance
	reg2 := DefaultRegistry()
	if reg != reg2 {
		t.Error("DefaultRegistry should return the same instance")
	}
}

func TestMustRegisterTool_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil tool")
		}
	}()

	registry := NewToolRegistry()
	registry.MustRegisterTool(nil)
}

func TestRegisterDefaultFilesystemTools(t *testing.T) {
	// Reset default registry
	defaultRegistry = NewToolRegistry()

	err := RegisterDefaultFilesystemTools()
	if err != nil {
		t.Fatalf("failed to register default filesystem tools: %v", err)
	}

	if defaultRegistry.ToolCount() != 5 {
		t.Errorf("expected 5 tools, got %d", defaultRegistry.ToolCount())
	}
}

func TestRegisterDefaultCommandTools(t *testing.T) {
	// Reset default registry
	defaultRegistry = NewToolRegistry()

	err := RegisterDefaultCommandTools()
	if err != nil {
		t.Fatalf("failed to register default command tools: %v", err)
	}

	if defaultRegistry.ToolCount() != 2 {
		t.Errorf("expected 2 tools, got %d", defaultRegistry.ToolCount())
	}
}

func TestToolParameter_Default(t *testing.T) {
	param := &ToolParameter{
		Type:    "string",
		Default: "default_value",
	}

	m := param.ToMap()
	if m["default"] != "default_value" {
		t.Error("default value mismatch")
	}
}

func TestToolParameter_EmptyEnum(t *testing.T) {
	param := &ToolParameter{
		Type: "string",
		Enum: []string{},
	}

	m := param.ToMap()
	if _, ok := m["enum"]; ok {
		t.Error("empty enum should not be in map")
	}
}

func TestCommandTimeout_ContextCancellation(t *testing.T) {
	tool := NewExecuteCommandTool()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "ping -n 10 127.0.0.1"
	} else {
		cmd = "sleep 10"
	}

	result, _ := tool.Execute(ctx, map[string]interface{}{
		"command": cmd,
		"timeout": 10, // 10 seconds, but context will be cancelled before
	})

	data := result.Data.(*CommandResult)
	if data.ExitCode == 0 {
		t.Error("expected non-zero exit code for cancelled context")
	}
}

func TestReadFileTool_DirectoryPath(t *testing.T) {
	tool := NewReadFileTool()

	tmpDir := t.TempDir()

	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"path": tmpDir,
	})

	if result.Success {
		t.Error("expected error for directory path")
	}
}

func TestWriteFileTool_OverwriteExisting(t *testing.T) {
	tool := NewWriteFileTool()

	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("original content"), 0644)

	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"path":    tmpFile,
		"content": "new content",
	})

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}

	content, _ := os.ReadFile(tmpFile)
	if string(content) != "new content" {
		t.Error("file should be overwritten")
	}
}

func TestCreateDirectoryTool_ExistingFile(t *testing.T) {
	tool := NewCreateDirectoryTool()

	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("content"), 0644)

	result, _ := tool.Execute(context.Background(), map[string]interface{}{
		"path": tmpFile,
	})

	if result.Success {
		t.Error("expected error when path is an existing file")
	}
}

func TestExecuteCommandTool_MissingCommand(t *testing.T) {
	tool := NewExecuteCommandTool()

	result, _ := tool.Execute(context.Background(), map[string]interface{}{})

	if result.Success {
		t.Error("expected error for missing command")
	}
}

func TestShellExecuteTool_MissingCommand(t *testing.T) {
	tool := NewShellExecuteTool()

	result, _ := tool.Execute(context.Background(), map[string]interface{}{})

	if result.Success {
		t.Error("expected error for missing command")
	}
}

func TestToolRegistry_ExecuteTool_WithStructParams(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterTool(NewReadFileTool())

	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	os.WriteFile(tmpFile, []byte("content"), 0644)

	// Create a struct that can be marshaled to JSON
	params := struct {
		Path string `json:"path"`
	}{
		Path: tmpFile,
	}

	result, err := registry.ExecuteTool(context.Background(), "read_file", params)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

func TestToolRegistry_ExecuteTool_WithInvalidJSONString(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterTool(NewReadFileTool())

	_, err := registry.ExecuteTool(context.Background(), "read_file", "invalid json {")
	if err == nil {
		t.Error("expected error for invalid JSON string")
	}
}

func TestToolRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewToolRegistry()

	// Concurrent registration
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			tool := NewReadFileTool()
			// This will fail for duplicates, but shouldn't panic
			registry.RegisterTool(tool)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			registry.GetAllTools()
			registry.GetAllToolSchemas()
			registry.ToolCount()
			registry.ListToolNames()
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestToolRegistry_ExecuteTool_WithComplexJSONString(t *testing.T) {
	registry := NewToolRegistry()
	registry.RegisterTool(NewWriteFileTool())

	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	// Use forward slashes for JSON compatibility
	jsonPath := strings.ReplaceAll(tmpFile, "\\", "/")

	jsonParams := `{"path":"` + jsonPath + `","content":"test content with quotes and newlines"}`

	result, err := registry.ExecuteTool(context.Background(), "write_file", jsonParams)
	if err != nil {
		t.Fatalf("execute failed: %v", err)
	}

	if !result.Success {
		t.Errorf("expected success, got error: %s", result.Error)
	}
}

func TestToolParameter_ComplexNested(t *testing.T) {
	param := &ToolParameter{
		Type: "object",
		Properties: map[string]*ToolParameter{
			"user": {
				Type: "object",
				Properties: map[string]*ToolParameter{
					"name": {
						Type:        "string",
						Description: "User name",
					},
					"age": {
						Type:    "integer",
						Minimum: func() *float64 { v := float64(0); return &v }(),
						Maximum: func() *float64 { v := float64(150); return &v }(),
					},
					"tags": {
						Type: "array",
						Items: &ToolParameter{
							Type: "string",
						},
					},
				},
			},
		},
	}

	m := param.ToMap()

	props, ok := m["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found")
	}

	userProps, ok := props["user"].(map[string]interface{})
	if !ok {
		t.Fatal("user property not found")
	}

	userPropsMap, ok := userProps["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("user properties not found")
	}

	if _, ok := userPropsMap["name"]; !ok {
		t.Error("name property not found")
	}

	if _, ok := userPropsMap["age"]; !ok {
		t.Error("age property not found")
	}

	if _, ok := userPropsMap["tags"]; !ok {
		t.Error("tags property not found")
	}
}

func TestBuildSchema_WithMultipleProperties(t *testing.T) {
	properties := map[string]*ToolParameter{
		"path": {
			Type:        "string",
			Description: "File path",
		},
		"content": {
			Type:        "string",
			Description: "File content",
		},
		"overwrite": {
			Type:        "boolean",
			Description: "Overwrite existing file",
			Default:     false,
		},
	}

	schema := BuildSchema(properties, []string{"path", "content"})

	props, ok := schema["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties not found")
	}

	if len(props) != 3 {
		t.Errorf("expected 3 properties, got %d", len(props))
	}

	required, ok := schema["required"].([]string)
	if !ok {
		t.Fatal("required not found")
	}

	if len(required) != 2 {
		t.Errorf("expected 2 required fields, got %d", len(required))
	}
}

func TestToolRegistry_ConcurrentReadWrite(t *testing.T) {
	registry := NewToolRegistry()

	// Start writers
	writerDone := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				tool := NewReadFileTool()
				registry.RegisterTool(tool)
			}
			writerDone <- true
		}(i)
	}

	// Start readers
	readerDone := make(chan bool)
	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				registry.GetAllTools()
				registry.GetAllToolSchemas()
				registry.ToolCount()
			}
			readerDone <- true
		}()
	}

	// Wait for all to complete
	for i := 0; i < 5; i++ {
		<-writerDone
	}
	for i := 0; i < 5; i++ {
		<-readerDone
	}
}

func TestToolResult_ToJSON_WithComplexData(t *testing.T) {
	result := NewSuccessResult(map[string]interface{}{
		"string":   "value",
		"int":      123,
		"float":    45.67,
		"bool":     true,
		"slice":    []string{"a", "b", "c"},
		"map":      map[string]int{"x": 1, "y": 2},
		"nil":      nil,
		"nested":   map[string]interface{}{"a": 1, "b": 2},
		"empty":    []string{},
		"emptyMap": map[string]string{},
	})

	jsonStr := result.ToJSON()

	if !contains(jsonStr, "string") {
		t.Error("JSON should contain 'string'")
	}

	if !contains(jsonStr, "nested") {
		t.Error("JSON should contain 'nested'")
	}
}

func TestToolSchema_ToJSON(t *testing.T) {
	schema := ToolSchema{
		Type: "function",
		Function: FunctionSchema{
			Name:        "test",
			Description: "test function",
			Parameters: map[string]interface{}{
				"type": "object",
			},
		},
	}

	// ToolSchema can be marshaled to JSON
	bytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(bytes)

	if !contains(jsonStr, "function") {
		t.Error("JSON should contain 'function'")
	}

	if !contains(jsonStr, "test") {
		t.Error("JSON should contain 'test'")
	}
}

func TestCommandResult_ToJSON(t *testing.T) {
	result := &CommandResult{
		Command:    "echo test",
		Stdout:     "output",
		Stderr:     "error",
		ExitCode:   0,
		Success:    true,
		Duration:   "1ms",
		TimedOut:   false,
		WorkingDir: "/tmp",
	}

	bytes, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(bytes)

	if !contains(jsonStr, "echo test") {
		t.Error("JSON should contain command")
	}

	if !contains(jsonStr, "output") {
		t.Error("JSON should contain stdout")
	}
}

func TestFileInfo_ToJSON(t *testing.T) {
	info := FileInfo{
		Name:    "test.txt",
		Path:    "/tmp/test.txt",
		IsDir:   false,
		Size:    100,
		Mode:    "-rw-r--r--",
		ModTime: "2024-01-01 00:00:00",
	}

	bytes, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal: %v", err)
	}

	jsonStr := string(bytes)

	if !contains(jsonStr, "test.txt") {
		t.Error("JSON should contain name")
	}

	if !contains(jsonStr, "/tmp/test.txt") {
		t.Error("JSON should contain path")
	}
}
