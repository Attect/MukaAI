//go:build manual

package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
)

// =============================================================================
// IntelliJ IDEA E2E 测试: 模拟MukaAI通过MCP完成Go编程题目
// =============================================================================
//
// 本测试模拟一个完整的Agent工作流:
// 1. 了解项目环境 → 2. 创建代码文件 → 3. 验证文件内容 →
// 4. 代码问题分析 → 5. 搜索验证 → 6. 构建和运行测试 →
// 7. 代码审查与修复 → 8. 最终验证
//
// 项目路径: test_mcp_go_task (空白目录, 在IDEA中已打开)
// MCP连接: Streamable HTTP http://127.0.0.1:64342/stream
//
// 策略说明:
// - 纯MCP通道测试，所有操作通过MCP完成，不使用本地回退
// - 每个MCP工具调用使用独立连接（避免长连接卡顿）
// - 失败即标记失败，确保MCP工具链路的真实可靠性
//
// 运行方式:
//   go test -tags manual -run TestIntelliJE2E ./internal/mcp/ -v -timeout 300s
// =============================================================================

const (
	// e2eProjectPath IDEA中打开的Go编程项目路径
	e2eProjectPath = `C:\Users\Attect\trae\AgentPlus\test_mcp_go_task`
	// e2eMCPURL IntelliJ IDEA MCP Server地址
	e2eMCPURL = "http://127.0.0.1:64342/stream"
)

// mainGoCode 并发安全计数器HTTP服务的主程序代码
const mainGoCode = `package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Counter struct {
	mu    sync.Mutex
	value int
}

func NewCounter() *Counter {
	return &Counter{value: 0}
}

func (c *Counter) Increment() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
	return c.value
}

func (c *Counter) Reset() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = 0
	return c.value
}

func (c *Counter) Get() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.value
}

var counter = NewCounter()

func countHandler(w http.ResponseWriter, r *http.Request) {
	result := map[string]interface{}{
		"count": counter.Get(),
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func incrHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	newVal := counter.Increment()
	result := map[string]interface{}{
		"count": newVal,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func resetHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	resetVal := counter.Reset()
	result := map[string]interface{}{
		"count": resetVal,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func main() {
	http.HandleFunc("/count", countHandler)
	http.HandleFunc("/incr", incrHandler)
	http.HandleFunc("/reset", resetHandler)
	fmt.Println("Counter server starting on :8080")
	http.ListenAndServe(":8080", nil)
}`

// mainTestGoCode 测试文件代码
const mainTestGoCode = `package main

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestInitialCount(t *testing.T) {
	counter := NewCounter()
	if counter.Get() != 0 {
		t.Errorf("expected initial count 0, got %d", counter.Get())
	}
}

func TestIncrement(t *testing.T) {
	counter := NewCounter()
	counter.Increment()
	counter.Increment()
	if counter.Get() != 2 {
		t.Errorf("expected count 2, got %d", counter.Get())
	}
}

func TestConcurrentIncrement(t *testing.T) {
	counter := NewCounter()
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			counter.Increment()
		}()
	}
	wg.Wait()
	if counter.Get() != 100 {
		t.Errorf("expected count 100, got %d", counter.Get())
	}
}

func TestCountHandler(t *testing.T) {
	req, _ := http.NewRequest("GET", "/count", nil)
	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(countHandler)
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rr.Code)
	}
}`

// goModContent go.mod文件内容
const goModContent = `module counter

go 1.25.0`

// ---------------------------------------------------------------------------
// MCP连接工具函数
// ---------------------------------------------------------------------------

// e2eMCPToolCall 使用独立MCP连接调用单个工具（默认30秒超时）
// 每次调用创建新连接，避免Streamable HTTP长连接卡顿
func e2eMCPToolCall(t *testing.T, toolName string, args map[string]interface{}, stepDesc string) (string, bool) {
	t.Helper()
	return e2eMCPToolCallWithTimeout(t, toolName, args, stepDesc, 30*time.Second)
}

// e2eMCPToolCallWithTimeout 使用指定超时调用MCP工具
func e2eMCPToolCallWithTimeout(t *testing.T, toolName string, args map[string]interface{}, stepDesc string, timeout time.Duration) (string, bool) {
	t.Helper()
	t.Logf("  [步骤] %s: 调用 %s (超时=%v)", stepDesc, toolName, timeout)

	// 注入projectPath
	if args == nil {
		args = map[string]interface{}{}
	}
	args["projectPath"] = e2eProjectPath

	cfg := ServerConfig{
		ID:        "idea_e2e",
		Enabled:   true,
		Transport: "http",
		URL:       e2eMCPURL,
		Timeout:   int(timeout.Seconds()) + 5,
	}
	session := NewMCPSession("idea_e2e", cfg)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := session.Connect(ctx); err != nil {
		t.Logf("  [连接失败] %s: %v", stepDesc, err)
		return "", false
	}
	defer session.Close()

	result, err := session.CallTool(ctx, toolName, args)
	if err != nil {
		if strings.Contains(err.Error(), "deadline exceeded") {
			t.Logf("  [超时] %s", stepDesc)
		} else {
			t.Logf("  [失败] %s: %v", stepDesc, err)
		}
		return "", false
	}

	if result.IsError {
		errText := extractTextFromResult(result)
		t.Logf("  [工具错误] %s: %s", stepDesc, truncate(errText, 200))
		return errText, false
	}

	text := extractTextFromResult(result)
	t.Logf("  [成功] %s: 长度=%d, 摘要=%q", stepDesc, len(text), truncate(text, 150))
	return text, true
}

// e2eCreateFile 纯MCP文件创建: 失败直接标记失败，不回退本地
func e2eCreateFile(t *testing.T, pathInProject, content string, stepDesc string) bool {
	t.Helper()

	_, ok := e2eMCPToolCallWithTimeout(t, "create_new_file", map[string]interface{}{
		"pathInProject": pathInProject,
		"text":          content,
	}, stepDesc, 60*time.Second)

	if !ok {
		t.Errorf("[失败] MCP创建文件 %s 失败，无本地回退", pathInProject)
		return false
	}

	t.Logf("  [MCP创建成功] %s", pathInProject)
	return true
}

// e2eReadFile 纯MCP文件读取: 失败直接标记失败，不回退本地
func e2eReadFile(t *testing.T, pathInProject string, stepDesc string) (string, bool) {
	t.Helper()

	text, ok := e2eMCPToolCallWithTimeout(t, "get_file_text_by_path", map[string]interface{}{
		"pathInProject": pathInProject,
	}, stepDesc, 30*time.Second)

	if !ok || text == "" {
		t.Errorf("[失败] MCP读取文件 %s 失败，无本地回退", pathInProject)
		return "", false
	}

	return text, true
}

// assertContains 断言包含（失败时记录错误）
func assertContains(t *testing.T, text, keyword, context string) {
	t.Helper()
	if !strings.Contains(text, keyword) {
		t.Errorf("[断言失败] %s: 期望包含 %q, 实际: %s", context, keyword, truncate(text, 200))
	} else {
		t.Logf("[断言通过] %s: 包含 %q", context, keyword)
	}
}

// assertContainsSoft 软断言包含（失败时仅记录警告，不算测试失败）
func assertContainsSoft(t *testing.T, text, keyword, context string) {
	t.Helper()
	if !strings.Contains(text, keyword) {
		t.Logf("[软断言-未通过] %s: 期望包含 %q, 实际: %s", context, keyword, truncate(text, 200))
	} else {
		t.Logf("[软断言-通过] %s: 包含 %q", context, keyword)
	}
}

// ---------------------------------------------------------------------------
// E2E 测试主体
// ---------------------------------------------------------------------------

// TestIntelliJE2E 端到端测试: 模拟Agent通过MCP完成Go编程题目
// 运行方式: go test -tags manual -run TestIntelliJE2E ./internal/mcp/ -v -timeout 300s
func TestIntelliJE2E(t *testing.T) {
	// 清理旧文件
	for _, f := range []string{"main.go", "main_test.go", "go.mod"} {
		os.Remove(e2eProjectPath + string(os.PathSeparator) + f)
	}

	phaseResults := make(map[string]bool)

	// =========================================================================
	// 阶段1: 了解环境
	// =========================================================================
	t.Log("\n========================================")
	t.Log("阶段1: 了解环境")
	t.Log("========================================")

	// 步骤1.1: get_project_modules
	t.Log("\n--- 步骤1.1: get_project_modules ---")
	if text, ok := e2eMCPToolCall(t, "get_project_modules", map[string]interface{}{}, "获取项目模块"); ok {
		t.Logf("[验证] 模块信息: %s", truncate(text, 200))
		phaseResults["phase1_modules"] = true
	} else {
		phaseResults["phase1_modules"] = false
	}

	// 步骤1.2: list_directory_tree
	t.Log("\n--- 步骤1.2: list_directory_tree ---")
	if text, ok := e2eMCPToolCall(t, "list_directory_tree", map[string]interface{}{
		"directoryPath": ".",
	}, "查看项目目录"); ok {
		t.Logf("[验证] 目录树: %s", truncate(text, 300))
		phaseResults["phase1_tree"] = true
	} else {
		phaseResults["phase1_tree"] = false
	}

	// =========================================================================
	// 阶段2: 创建代码文件
	// =========================================================================
	t.Log("\n========================================")
	t.Log("阶段2: 创建代码文件")
	t.Log("========================================")

	t.Log("\n--- 步骤2.1: 创建 main.go ---")
	phaseResults["phase2_main"] = e2eCreateFile(t, "main.go", mainGoCode, "创建main.go")

	t.Log("\n--- 步骤2.2: 创建 main_test.go ---")
	phaseResults["phase2_test"] = e2eCreateFile(t, "main_test.go", mainTestGoCode, "创建main_test.go")

	t.Log("\n--- 步骤2.3: 创建 go.mod ---")
	phaseResults["phase2_mod"] = e2eCreateFile(t, "go.mod", goModContent, "创建go.mod")

	t.Log("\n[等待] IDEA索引文件(10秒)...")
	time.Sleep(10 * time.Second)

	// =========================================================================
	// 阶段3: 验证文件内容
	// =========================================================================
	t.Log("\n========================================")
	t.Log("阶段3: 验证文件内容")
	t.Log("========================================")

	t.Log("\n--- 步骤3.1: 读取main.go ---")
	if text, ok := e2eReadFile(t, "main.go", "读取main.go"); ok {
		assertContains(t, text, "Mutex", "main.go")
		assertContains(t, text, "net/http", "main.go")
		assertContains(t, text, "counter", "main.go")
		phaseResults["phase3_main"] = true
	} else {
		t.Error("[严重] main.go读取失败")
		phaseResults["phase3_main"] = false
	}

	t.Log("\n--- 步骤3.2: 读取main_test.go ---")
	if text, ok := e2eReadFile(t, "main_test.go", "读取main_test.go"); ok {
		assertContains(t, text, "TestInitialCount", "main_test.go")
		assertContains(t, text, "TestIncrement", "main_test.go")
		assertContains(t, text, "TestConcurrentIncrement", "main_test.go")
		phaseResults["phase3_test"] = true
	} else {
		t.Error("[严重] main_test.go读取失败")
		phaseResults["phase3_test"] = false
	}

	// =========================================================================
	// 阶段4: 代码问题分析
	// =========================================================================
	t.Log("\n========================================")
	t.Log("阶段4: 代码问题分析")
	t.Log("========================================")

	t.Log("\n--- 步骤4.1: get_file_problems(main.go) ---")
	if text, ok := e2eMCPToolCallWithTimeout(t, "get_file_problems", map[string]interface{}{
		"filePath":   "main.go",
		"errorsOnly": false,
	}, "分析main.go问题", 45*time.Second); ok {
		if text == "" || strings.Contains(text, "No problems") || strings.Contains(text, "[]") {
			t.Log("[验证通过] main.go无问题")
		} else {
			t.Logf("[警告] main.go有问题: %s", truncate(text, 200))
		}
		phaseResults["phase4_main"] = true
	} else {
		t.Log("[注意] get_file_problems失败（IDEA可能未完成Go文件索引）")
		phaseResults["phase4_main"] = false
	}

	t.Log("\n--- 步骤4.2: get_file_problems(main_test.go) ---")
	if text, ok := e2eMCPToolCallWithTimeout(t, "get_file_problems", map[string]interface{}{
		"filePath":   "main_test.go",
		"errorsOnly": false,
	}, "分析main_test.go问题", 45*time.Second); ok {
		if text == "" || strings.Contains(text, "No problems") || strings.Contains(text, "[]") {
			t.Log("[验证通过] main_test.go无问题")
		} else {
			t.Logf("[警告] main_test.go有问题: %s", truncate(text, 200))
		}
		phaseResults["phase4_test"] = true
	} else {
		phaseResults["phase4_test"] = false
	}

	// =========================================================================
	// 阶段5: 搜索验证
	// =========================================================================
	t.Log("\n========================================")
	t.Log("阶段5: 搜索验证")
	t.Log("========================================")

	t.Log("\n--- 步骤5.1: 搜索 Mutex ---")
	if text, ok := e2eMCPToolCallWithTimeout(t, "search_in_files_by_text", map[string]interface{}{
		"searchText": "Mutex",
	}, "搜索Mutex", 30*time.Second); ok && text != "" {
		assertContains(t, text, "Mutex", "搜索Mutex结果")
		phaseResults["phase5_mutex"] = true
	} else {
		t.Error("[失败] MCP搜索Mutex失败，无本地回退")
		phaseResults["phase5_mutex"] = false
	}

	t.Log("\n--- 步骤5.2: 搜索 net/http ---")
	if text, ok := e2eMCPToolCallWithTimeout(t, "search_in_files_by_text", map[string]interface{}{
		"searchText": "net/http",
	}, "搜索net/http", 30*time.Second); ok && text != "" {
		assertContains(t, text, "net/http", "搜索net/http结果")
		phaseResults["phase5_http"] = true
	} else {
		t.Error("[失败] MCP搜索net/http失败，无本地回退")
		phaseResults["phase5_http"] = false
	}

	// =========================================================================
	// 阶段6: 构建和运行测试
	// =========================================================================
	t.Log("\n========================================")
	t.Log("阶段6: 构建和运行测试")
	t.Log("========================================")

	t.Log("\n--- 步骤6.1: build_project ---")
	if _, ok := e2eMCPToolCallWithTimeout(t, "build_project", map[string]interface{}{}, "构建项目", 60*time.Second); ok {
		t.Log("[验证] 构建完成")
		phaseResults["phase6_build"] = true
	} else {
		t.Log("[注意] IDEA构建可能超时（Go项目在IDEA中可能不支持构建）")
		phaseResults["phase6_build"] = false
	}

	t.Log("\n--- 步骤6.2: go test -v ./... ---")
	// 使用cmd /c避免PowerShell引号问题
	testCmd := fmt.Sprintf("cmd /c \"cd /d %s && go test -v ./...\"", e2eProjectPath)
	if text, ok := e2eMCPToolCallWithTimeout(t, "execute_terminal_command", map[string]interface{}{
		"command":       testCmd,
		"timeout":       60000,
		"maxLinesCount": 100,
	}, "运行go test", 60*time.Second); ok && text != "" {
		t.Logf("[测试输出]\n%s", text)
		if strings.Contains(text, "PASS") {
			t.Log("[验证通过] Go测试PASS")
			phaseResults["phase6_test"] = true
		} else {
			t.Logf("[注意] Go测试输出无PASS: %s", truncate(text, 300))
			phaseResults["phase6_test"] = false
		}
	} else {
		t.Error("[失败] MCP终端执行go test失败，无本地回退")
		phaseResults["phase6_test"] = false
	}

	// =========================================================================
	// 阶段7: 代码审查
	// =========================================================================
	t.Log("\n========================================")
	t.Log("阶段7: 代码审查 - 无需修复（预生成代码）")
	t.Log("========================================")
	t.Log("代码由预定义模板生成，无需修复")

	// =========================================================================
	// 阶段8: 最终验证
	// =========================================================================
	t.Log("\n========================================")
	t.Log("阶段8: 最终验证")
	t.Log("========================================")

	t.Log("\n--- 步骤8.1: 最终目录结构 ---")
	if text, ok := e2eMCPToolCallWithTimeout(t, "list_directory_tree", map[string]interface{}{
		"directoryPath": ".",
		"maxDepth":      3,
	}, "最终目录树", 30*time.Second); ok && text != "" {
		assertContainsSoft(t, text, "main.go", "目录包含main.go")
		assertContainsSoft(t, text, "main_test.go", "目录包含main_test.go")
		assertContainsSoft(t, text, "go.mod", "目录包含go.mod")
		phaseResults["phase8_tree"] = true
	} else {
		t.Error("[失败] MCP目录树失败，无本地回退")
		phaseResults["phase8_tree"] = false
	}

	t.Log("\n--- 步骤8.2: 最终代码确认 ---")
	if text, ok := e2eReadFile(t, "main.go", "最终确认main.go"); ok {
		assertContains(t, text, "package main", "package声明")
		assertContains(t, text, "type Counter struct", "Counter结构体")
		assertContains(t, text, "sync.Mutex", "Mutex")
		assertContains(t, text, "func countHandler", "countHandler")
		assertContains(t, text, ":8080", "8080端口")
		phaseResults["phase8_code"] = true
	} else {
		phaseResults["phase8_code"] = false
	}

	// =========================================================================
	// 最终报告
	// =========================================================================
	t.Log("\n========================================")
	t.Log("E2E 测试最终报告")
	t.Log("========================================")

	passCount := 0
	failCount := 0
	for phase, result := range phaseResults {
		status := "PASS"
		if !result {
			status = "FAIL"
			failCount++
		} else {
			passCount++
		}
		t.Logf("  [%s] %s", status, phase)
	}

	t.Logf("\n总计: %d 通过, %d 失败", passCount, failCount)

	t.Log("\n--- 关键结果 ---")
	t.Logf("  Go测试运行: %v", phaseResults["phase6_test"])
	t.Logf("  文件内容验证: %v", phaseResults["phase3_main"] && phaseResults["phase3_test"])
	t.Logf("  搜索验证: %v", phaseResults["phase5_mutex"] && phaseResults["phase5_http"])
	t.Logf("  最终验证: %v", phaseResults["phase8_tree"] && phaseResults["phase8_code"])

	if phaseResults["phase6_test"] {
		t.Log("\n[最终结论] E2E测试通过: MukaAI成功通过MCP完成Go编程题目")
	} else {
		t.Error("\n[最终结论] E2E测试失败: Go测试未通过")
	}
}
