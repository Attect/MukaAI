package context

import (
	"github.com/Attect/MukaAI/internal/model"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// === 测试辅助函数 ===

// createTestProject 创建测试用的项目目录结构
func createTestProject(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// 创建目录结构
	dirs := []string{
		"cmd/app",
		"internal/service",
		"internal/model",
		"pkg/utils",
		".git/objects",
		"node_modules/lib",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0755); err != nil {
			t.Fatalf("创建目录失败: %v", err)
		}
	}

	// 创建文件
	files := map[string]string{
		"cmd/app/main.go": `package main

import (
	"fmt"
	"myapp/internal/service"
)

func main() {
	svc := service.NewUserService()
	fmt.Println(svc.Greet("World"))
}

func helper() string {
	return "helper"
}
`,
		"internal/service/user.go": `package service

// UserService 用户服务
type UserService struct {
	users map[string]string
}

// NewUserService 创建用户服务
func NewUserService() *UserService {
	return &UserService{
		users: make(map[string]string),
	}
}

// Greet 问候用户
func (s *UserService) Greet(name string) string {
	return "Hello, " + name
}

// AddUser 添加用户
func (s *UserService) AddUser(id, name string) {
	s.users[id] = name
}
`,
		"internal/model/types.go": `package model

// User 用户模型
type User struct {
	ID   string
	Name string
	Age  int
}

// UserRequest 用户请求
type UserRequest struct {
	Name string
	Age  int
}
`,
		"pkg/utils/helper.go": `package utils

import "strings"

// TrimSpace 去除空格
func TrimSpace(s string) string {
	return strings.TrimSpace(s)
}

var Version = "1.0.0"
`,
		"README.md": `# My App
A sample application for testing context indexing.
`,
		"config.yaml": `server:
  port: 8080
database:
  host: localhost
`,
		".git/HEAD":           `ref: refs/heads/main`,
		"node_modules/lib.js": `module.exports = {}`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("写入文件失败: %v", err)
		}
	}

	return tmpDir
}

// === FileEntry和类型测试 ===

func TestDetectLanguage(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"App.java", "java"},
		{"index.ts", "typescript"},
		{"app.py", "python"},
		{"main.rs", "rust"},
		{"styles.css", "css"},
		{"config.yaml", "yaml"},
		{"data.json", "json"},
		{"unknown.xyz", "unknown"},
	}

	for _, tt := range tests {
		got := detectLanguage(tt.path)
		if got != tt.want {
			t.Errorf("detectLanguage(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestIsBinaryFile(t *testing.T) {
	binaryFiles := []string{"app.exe", "lib.dll", "image.png", "data.zip", "font.ttf"}
	for _, f := range binaryFiles {
		if !isBinaryFile(f) {
			t.Errorf("isBinaryFile(%q) = false, want true", f)
		}
	}

	textFiles := []string{"main.go", "readme.md", "config.yaml"}
	for _, f := range textFiles {
		if isBinaryFile(f) {
			t.Errorf("isBinaryFile(%q) = true, want false", f)
		}
	}
}

func TestShouldIgnoreDir(t *testing.T) {
	ignoreDirs := []string{".git", "node_modules", "vendor", "__pycache__", "dist", "build", "ai"}
	for _, d := range ignoreDirs {
		if !shouldIgnoreDir(d) {
			t.Errorf("shouldIgnoreDir(%q) = false, want true", d)
		}
	}

	keepDirs := []string{"src", "internal", "pkg", "cmd"}
	for _, d := range keepDirs {
		if shouldIgnoreDir(d) {
			t.Errorf("shouldIgnoreDir(%q) = true, want false", d)
		}
	}
}

// === 文件树扫描测试 ===

func TestFileTreeScanner_Scan(t *testing.T) {
	tmpDir := createTestProject(t)
	scanner := NewFileTreeScanner(tmpDir)
	entries, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// 验证文件数量（应排除.git和node_modules下的文件）
	if len(entries) == 0 {
		t.Fatal("Scan() returned no entries")
	}

	// 验证没有.git目录下的文件
	for _, e := range entries {
		if strings.HasPrefix(e.Path, ".git/") {
			t.Errorf("Scan() should ignore .git directory, got: %s", e.Path)
		}
		if strings.HasPrefix(e.Path, "node_modules/") {
			t.Errorf("Scan() should ignore node_modules directory, got: %s", e.Path)
		}
	}

	// 验证关键文件被索引
	foundFiles := make(map[string]bool)
	for _, e := range entries {
		foundFiles[e.Path] = true
	}

	expectedFiles := []string{
		"cmd/app/main.go",
		"internal/service/user.go",
		"internal/model/types.go",
		"pkg/utils/helper.go",
	}
	for _, f := range expectedFiles {
		if !foundFiles[f] {
			t.Errorf("Expected file %s to be indexed", f)
		}
	}

	// 验证语言检测
	for _, e := range entries {
		if strings.HasSuffix(e.Path, ".go") && e.Language != "go" {
			t.Errorf("Expected language 'go' for %s, got '%s'", e.Path, e.Language)
		}
	}
}

func TestFileTreeScanner_EmptyDir(t *testing.T) {
	tmpDir := t.TempDir()
	scanner := NewFileTreeScanner(tmpDir)
	entries, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("Empty dir should have 0 entries, got %d", len(entries))
	}
}

func TestReadFileContent(t *testing.T) {
	tmpDir := t.TempDir()

	// 创建正常文件
	content := strings.Repeat("line\n", 100)
	path := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(path, []byte(content), 0644)

	full, truncated, err := ReadFileContent(path)
	if err != nil {
		t.Fatalf("ReadFileContent() error = %v", err)
	}

	if full != truncated {
		t.Error("For files within limit, full and truncated should be equal")
	}

	// 创建超长文件
	longContent := strings.Repeat("line\n", 600)
	os.WriteFile(path, []byte(longContent), 0644)

	_, truncated, err = ReadFileContent(path)
	if err != nil {
		t.Fatalf("ReadFileContent() error = %v", err)
	}

	if !strings.Contains(truncated, "truncated") {
		t.Error("Long file should contain truncation marker")
	}
}

// === 关键词提取测试 ===

func TestExtractKeywordsFromFilename(t *testing.T) {
	tests := []struct {
		filename string
		want     []string
	}{
		{"user_service.go", []string{"user", "service"}},
		{"UserService.java", []string{"user", "service"}},
		{"my-component.tsx", []string{"component"}}, // "my" is a stopword
		{"main.go", nil}, // "main" is too short (len <= 1), but actually len=4 so it should be included
	}

	for _, tt := range tests {
		got := ExtractKeywordsFromFilename(tt.filename)
		if tt.filename == "main.go" {
			// "main" has length 4, should be included
			if len(got) == 0 {
				t.Errorf("ExtractKeywordsFromFilename(%q) returned empty, expected 'main'", tt.filename)
			}
			continue
		}
		if len(got) != len(tt.want) {
			t.Errorf("ExtractKeywordsFromFilename(%q) = %v, want %v", tt.filename, got, tt.want)
			continue
		}
		gotSet := make(map[string]bool)
		for _, g := range got {
			gotSet[g] = true
		}
		for _, w := range tt.want {
			if !gotSet[w] {
				t.Errorf("ExtractKeywordsFromFilename(%q) missing keyword %q", tt.filename, w)
			}
		}
	}
}

func TestExtractKeywordsFromContent(t *testing.T) {
	content := `
package service

// UserService 用户服务
type UserService struct {
	users map[string]string
}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) Greet(name string) string {
	return "Hello"
}

import "fmt"
`
	keywords := ExtractKeywordsFromContent(content, "go")

	// 验证提取到的关键词
	keywordSet := make(map[string]bool)
	for _, kw := range keywords {
		keywordSet[kw] = true
	}

	// 应该包含函数名
	expectedKeywords := []string{"userservice", "newuserservice", "greet"}
	for _, kw := range expectedKeywords {
		if !keywordSet[kw] {
			t.Errorf("Expected keyword %q not found in %v", kw, keywords)
		}
	}
}

func TestExtractKeywordsFromTask(t *testing.T) {
	tests := []struct {
		task string
		want []string // 至少包含这些关键词
	}{
		{
			task: "修改 user.go 中的 UserService",
			want: []string{"user", "userservice"},
		},
		{
			task: "Fix the authentication module",
			want: []string{"fix", "authentication", "module"},
		},
		{
			task: "添加新的API端点 /api/v1/users",
			want: []string{"api"},
		},
	}

	for _, tt := range tests {
		got := ExtractKeywordsFromTask(tt.task)
		gotSet := make(map[string]bool)
		for _, g := range got {
			gotSet[g] = true
		}
		for _, w := range tt.want {
			if !gotSet[w] {
				t.Errorf("ExtractKeywordsFromTask(%q) missing keyword %q, got %v", tt.task, w, got)
			}
		}
	}
}

// === 符号提取测试 ===

func TestExtractSymbols_Go(t *testing.T) {
	content := `
package main

type UserService struct {
	users map[string]string
}

type Handler interface {
	Handle() error
}

func NewUserService() *UserService {
	return &UserService{}
}

func (s *UserService) Greet(name string) string {
	return "Hello"
}
`
	symbols := ExtractSymbols(content, "go")

	if len(symbols) == 0 {
		t.Fatal("ExtractSymbols() returned no symbols")
	}

	// 验证提取到的符号
	symbolMap := make(map[string]string) // name -> kind
	for _, sym := range symbols {
		symbolMap[sym.Name] = sym.Kind
	}

	expectedSymbols := map[string]string{
		"NewUserService": "function",
		"Greet":          "function",
	}

	for name, kind := range expectedSymbols {
		if gotKind, ok := symbolMap[name]; !ok {
			t.Errorf("Expected symbol %q not found", name)
		} else if gotKind != kind {
			t.Errorf("Symbol %q: kind = %q, want %q", name, gotKind, kind)
		}
	}
}

func TestExtractSymbols_Python(t *testing.T) {
	content := `
class UserService:
    def __init__(self):
        self.users = {}

    def greet(self, name):
        return "Hello, " + name

def helper():
    pass
`
	symbols := ExtractSymbols(content, "python")

	symbolMap := make(map[string]string)
	for _, sym := range symbols {
		symbolMap[sym.Name] = sym.Kind
	}

	expected := map[string]string{
		"UserService": "class",
		"greet":       "function",
		"helper":      "function",
	}

	for name, kind := range expected {
		if gotKind, ok := symbolMap[name]; !ok {
			t.Errorf("Expected symbol %q not found", name)
		} else if gotKind != kind {
			t.Errorf("Symbol %q: kind = %q, want %q", name, gotKind, kind)
		}
	}
}

// === 索引器测试 ===

func TestIndexer_Scan(t *testing.T) {
	tmpDir := createTestProject(t)
	indexer := NewIndexer(tmpDir)

	count, err := indexer.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	if count == 0 {
		t.Fatal("Scan() indexed 0 files")
	}

	if !indexer.IsReady() {
		t.Error("IsReady() should be true after Scan()")
	}

	if indexer.GetProjectName() == "" {
		t.Error("GetProjectName() should not be empty")
	}
}

func TestIndexer_Query(t *testing.T) {
	tmpDir := createTestProject(t)
	indexer := NewIndexer(tmpDir)

	_, err := indexer.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// 查询与用户服务相关的文件
	results := indexer.Query("修改 UserService 的 Greet 方法", 5)
	if len(results) == 0 {
		t.Fatal("Query() returned no results")
	}

	// user.go 应该排在最前面
	found := false
	for _, f := range results {
		if strings.Contains(f.Path, "user.go") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected user.go in results, got: %v", results)
	}
}

func TestIndexer_QueryByFileName(t *testing.T) {
	tmpDir := createTestProject(t)
	indexer := NewIndexer(tmpDir)

	_, err := indexer.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// 直接通过文件名查询
	results := indexer.Query("修改 main.go 的入口函数", 5)
	if len(results) == 0 {
		t.Fatal("Query() returned no results for main.go")
	}

	// main.go 应该在结果中
	found := false
	for _, f := range results {
		if strings.Contains(f.Path, "main.go") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected main.go in results, got: %v", results)
	}
}

func TestIndexer_QueryNotReady(t *testing.T) {
	indexer := NewIndexer(t.TempDir())
	results := indexer.Query("test", 5)
	if results != nil {
		t.Error("Query() on unready indexer should return nil")
	}
}

func TestIndexer_GetFileByPath(t *testing.T) {
	tmpDir := createTestProject(t)
	indexer := NewIndexer(tmpDir)

	_, err := indexer.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// 精确查找
	f, ok := indexer.GetFileByPath("cmd/app/main.go")
	if !ok {
		t.Fatal("GetFileByPath() should find cmd/app/main.go")
	}
	if f.Language != "go" {
		t.Errorf("Language = %q, want 'go'", f.Language)
	}

	// 不存在的文件
	_, ok = indexer.GetFileByPath("nonexistent.go")
	if ok {
		t.Error("GetFileByPath() should not find nonexistent file")
	}
}

// === 查询引擎测试 ===

func TestQueryEngine_QueryWithFileHints(t *testing.T) {
	tmpDir := createTestProject(t)
	indexer := NewIndexer(tmpDir)

	_, err := indexer.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	engine := NewQueryEngine(indexer)

	// 任务中直接提到文件路径
	results := engine.QueryWithFileHints("修改 config.yaml 的端口配置", 5)
	if len(results) == 0 {
		t.Fatal("QueryWithFileHints() returned no results")
	}

	// config.yaml 应该在结果中
	found := false
	for _, f := range results {
		if f.Path == "config.yaml" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected config.yaml in results, got: %v", results)
	}
}

func TestEstimateTokenCount(t *testing.T) {
	// 空文本
	if EstimateTokenCount("") != 0 {
		t.Error("Empty string should have 0 tokens")
	}

	// 英文文本（4字符约1token）
	text := "Hello World Test"
	tokens := EstimateTokenCount(text)
	if tokens <= 0 {
		t.Errorf("EstimateTokenCount(%q) = %d, want > 0", text, tokens)
	}

	// 中文文本
	text = "你好世界测试"
	tokens = EstimateTokenCount(text)
	if tokens <= 0 {
		t.Errorf("EstimateTokenCount(%q) = %d, want > 0", text, tokens)
	}
}

// === 注入器测试 ===

func TestInjector_InjectContext(t *testing.T) {
	tmpDir := createTestProject(t)
	indexer := NewIndexer(tmpDir)

	_, err := indexer.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	injector := NewInjector(indexer, 10000)

	// 构造输入消息
	messages := []model.Message{
		{Role: "system", Content: "You are an assistant"},
		{Role: "user", Content: "修改 UserService 的 Greet 方法"},
	}

	// 注入上下文
	result := injector.InjectContext("修改 UserService 的 Greet 方法", messages)

	// 应该多出一条消息（上下文注入）
	if len(result) != len(messages)+1 {
		t.Errorf("InjectContext() result length = %d, want %d", len(result), len(messages)+1)
	}

	// 第二条消息应该是用户消息（上下文注入使用user角色，兼容llama.cpp的Jinja模板）
	if result[1].Role != "user" {
		t.Errorf("Injected message role = %q, want 'user'", result[1].Role)
	}

	// 上下文内容应包含项目信息
	content := result[1].Content
	if !strings.Contains(content, "Project Context") {
		t.Error("Injected context should contain 'Project Context'")
	}
	if !strings.Contains(content, "Related Files") {
		t.Error("Injected context should contain 'Related Files'")
	}
}

func TestInjector_InjectContext_NotReady(t *testing.T) {
	indexer := NewIndexer(t.TempDir())
	injector := NewInjector(indexer, 10000)

	messages := []model.Message{
		{Role: "system", Content: "test"},
	}

	result := injector.InjectContext("test task", messages)
	if len(result) != len(messages) {
		t.Error("InjectContext() on unready indexer should return original messages")
	}
}

func TestInjector_BuildContextString(t *testing.T) {
	tmpDir := createTestProject(t)
	indexer := NewIndexer(tmpDir)

	_, err := indexer.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	injector := NewInjector(indexer, 10000)
	contextStr, files := injector.BuildContextString("修改 UserService", 5)

	if len(files) == 0 {
		t.Fatal("BuildContextString() returned no files")
	}
	if contextStr == "" {
		t.Fatal("BuildContextString() returned empty context")
	}
	if !strings.Contains(contextStr, "Project Context") {
		t.Error("Context should contain header")
	}
}

func TestInjector_TokenBudget(t *testing.T) {
	tmpDir := createTestProject(t)
	indexer := NewIndexer(tmpDir)

	_, err := indexer.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// 设置极小的Token预算
	injector := NewInjector(indexer, 10) // 只有10个token

	messages := []model.Message{
		{Role: "system", Content: "test"},
	}

	result := injector.InjectContext("修改 UserService", messages)

	// 即使预算很小，也应该成功注入（截断后）
	// 或者返回原始消息（如果完全无法注入）
	if len(result) < len(messages) {
		t.Error("Result should have at least original messages")
	}
}

func TestNewInjectorFromContextSize(t *testing.T) {
	indexer := NewIndexer(t.TempDir())
	injector := NewInjectorFromContextSize(indexer, 200000)

	expected := 200000 / 5 // 20%
	if injector.GetMaxTokens() != expected {
		t.Errorf("GetMaxTokens() = %d, want %d", injector.GetMaxTokens(), expected)
	}
}

// === 异步扫描测试 ===

func TestIndexer_ScanAsync(t *testing.T) {
	tmpDir := createTestProject(t)
	indexer := NewIndexer(tmpDir)

	ch := indexer.ScanAsync()
	result := <-ch

	if result.Err != nil {
		t.Fatalf("ScanAsync() error = %v", result.Err)
	}
	if result.FileCount == 0 {
		t.Fatal("ScanAsync() indexed 0 files")
	}
	if !indexer.IsReady() {
		t.Error("Indexer should be ready after ScanAsync()")
	}
}

// === 排序因子测试 ===

func TestComputePathRelevanceScore(t *testing.T) {
	tests := []struct {
		path     string
		taskDesc string
		wantHigh bool // 期望高分
	}{
		{"internal/service/user.go", "修改 user.go 文件", true},
		{"internal/service/user.go", "修改 UserService", true},
		{"cmd/app/main.go", "修改 main.go", true},
		{"pkg/utils/helper.go", "修改 config.yaml", false},
	}

	for _, tt := range tests {
		score := computePathRelevanceScore(tt.path, tt.taskDesc)
		if tt.wantHigh && score <= 0 {
			t.Errorf("computePathRelevanceScore(%q, %q) = %f, want > 0", tt.path, tt.taskDesc, score)
		}
		if !tt.wantHigh && score > 0.3 {
			t.Errorf("computePathRelevanceScore(%q, %q) = %f, want low score", tt.path, tt.taskDesc, score)
		}
	}
}

// === 综合测试 ===

func TestEndToEnd(t *testing.T) {
	tmpDir := createTestProject(t)

	// 1. 创建索引器并扫描
	indexer := NewIndexer(tmpDir)
	count, err := indexer.Scan()
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}
	t.Logf("Indexed %d files", count)

	// 2. 查询相关文件
	results := indexer.Query("修改 UserService 的 Greet 方法返回个性化问候", 5)
	t.Logf("Query results:")
	for i, f := range results {
		t.Logf("  %d. %s (%s) - %d keywords, %d symbols",
			i+1, f.Path, f.Language, len(f.Keywords), len(f.Symbols))
	}

	// 3. 验证user.go在最前面
	if len(results) > 0 {
		if !strings.Contains(results[0].Path, "user.go") {
			t.Errorf("Most relevant file should be user.go, got %s", results[0].Path)
		}
	}

	// 4. 注入上下文
	injector := NewInjector(indexer, 10000)
	contextStr, files := injector.BuildContextString("修改 Greet 方法", 3)
	if contextStr == "" {
		t.Fatal("BuildContextString() returned empty context")
	}

	t.Logf("Context length: %d chars, %d estimated tokens",
		len(contextStr), EstimateTokenCount(contextStr))

	// 5. 验证上下文格式
	if !strings.Contains(contextStr, "Project Context") {
		t.Error("Context should contain header")
	}
	if !strings.Contains(contextStr, "End Project Context") {
		t.Error("Context should contain footer")
	}

	_ = files // used above
}

// === 基准测试 ===

func BenchmarkExtractKeywordsFromContent(b *testing.B) {
	content := strings.Repeat(`
func ProcessRequest(req *Request) (*Response, error) {
	// Process the request
	if req == nil {
		return nil, ErrNilRequest
	}
	result := handler.Handle(req.Data)
	return &Response{Data: result}, nil
}

type Request struct {
	Data string
}
`, 10)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ExtractKeywordsFromContent(content, "go")
	}
}

func BenchmarkIndexerQuery(b *testing.B) {
	tmpDir := b.TempDir()

	// 创建大量文件
	for i := 0; i < 100; i++ {
		content := "package pkg" + strings.Repeat("\n// Comment\nfunc Func"+string(rune('A'+i%26))+"() {}\n", 10)
		os.WriteFile(filepath.Join(tmpDir, "file"+string(rune('A'+i%26))+".go"), []byte(content), 0644)
	}

	indexer := NewIndexer(tmpDir)
	indexer.Scan()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		indexer.Query("修改 FuncA 的实现", 10)
	}
}

// === 关键词合并测试 ===

func TestMergeKeywords(t *testing.T) {
	existing := []string{"user", "service"}
	additional := []string{"service", "handler", "user"}

	result := mergeKeywords(existing, additional)

	// 验证去重
	seen := make(map[string]bool)
	for _, kw := range result {
		if seen[kw] {
			t.Errorf("Duplicate keyword: %s", kw)
		}
		seen[kw] = true
	}

	if len(result) != 3 {
		t.Errorf("mergeKeywords() = %v, want 3 unique keywords", result)
	}
}

// === 排序验证测试 ===

func TestQueryResultOrdering(t *testing.T) {
	tmpDir := createTestProject(t)
	indexer := NewIndexer(tmpDir)

	_, err := indexer.Scan()
	if err != nil {
		t.Fatalf("Scan() error = %v", err)
	}

	// 查询应该返回按相关性排序的结果
	results := indexer.Query("UserService Greet method", 10)

	// 验证结果是按得分排序的（user.go 应该比其他文件更相关）
	if len(results) >= 2 {
		// 至少验证第一个结果是合理的
		t.Logf("Top result: %s", results[0].Path)
		if !strings.Contains(results[0].Path, "user") &&
			!strings.Contains(results[0].Path, "service") {
			t.Errorf("Top result %s doesn't seem relevant to UserService", results[0].Path)
		}
	}
}

// === 语言特定符号提取测试 ===

func TestExtractSymbols_Java(t *testing.T) {
	content := `
public class UserService {
    private String name;
    
    public UserService(String name) {
        this.name = name;
    }
    
    public String greet() {
        return "Hello";
    }
}

interface UserListener {
    void onUserCreated(User user);
}
`
	symbols := ExtractSymbols(content, "java")
	if len(symbols) == 0 {
		t.Fatal("ExtractSymbols() for Java returned no symbols")
	}
}

func TestExtractSymbols_TypeScript(t *testing.T) {
	content := `
export class UserService {
  private name: string;

  constructor(name: string) {
    this.name = name;
  }

  greet(): string {
    return "Hello";
  }
}

export interface User {
  id: string;
  name: string;
}

export function createUser(name: string): User {
  return { id: "1", name };
}
`
	symbols := ExtractSymbols(content, "typescript")
	if len(symbols) == 0 {
		t.Fatal("ExtractSymbols() for TypeScript returned no symbols")
	}
}

// === ensure sort import is used ===
var _ = sort.Strings
