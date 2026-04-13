//go:build manual

package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// getProjectPath 返回当前项目在IntelliJ IDEA中的路径
// IntelliJ MCP要求使用与IDE中打开的项目完全一致的路径格式
// 可通过环境变量 INTELLIJ_PROJECT_PATH 覆盖默认值
func getProjectPath() string {
	if env := os.Getenv("INTELLIJ_PROJECT_PATH"); env != "" {
		return env
	}
	// 使用已知的MukaAI项目根路径
	// 可通过环境变量 INTELLIJ_PROJECT_PATH 覆盖
	return `C:\Users\Attect\trae\AgentPlus`
}

// getIntelliJSession 创建并返回一个已连接的IntelliJ IDEA MCP会话
// 需要IntelliJ IDEA正在运行且MCP Server端口64342可用
func getIntelliJSession(t *testing.T) *MCPSession {
	t.Helper()
	cfg := ServerConfig{
		ID:        "idea",
		Enabled:   true,
		Transport: "http",
		URL:       "http://127.0.0.1:64342/stream",
		Timeout:   60,
	}
	ctx := context.Background()
	session := NewMCPSession("idea", cfg)
	if err := session.Connect(ctx); err != nil {
		t.Fatalf("连接IntelliJ MCP Server失败: %v", err)
	}
	t.Cleanup(func() { session.Close() })
	return session
}

// extractTextFromResult 从CallToolResult中提取所有文本内容
func extractTextFromResult(result *mcp.CallToolResult) string {
	var parts []string
	for _, c := range result.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			parts = append(parts, tc.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// truncate 截断字符串到指定长度
func truncate(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "")
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}
	return s
}

// callAndLog 调用工具并记录结果摘要，返回文本内容和原始结果
// 使用t.Errorf记录错误但不终止，允许后续步骤继续执行
func callAndLog(t *testing.T, ctx context.Context, session *MCPSession, toolName string, args map[string]interface{}, stepDesc string) (string, *mcp.CallToolResult) {
	t.Helper()
	t.Logf("[步骤] %s: 调用工具 %s", stepDesc, toolName)

	// 添加projectPath到参数中（如果参数中没有指定）
	if _, ok := args["projectPath"]; !ok && args != nil {
		args["projectPath"] = getProjectPath()
	}

	result, err := session.CallTool(ctx, toolName, args)
	if err != nil {
		t.Errorf("[失败] %s: 调用工具 %s 出错: %v", stepDesc, toolName, err)
		return "", nil
	}

	if result.IsError {
		errText := extractTextFromResult(result)
		t.Errorf("[失败] %s: 工具 %s 返回错误: %s", stepDesc, toolName, truncate(errText, 300))
		return errText, result
	}

	text := extractTextFromResult(result)
	summary := truncate(text, 300)
	t.Logf("[结果] %s: 成功 (长度=%d, 摘要=%q)", stepDesc, len(text), summary)
	return text, result
}

// callAndLogNoProject 与callAndLog相同，但不会自动添加projectPath
// 用于不需要projectPath的工具（如某些使用绝对路径的工具）
func callAndLogNoProject(t *testing.T, ctx context.Context, session *MCPSession, toolName string, args map[string]interface{}, stepDesc string) (string, *mcp.CallToolResult) {
	t.Helper()
	t.Logf("[步骤] %s: 调用工具 %s (不自动添加projectPath)", stepDesc, toolName)

	result, err := session.CallTool(ctx, toolName, args)
	if err != nil {
		t.Errorf("[失败] %s: 调用工具 %s 出错: %v", stepDesc, toolName, err)
		return "", nil
	}

	if result.IsError {
		errText := extractTextFromResult(result)
		t.Errorf("[失败] %s: 工具 %s 返回错误: %s", stepDesc, toolName, truncate(errText, 300))
		return errText, result
	}

	text := extractTextFromResult(result)
	summary := truncate(text, 300)
	t.Logf("[结果] %s: 成功 (长度=%d, 摘要=%q)", stepDesc, len(text), summary)
	return text, result
}

// =============================================================================
// 场景12: 工具Schema完整性检查
// =============================================================================

// TestIntelliJToolSchemaValidation 验证所有已注册工具的Schema完整性
// 运行方式: go test -tags manual -run TestIntelliJToolSchemaValidation ./internal/mcp/ -v -timeout 180s
func TestIntelliJToolSchemaValidation(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tools, err := session.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("发现工具失败: %v", err)
	}

	t.Logf("共发现 %d 个工具，开始逐一检查Schema完整性", len(tools))

	if len(tools) == 0 {
		t.Fatal("期望发现至少一个工具，但没有发现任何工具")
	}

	toolNames := make(map[string]bool)
	for i, tool := range tools {
		t.Run(fmt.Sprintf("tool_%02d_%s", i+1, tool.Name), func(t *testing.T) {
			// 检查Name非空
			if tool.Name == "" {
				t.Error("工具名称为空")
				return
			}

			// 检查名称唯一性
			if toolNames[tool.Name] {
				t.Errorf("工具名称重复: %s", tool.Name)
			}
			toolNames[tool.Name] = true

			// 检查Description非空
			if tool.Description == "" && tool.Title == "" {
				t.Errorf("工具 %s: Description和Title都为空", tool.Name)
			} else {
				t.Logf("描述: %s", truncate(tool.Description, 100))
			}

			// 检查InputSchema非nil
			if tool.InputSchema == nil {
				t.Errorf("工具 %s: InputSchema为nil", tool.Name)
			} else {
				// 解析schema，检查是否有properties字段
				schemaMap, ok := tool.InputSchema.(map[string]interface{})
				if ok {
					t.Logf("Schema类型: %v, 属性数: %d",
						schemaMap["type"],
						len(schemaMap))
				}
			}
		})
	}

	t.Logf("Schema完整性检查完成，共检查 %d 个工具", len(tools))
}

// =============================================================================
// 场景1: 项目探索（模拟新开发者了解项目）
// =============================================================================

// TestIntelliJProjectExploration 测试项目探索相关工具
// 运行方式: go test -tags manual -run TestIntelliJProjectExploration ./internal/mcp/ -v -timeout 180s
func TestIntelliJProjectExploration(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 步骤1: get_project_modules - 了解项目模块结构
	text, _ := callAndLog(t, ctx, session, "get_project_modules", map[string]interface{}{}, "获取项目模块")
	if text == "" {
		t.Error("[验证失败] get_project_modules 返回空内容")
	} else {
		t.Logf("[验证] 项目模块信息长度: %d", len(text))
	}

	// 步骤2: list_directory_tree - 查看项目目录树
	text, _ = callAndLog(t, ctx, session, "list_directory_tree", map[string]interface{}{
		"directoryPath": ".",
		"maxDepth":      2,
	}, "查看项目根目录树(深度2)")
	if text == "" {
		t.Error("[验证失败] list_directory_tree 返回空内容")
	} else {
		t.Log("[验证通过] 目录树信息获取成功")
	}

	// 步骤3: find_files_by_glob - 搜索Go源文件
	text, _ = callAndLog(t, ctx, session, "find_files_by_glob", map[string]interface{}{
		"globPattern":    "**/*.go",
		"fileCountLimit": 20,
	}, "搜索Go源文件(限制20个)")
	if text == "" {
		t.Error("[验证失败] find_files_by_glob 返回空内容")
	} else if !strings.Contains(text, ".go") {
		t.Error("[验证失败] find_files_by_glob 结果中未包含.go文件")
	} else {
		t.Log("[验证通过] 找到Go源文件")
	}

	// 步骤4: search_text - 搜索核心结构体定义
	text, _ = callAndLog(t, ctx, session, "search_text", map[string]interface{}{
		"query": "type Agent struct",
	}, "搜索Agent结构体定义")

	// 步骤5: search_file - 按glob模式搜索文件
	text, _ = callAndLog(t, ctx, session, "search_file", map[string]interface{}{
		"patterns": []string{"**/*_test.go"},
		"limit":    10,
	}, "搜索测试文件")
	if text == "" {
		t.Log("[注意] search_file 返回空")
	} else {
		t.Log("[验证通过] search_file 搜索成功")
	}
}

// =============================================================================
// 场景2: 代码阅读（模拟理解某个功能模块）
// =============================================================================

// TestIntelliJCodeReading 测试代码阅读相关工具
// 运行方式: go test -tags manual -run TestIntelliJCodeReading ./internal/mcp/ -v -timeout 180s
func TestIntelliJCodeReading(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 步骤1: read_file - 读取核心文件（带行号）
	// read_file工具使用file_path参数（非path），同时也需要projectPath
	text, _ := callAndLog(t, ctx, session, "read_file", map[string]interface{}{
		"file_path":  "internal/agent/core.go",
		"start_line": 1,
		"max_lines":  30,
	}, "读取core.go前30行")
	if text == "" {
		t.Error("[验证失败] read_file 返回空内容")
	} else if !strings.Contains(text, "package") {
		t.Error("[验证失败] read_file 结果中未包含 'package' 关键字")
	} else {
		t.Log("[验证通过] 文件内容包含package声明")
	}

	// 步骤2: get_file_text_by_path - 获取完整文件内容
	text, _ = callAndLog(t, ctx, session, "get_file_text_by_path", map[string]interface{}{
		"pathInProject": "go.mod",
		"maxLinesCount": 20,
	}, "读取go.mod前20行")
	if text == "" {
		t.Error("[验证失败] get_file_text_by_path 返回空内容")
	} else if !strings.Contains(text, "module") {
		t.Error("[验证失败] go.mod中未包含 'module'")
	} else {
		t.Log("[验证通过] go.mod内容读取成功")
	}

	// 步骤3: search_symbol - 搜索符号
	text, _ = callAndLog(t, ctx, session, "search_symbol", map[string]interface{}{
		"query": "MCPSession",
		"limit": 10,
	}, "搜索MCPSession符号")
	if text == "" {
		t.Log("[注意] search_symbol 未找到 MCPSession")
	} else {
		t.Log("[验证通过] search_symbol 搜索成功")
	}

	// 步骤4: search_regex - 搜索Agent的所有方法
	text, _ = callAndLog(t, ctx, session, "search_regex", map[string]interface{}{
		"pattern": `func \(a \*Agent\) \w+`,
		"limit":   20,
	}, "搜索Agent的所有方法")
	if text == "" {
		t.Log("[注意] search_regex 未找到Agent方法")
	} else {
		t.Log("[验证通过] search_regex 搜索成功")
	}

	// 步骤5: get_symbol_info - 获取符号信息（需要文件+行+列）
	text, _ = callAndLog(t, ctx, session, "get_symbol_info", map[string]interface{}{
		"filePath": "internal/mcp/session.go",
		"line":     17,
		"column":   6,
	}, "获取MCPSession结构体符号信息")
	if text == "" {
		t.Log("[注意] get_symbol_info 返回空")
	} else {
		t.Log("[验证通过] get_symbol_info 获取成功")
	}
}

// =============================================================================
// 场景3: 代码搜索（模拟查找Bug/功能定位）
// =============================================================================

// TestIntelliJCodeSearch 测试代码搜索相关工具
// 运行方式: go test -tags manual -run TestIntelliJCodeSearch ./internal/mcp/ -v -timeout 180s
func TestIntelliJCodeSearch(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 步骤1: search_in_files_by_text - 全文搜索
	text, _ := callAndLog(t, ctx, session, "search_in_files_by_text", map[string]interface{}{
		"text": "MCPSession",
	}, "全文搜索MCPSession")
	if text == "" {
		t.Error("[验证失败] search_in_files_by_text 返回空")
	} else {
		t.Log("[验证通过] search_in_files_by_text 搜索成功")
	}

	// 步骤2: search_in_files_by_regex - 搜索TODO/FIXME/HACK
	text, _ = callAndLog(t, ctx, session, "search_in_files_by_regex", map[string]interface{}{
		"regex": "TODO|FIXME|HACK",
	}, "搜索TODO/FIXME/HACK标记")
	if text == "" {
		t.Log("[注意] search_in_files_by_regex 未找到匹配（可能项目中无此类标记）")
	} else {
		t.Log("[验证通过] 找到TODO/FIXME/HACK标记")
	}

	// 步骤3: find_files_by_name_keyword - 查找测试文件
	text, _ = callAndLog(t, ctx, session, "find_files_by_name_keyword", map[string]interface{}{
		"nameKeyword":    "test",
		"fileCountLimit": 15,
	}, "查找测试相关文件")
	if text == "" {
		t.Error("[验证失败] find_files_by_name_keyword 返回空")
	} else {
		t.Log("[验证通过] 找到测试文件")
	}

	// 步骤4: get_all_open_file_paths - 获取当前打开的文件
	text, _ = callAndLog(t, ctx, session, "get_all_open_file_paths", map[string]interface{}{}, "获取IDE当前打开的文件")
	t.Logf("[验证] 当前打开的文件信息长度: %d", len(text))
}

// =============================================================================
// 场景4: 问题分析（模拟代码审查）
// =============================================================================

// TestIntelliJProblemAnalysis 测试问题分析相关工具
// 运行方式: go test -tags manual -run TestIntelliJProblemAnalysis ./internal/mcp/ -v -timeout 180s
func TestIntelliJProblemAnalysis(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 步骤1: get_file_problems - 分析core.go（使用filePath参数）
	text, _ := callAndLog(t, ctx, session, "get_file_problems", map[string]interface{}{
		"filePath":   "internal/agent/core.go",
		"errorsOnly": false,
	}, "分析core.go的代码问题(含警告)")
	t.Logf("[验证] core.go 问题分析结果长度: %d", len(text))
	// 问题为空也是正常的（代码质量好）
	if text == "" || strings.Contains(text, "No problems") || strings.Contains(text, "[]") {
		t.Log("[验证] core.go 无严重问题")
	}

	// 步骤2: get_file_problems - 分析session.go
	text, _ = callAndLog(t, ctx, session, "get_file_problems", map[string]interface{}{
		"filePath":   "internal/mcp/session.go",
		"errorsOnly": true,
	}, "分析session.go的代码错误")
	t.Logf("[验证] session.go 错误分析结果长度: %d", len(text))

	// 步骤3: get_file_problems - 分析MCP manager.go
	text, _ = callAndLog(t, ctx, session, "get_file_problems", map[string]interface{}{
		"filePath":   "internal/mcp/manager.go",
		"errorsOnly": false,
	}, "分析manager.go的代码问题")
	t.Logf("[验证] manager.go 问题分析结果长度: %d", len(text))
}

// =============================================================================
// 场景5: 文件操作（模拟创建新功能文件）
// =============================================================================

// TestIntelliJFileOperations 测试文件创建、读取、编辑、删除完整流程
// 运行方式: go test -tags manual -run TestIntelliJFileOperations ./internal/mcp/ -v -timeout 180s
func TestIntelliJFileOperations(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	tempFilePath := "test_mcp_temp.txt"

	// 确保最终清理测试文件
	t.Cleanup(func() {
		// 通过MCP尝试删除
		_, _ = session.CallTool(context.Background(), "delete_file", map[string]interface{}{
			"projectPath":   getProjectPath(),
			"pathInProject": tempFilePath,
		})
		// 本地删除兜底
		_ = os.Remove(tempFilePath)
	})

	// 步骤1: create_new_file - 创建测试文件
	text, _ := callAndLog(t, ctx, session, "create_new_file", map[string]interface{}{
		"pathInProject": tempFilePath,
		"content":       "MCP test file - created by integration test",
	}, "创建测试文件")
	if text == "" {
		t.Error("[验证失败] create_new_file 返回空")
	}

	// 步骤2: read_file - 验证创建的文件内容（使用file_path参数）
	text, _ = callAndLog(t, ctx, session, "read_file", map[string]interface{}{
		"file_path": tempFilePath,
	}, "读取测试文件验证内容")
	if text == "" {
		t.Error("[验证失败] read_file 返回空，文件可能未成功创建")
	} else if !strings.Contains(text, "MCP test") {
		t.Errorf("[验证失败] 文件内容不包含预期文本 'MCP test'，实际内容: %s", truncate(text, 200))
	} else {
		t.Log("[验证通过] 文件内容包含预期文本")
	}

	// 步骤3: get_file_text_by_path - 另一种方式读取验证
	text, _ = callAndLog(t, ctx, session, "get_file_text_by_path", map[string]interface{}{
		"pathInProject": tempFilePath,
	}, "使用get_file_text_by_path读取验证")
	if text != "" && strings.Contains(text, "MCP test") {
		t.Log("[验证通过] get_file_text_by_path 确认文件内容正确")
	}

	// 步骤4: replace_text_in_file - 编辑文件（注意参数名: pathInProject, oldText, newText）
	text, _ = callAndLog(t, ctx, session, "replace_text_in_file", map[string]interface{}{
		"pathInProject": tempFilePath,
		"oldText":       "MCP test",
		"newText":       "MCP verified",
	}, "替换文件文本: MCP test -> MCP verified")
	if text == "" {
		t.Error("[验证失败] replace_text_in_file 返回空")
	} else if strings.Contains(text, "ok") {
		t.Log("[验证通过] 文本替换操作成功")
	}

	// 步骤5: read_file - 验证修改后的内容
	text, _ = callAndLog(t, ctx, session, "read_file", map[string]interface{}{
		"file_path": tempFilePath,
	}, "读取修改后的文件内容")
	if text != "" {
		if strings.Contains(text, "MCP verified") {
			t.Log("[验证通过] 文件内容已成功修改为 'MCP verified'")
		} else if strings.Contains(text, "MCP test") {
			t.Log("[注意] 文件内容仍为原始值 'MCP test'，替换操作可能未生效")
		} else {
			t.Logf("[注意] 文件内容: %s", truncate(text, 200))
		}
	}

	// 步骤6: delete_file - 清理测试文件
	// IntelliJ MCP没有专门的delete_file，通过replace清空或使用其他方式
	// 尝试使用terminal命令删除
	text, _ = callAndLog(t, ctx, session, "execute_terminal_command", map[string]interface{}{
		"command": "del " + tempFilePath,
		"timeout": 10,
	}, "通过终端命令删除测试文件")
	// 也用本地方式确保清理
	os.Remove(tempFilePath)

	// 验证文件已删除
	text, _ = callAndLog(t, ctx, session, "read_file", map[string]interface{}{
		"file_path": tempFilePath,
	}, "验证文件已删除")
	if text == "" {
		t.Log("[验证通过] 测试文件已成功清理")
	} else {
		t.Log("[注意] 测试文件可能未删除，依赖Cleanup兜底")
	}
}

// =============================================================================
// 场景6: 构建和运行（模拟CI流程）
// =============================================================================

// TestIntelliJBuildAndRun 测试构建和运行配置相关工具
// 运行方式: go test -tags manual -run TestIntelliJBuildAndRun ./internal/mcp/ -v -timeout 180s
func TestIntelliJBuildAndRun(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	// 步骤1: build_project - 构建项目
	text, result := callAndLog(t, ctx, session, "build_project", map[string]interface{}{}, "构建项目")
	if result == nil {
		t.Log("[注意] build_project 返回nil结果")
	} else {
		t.Logf("[验证] 构建项目结果: isError=%v, 长度=%d", result.IsError, len(text))
		if !result.IsError {
			t.Log("[验证通过] 项目构建成功（无错误）")
		} else {
			t.Log("[注意] 项目构建有错误，这可能是正常的（取决于项目状态）")
		}
	}

	// 步骤2: get_run_configurations - 获取运行配置
	text, _ = callAndLog(t, ctx, session, "get_run_configurations", map[string]interface{}{}, "获取运行配置")
	if text == "" {
		t.Log("[注意] get_run_configurations 返回空，项目可能没有配置运行项")
	} else {
		t.Logf("[验证通过] 运行配置信息长度: %d", len(text))
	}
}

// =============================================================================
// 场景7: PSI和代码分析（模拟深度代码理解）
// =============================================================================

// TestIntelliJPSIAnalysis 测试PSI树生成和符号分析
// 运行方式: go test -tags manual -run TestIntelliJPSIAnalysis ./internal/mcp/ -v -timeout 180s
func TestIntelliJPSIAnalysis(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 步骤1: generate_psi_tree - 生成PSI树（需要code参数和language参数，而非文件路径）
	text, _ := callAndLog(t, ctx, session, "generate_psi_tree", map[string]interface{}{
		"code":     "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}",
		"language": "Java", // 使用Java解析器作为Go可能不支持，但仍然测试接口
	}, "生成简单代码的PSI树")
	if text == "" {
		t.Log("[注意] generate_psi_tree 返回空，可能不支持Go语言")
	} else {
		t.Logf("[验证通过] PSI树生成成功，长度: %d", len(text))
		if strings.Contains(text, "Psi") || strings.Contains(text, "METHOD") || strings.Contains(text, "CLASS") {
			t.Log("[验证通过] PSI树内容包含PSI节点信息")
		}
	}

	// 步骤2: search_symbol - 搜索配置结构体
	text, _ = callAndLog(t, ctx, session, "search_symbol", map[string]interface{}{
		"query": "MCPConfig",
		"limit": 5,
	}, "搜索MCPConfig结构体")
	if text == "" {
		t.Log("[注意] search_symbol 未找到 MCPConfig（Go符号搜索可能受限）")
	} else {
		t.Log("[验证通过] search_symbol 搜索成功")
	}

	// 步骤3: get_symbol_info - 获取符号详情（需要指定文件+行+列位置）
	text, _ = callAndLog(t, ctx, session, "get_symbol_info", map[string]interface{}{
		"filePath": "internal/mcp/config.go",
		"line":     14,
		"column":   6,
	}, "获取MCPConfig结构体符号详情(第14行)")
	t.Logf("[验证] get_symbol_info 结果长度: %d", len(text))

	// 步骤4: generate_inspection_kts_api - 获取Inspection KTS API文档
	text, _ = callAndLog(t, ctx, session, "generate_inspection_kts_api", map[string]interface{}{
		"language": "Java",
	}, "获取Java Inspection KTS API文档")
	t.Logf("[验证] Inspection KTS API 文档长度: %d", len(text))
}

// =============================================================================
// 场景8: 格式化和重构（模拟代码优化）
// =============================================================================

// TestIntelliJReformat 测试文件格式化
// 运行方式: go test -tags manual -run TestIntelliJReformat ./internal/mcp/ -v -timeout 180s
func TestIntelliJReformat(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 创建一个格式不规范的临时文件用于测试格式化
	tempFile := "test_mcp_reformat_temp.go"
	t.Cleanup(func() {
		os.Remove(tempFile)
	})

	// 步骤1: 创建测试文件
	uglyCode := `package main
import "fmt"
func main(){
fmt.Println("hello")
x:=1+2
if x>0{fmt.Println(x)}
}`
	callAndLog(t, ctx, session, "create_new_file", map[string]interface{}{
		"pathInProject": tempFile,
		"content":       uglyCode,
		"overwrite":     true,
	}, "创建格式不规范的测试Go文件")

	// 步骤2: reformat_file - 格式化文件
	text, _ := callAndLog(t, ctx, session, "reformat_file", map[string]interface{}{
		"path": tempFile,
	}, "格式化测试Go文件")
	if text == "" {
		t.Log("[注意] reformat_file 返回空")
	} else {
		t.Log("[验证通过] 格式化操作返回了结果")
	}

	// 步骤3: 读取格式化后的文件
	text, _ = callAndLog(t, ctx, session, "read_file", map[string]interface{}{
		"file_path": tempFile,
	}, "读取格式化后的文件内容")
	if text != "" {
		t.Logf("[验证] 格式化后文件内容长度: %d", len(text))
	}

	// 步骤4: 清理
	callAndLog(t, ctx, session, "execute_terminal_command", map[string]interface{}{
		"command": "del " + tempFile,
		"timeout": 5,
	}, "删除临时Go文件")
	os.Remove(tempFile)
}

// =============================================================================
// 场景9: 依赖和模块分析
// =============================================================================

// TestIntelliJDependencyAnalysis 测试依赖和模块分析工具
// 运行方式: go test -tags manual -run TestIntelliJDependencyAnalysis ./internal/mcp/ -v -timeout 180s
func TestIntelliJDependencyAnalysis(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 步骤1: get_project_dependencies - 获取项目依赖
	text, _ := callAndLog(t, ctx, session, "get_project_dependencies", map[string]interface{}{}, "获取项目依赖")
	if text == "" {
		t.Log("[注意] get_project_dependencies 返回空")
	} else {
		t.Logf("[验证] 项目依赖信息长度: %d", len(text))
		// Go项目应该包含一些依赖信息
		if strings.Contains(text, "github.com") || strings.Contains(text, "wails") || strings.Contains(text, "mcp") {
			t.Log("[验证通过] 依赖信息包含已知的Go模块")
		}
	}

	// 步骤2: get_project_modules - 获取模块列表
	text, _ = callAndLog(t, ctx, session, "get_project_modules", map[string]interface{}{}, "获取项目模块列表")
	if text == "" {
		t.Error("[验证失败] get_project_modules 返回空内容")
	} else {
		t.Logf("[验证通过] 项目模块信息长度: %d", len(text))
	}
}

// =============================================================================
// 场景10: VCS操作
// =============================================================================

// TestIntelliJVCSOperations 测试VCS相关工具
// 运行方式: go test -tags manual -run TestIntelliJVCSOperations ./internal/mcp/ -v -timeout 180s
func TestIntelliJVCSOperations(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 步骤1: get_repositories - 获取VCS仓库信息
	text, _ := callAndLog(t, ctx, session, "get_repositories", map[string]interface{}{}, "获取VCS仓库信息")
	if text == "" {
		t.Log("[注意] get_repositories 返回空，项目可能未被IDE识别为VCS仓库")
	} else {
		t.Logf("[验证通过] VCS仓库信息长度: %d", len(text))
		if strings.Contains(text, "git") || strings.Contains(text, "Git") {
			t.Log("[验证通过] VCS信息包含Git相关内容")
		}
	}
}

// =============================================================================
// 场景11: 终端操作
// =============================================================================

// TestIntelliJTerminalOperation 测试终端命令执行
// 运行方式: go test -tags manual -run TestIntelliJTerminalOperation ./internal/mcp/ -v -timeout 180s
func TestIntelliJTerminalOperation(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 步骤1: execute_terminal_command - 执行终端命令
	text, _ := callAndLog(t, ctx, session, "execute_terminal_command", map[string]interface{}{
		"command":        "echo MCP integration test",
		"timeout":        10,
		"maxLinesCount":  50,
		"executeInShell": true,
	}, "执行echo命令测试终端")
	if text == "" {
		t.Log("[注意] execute_terminal_command 返回空（可能需要用户在IDE中确认）")
	} else {
		t.Logf("[验证] 终端命令执行结果长度: %d", len(text))
		if strings.Contains(text, "MCP") {
			t.Log("[验证通过] 终端命令输出包含预期内容")
		}
	}
}

// =============================================================================
// 场景13: 打开文件和编辑器操作
// =============================================================================

// TestIntelliJEditorOperations 测试编辑器相关操作
// 运行方式: go test -tags manual -run TestIntelliJEditorOperations ./internal/mcp/ -v -timeout 180s
func TestIntelliJEditorOperations(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 步骤1: open_file_in_editor - 在编辑器中打开文件
	text, _ := callAndLog(t, ctx, session, "open_file_in_editor", map[string]interface{}{
		"filePath": "internal/mcp/session.go",
	}, "在编辑器中打开session.go")
	if text == "" {
		t.Log("[注意] open_file_in_editor 返回空")
	} else {
		t.Log("[验证通过] 文件已在编辑器中打开")
	}

	// 步骤2: get_all_open_file_paths - 验证文件已打开
	text, _ = callAndLog(t, ctx, session, "get_all_open_file_paths", map[string]interface{}{}, "获取当前打开的文件列表")
	if text != "" && strings.Contains(text, "session.go") {
		t.Log("[验证通过] 打开的文件列表中包含session.go")
	} else {
		t.Logf("[注意] 打开的文件列表中可能不包含session.go（IDE可能已关闭该标签页）")
	}
}

// =============================================================================
// 场景14: 高级分析工具
// =============================================================================

// TestIntelliJAdvancedAnalysis 测试高级分析工具（锁需求、线程需求）
// 运行方式: go test -tags manual -run TestIntelliJAdvancedAnalysis ./internal/mcp/ -v -timeout 180s
func TestIntelliJAdvancedAnalysis(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 步骤1: find_lock_requirements_usages - 分析锁需求
	text, _ := callAndLog(t, ctx, session, "find_lock_requirements_usages", map[string]interface{}{
		"filePath": "internal/mcp/session.go",
		"line":     128,
		"column":   15,
	}, "分析CallTool方法的锁需求")
	if text == "" {
		t.Log("[注意] find_lock_requirements_usages 返回空（Go文件可能不支持此分析）")
	} else {
		t.Logf("[验证] 锁需求分析结果长度: %d", len(text))
	}

	// 步骤2: find_threading_requirements_usages - 分析线程需求
	text, _ = callAndLog(t, ctx, session, "find_threading_requirements_usages", map[string]interface{}{
		"filePath": "internal/mcp/session.go",
		"line":     128,
		"column":   15,
	}, "分析CallTool方法的线程需求")
	if text == "" {
		t.Log("[注意] find_threading_requirements_usages 返回空（Go文件可能不支持此分析）")
	} else {
		t.Logf("[验证] 线程需求分析结果长度: %d", len(text))
	}

	// 步骤3: generate_inspection_kts_examples - 获取Inspection KTS示例
	text, _ = callAndLog(t, ctx, session, "generate_inspection_kts_examples", map[string]interface{}{
		"language": "Java",
	}, "获取Java Inspection KTS示例")
	t.Logf("[验证] Inspection KTS 示例长度: %d", len(text))
}

// =============================================================================
// 综合测试: 遍历所有工具的调用可用性
// =============================================================================

// TestIntelliJAllToolsCallability 测试每个工具是否能被正确调用（使用正确的参数）
// 运行方式: go test -tags manual -run TestIntelliJAllToolsCallability ./internal/mcp/ -v -timeout 180s
func TestIntelliJAllToolsCallability(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 首先发现所有工具
	tools, err := session.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("发现工具失败: %v", err)
	}

	t.Logf("共发现 %d 个工具，开始逐一测试调用可用性", len(tools))

	// 为每个工具准备安全的测试参数（基于实际Schema）
	safeArgs := map[string]map[string]interface{}{
		// 不需要特殊参数的工具（只需要projectPath，由callAndLog自动添加）
		"get_project_modules":      {},
		"get_project_dependencies": {},
		"get_run_configurations":   {},
		"get_repositories":         {},
		"get_all_open_file_paths":  {},
		"build_project":            {},
		// 需要路径参数的工具
		"read_file":             {"file_path": "go.mod", "start_line": 1, "max_lines": 5},
		"get_file_text_by_path": {"pathInProject": "go.mod", "maxLinesCount": 5},
		"list_directory_tree":   {"directoryPath": ".", "maxDepth": 1},
		"get_file_problems":     {"filePath": "go.mod", "errorsOnly": true},
		"reformat_file":         {"path": "go.mod"},
		"open_file_in_editor":   {"filePath": "go.mod"},
		// 搜索工具
		"search_text":                {"query": "package"},
		"search_regex":               {"pattern": "package"},
		"search_symbol":              {"query": "main"},
		"search_file":                {"patterns": []string{"*.go"}, "limit": 5},
		"search_in_files_by_text":    {"text": "package"},
		"search_in_files_by_regex":   {"regex": "package"},
		"find_files_by_glob":         {"globPattern": "*.go", "fileCountLimit": 5},
		"find_files_by_name_keyword": {"nameKeyword": "go", "fileCountLimit": 5},
		// 符号信息（需要文件+位置）
		"get_symbol_info": {"filePath": "internal/mcp/session.go", "line": 17, "column": 6},
		// 文件操作（使用临时文件）
		"create_new_file":      {"pathInProject": "test_mcp_callability.txt", "content": "test"},
		"replace_text_in_file": {"pathInProject": "test_mcp_callability.txt", "oldText": "test", "newText": "ok"},
		// 终端命令
		"execute_terminal_command": {"command": "echo callability_test", "timeout": 5},
		// PSI树（需要code+language）
		"generate_psi_tree": {"code": "class Foo {}", "language": "Java"},
		// 高级分析
		"find_lock_requirements_usages":      {"filePath": "internal/mcp/session.go", "line": 17, "column": 6},
		"find_threading_requirements_usages": {"filePath": "internal/mcp/session.go", "line": 17, "column": 6},
		// Inspection KTS
		"generate_inspection_kts_api":      {"language": "Java"},
		"generate_inspection_kts_examples": {"language": "Java"},
		// 需要特殊条件的工具（跳过实际执行）
		"rename_refactoring":        nil, // 危险操作，不自动测试
		"run_inspection_kts":        nil, // 需要脚本内容
		"runNotebookCell":           nil, // 需要notebook文件
		"execute_run_configuration": nil, // 需要运行配置名
	}

	passed := 0
	failed := 0
	errored := 0
	skipped := 0

	for _, tool := range tools {
		t.Run(tool.Name, func(t *testing.T) {
			args, hasArgs := safeArgs[tool.Name]
			if hasArgs && args == nil {
				t.Skipf("工具 %s 被标记为危险/需要特殊条件，跳过自动测试", tool.Name)
				skipped++
				return
			}

			// 为参数添加projectPath（如果配置了）
			if args == nil {
				args = map[string]interface{}{}
			}
			if pp := getProjectPath(); pp != "" {
				if _, ok := args["projectPath"]; !ok {
					args["projectPath"] = pp
				}
			}

			result, err := session.CallTool(ctx, tool.Name, args)
			if err != nil {
				t.Errorf("[调用失败] 工具 %s: %v", tool.Name, err)
				errored++
				return
			}

			text := ""
			if result != nil {
				text = extractTextFromResult(result)
			}

			if result != nil && result.IsError {
				// 文件操作类的错误可能是预期中的（如替换文本找不到）
				t.Logf("[工具错误] 工具 %s 返回错误: %s", tool.Name, truncate(text, 200))
				failed++
				return
			}

			passed++
			t.Logf("[通过] 工具 %s: 结果长度=%d", tool.Name, len(text))
		})
	}

	// 清理临时文件
	_, _ = session.CallTool(ctx, "execute_terminal_command", map[string]interface{}{
		"command":     "del test_mcp_callability.txt",
		"timeout":     5,
		"projectPath": getProjectPath(),
	})
	os.Remove("test_mcp_callability.txt")

	t.Logf("\n========== 工具调用可用性汇总 ==========")
	t.Logf("总计: %d 工具", len(tools))
	t.Logf("通过: %d", passed)
	t.Logf("工具错误(预期内): %d", failed)
	t.Logf("调用失败: %d", errored)
	t.Logf("跳过: %d", skipped)
}

// =============================================================================
// 工具发现详情测试
// =============================================================================

// TestIntelliJToolDiscoveryDetail 详细列出所有工具的Schema信息
// 运行方式: go test -tags manual -run TestIntelliJToolDiscoveryDetail ./internal/mcp/ -v -timeout 180s
func TestIntelliJToolDiscoveryDetail(t *testing.T) {
	session := getIntelliJSession(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tools, err := session.DiscoverTools(ctx)
	if err != nil {
		t.Fatalf("发现工具失败: %v", err)
	}

	t.Logf("\n========== IntelliJ MCP 工具清单 (%d 个) ==========", len(tools))

	for i, tool := range tools {
		t.Logf("\n--- 工具 %02d: %s ---", i+1, tool.Name)
		if tool.Title != "" {
			t.Logf("标题: %s", tool.Title)
		}
		t.Logf("描述: %s", truncate(tool.Description, 150))

		// 打印InputSchema的属性列表
		if tool.InputSchema != nil {
			schemaMap, ok := tool.InputSchema.(map[string]interface{})
			if ok {
				if props, ok := schemaMap["properties"].(map[string]interface{}); ok {
					propNames := make([]string, 0, len(props))
					for k := range props {
						propNames = append(propNames, k)
					}
					t.Logf("参数列表: %v", propNames)
				}
				if required, ok := schemaMap["required"].([]interface{}); ok {
					t.Logf("必需参数: %v", required)
				}

				// 打印完整schema（限制长度）
				schemaBytes, err := json.MarshalIndent(tool.InputSchema, "", "  ")
				if err == nil {
					schemaStr := string(schemaBytes)
					if len(schemaStr) > 800 {
						schemaStr = schemaStr[:800] + "\n  ... (truncated)"
					}
					t.Logf("Schema:\n  %s", schemaStr)
				}
			}
		}
	}
}

// =============================================================================
// 综合全场景测试入口
// =============================================================================

// TestIntelliJComprehensive 运行所有场景的综合测试
// 这个测试按顺序执行所有场景，提供完整的MCP工具验证
// 运行方式: go test -tags manual -run TestIntelliJComprehensive ./internal/mcp/ -v -timeout 180s
func TestIntelliJComprehensive(t *testing.T) {
	session := getIntelliJSession(t)

	t.Log("========== IntelliJ IDEA MCP 综合集成测试 ==========")

	// 阶段0: 工具发现与Schema验证
	t.Log("\n>>> 阶段0: 工具发现与Schema验证")
	ctx0, cancel0 := context.WithTimeout(context.Background(), 30*time.Second)
	tools, err := session.DiscoverTools(ctx0)
	cancel0()
	if err != nil {
		t.Fatalf("工具发现失败: %v", err)
	}
	t.Logf("发现 %d 个工具", len(tools))
	for _, tool := range tools {
		t.Logf("  - %s", tool.Name)
	}
	if len(tools) < 20 {
		t.Errorf("工具数量偏少 (%d)，期望至少20个", len(tools))
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	// 阶段1: 项目探索
	t.Log("\n>>> 阶段1: 项目探索")
	callAndLog(t, ctx, session, "get_project_modules", map[string]interface{}{}, "S1-1 获取模块")
	callAndLog(t, ctx, session, "list_directory_tree", map[string]interface{}{"directoryPath": ".", "maxDepth": 1}, "S1-2 目录树")
	callAndLog(t, ctx, session, "find_files_by_glob", map[string]interface{}{"globPattern": "*.go", "fileCountLimit": 10}, "S1-3 搜索Go文件")

	// 阶段2: 代码阅读
	t.Log("\n>>> 阶段2: 代码阅读")
	callAndLog(t, ctx, session, "read_file", map[string]interface{}{"file_path": "go.mod", "max_lines": 10}, "S2-1 读取go.mod")
	callAndLog(t, ctx, session, "get_file_text_by_path", map[string]interface{}{"pathInProject": "go.mod", "maxLinesCount": 10}, "S2-2 获取文件文本")
	callAndLog(t, ctx, session, "search_symbol", map[string]interface{}{"q": "MCPSession", "limit": 5}, "S2-3 搜索符号")

	// 阶段3: 代码搜索
	t.Log("\n>>> 阶段3: 代码搜索")
	callAndLog(t, ctx, session, "search_in_files_by_text", map[string]interface{}{"searchText": "package"}, "S3-1 全文搜索")
	callAndLog(t, ctx, session, "search_in_files_by_regex", map[string]interface{}{"regexPattern": "TODO"}, "S3-2 正则搜索")
	callAndLog(t, ctx, session, "find_files_by_name_keyword", map[string]interface{}{"nameKeyword": "mcp", "fileCountLimit": 10}, "S3-3 按名称搜索")

	// 阶段4: 问题分析
	t.Log("\n>>> 阶段4: 问题分析")
	callAndLog(t, ctx, session, "get_file_problems", map[string]interface{}{"filePath": "go.mod", "errorsOnly": true}, "S4-1 分析go.mod问题")

	// 阶段5: 文件操作
	t.Log("\n>>> 阶段5: 文件操作")
	tempFile := "test_mcp_comprehensive_temp.txt"
	callAndLog(t, ctx, session, "create_new_file", map[string]interface{}{
		"pathInProject": tempFile, "text": "comprehensive test file content", "overwrite": true,
	}, "S5-1 创建文件")
	callAndLog(t, ctx, session, "get_file_text_by_path", map[string]interface{}{
		"pathInProject": tempFile,
	}, "S5-2 读取新文件")
	callAndLog(t, ctx, session, "replace_text_in_file", map[string]interface{}{
		"pathInProject": tempFile, "oldText": "comprehensive test", "newText": "verified test",
	}, "S5-3 替换文本")
	// 清理
	callAndLog(t, ctx, session, "execute_terminal_command", map[string]interface{}{
		"command": "cmd /c del " + tempFile, "timeout": 5000,
	}, "S5-4 清理临时文件")
	os.Remove(tempFile)

	// 阶段6: 构建和运行
	t.Log("\n>>> 阶段6: 构建和运行")
	callAndLog(t, ctx, session, "build_project", map[string]interface{}{}, "S6-1 构建项目")
	callAndLog(t, ctx, session, "get_run_configurations", map[string]interface{}{}, "S6-2 获取运行配置")

	// 阶段7: PSI分析
	t.Log("\n>>> 阶段7: PSI分析")
	callAndLog(t, ctx, session, "generate_psi_tree", map[string]interface{}{
		"code": "class Test { void run() {} }", "language": "Java",
	}, "S7-1 PSI树")

	// 阶段8: 依赖分析
	t.Log("\n>>> 阶段8: 依赖分析")
	callAndLog(t, ctx, session, "get_project_dependencies", map[string]interface{}{}, "S8-1 项目依赖")

	// 阶段9: VCS
	t.Log("\n>>> 阶段9: VCS操作")
	callAndLog(t, ctx, session, "get_repositories", map[string]interface{}{}, "S9-1 VCS仓库")

	// 阶段10: 终端
	t.Log("\n>>> 阶段10: 终端操作")
	callAndLog(t, ctx, session, "execute_terminal_command", map[string]interface{}{
		"command": "cmd /c echo comprehensive test passed", "timeout": 10000,
	}, "S10-1 终端命令")

	// 阶段11: 编辑器操作
	t.Log("\n>>> 阶段11: 编辑器操作")
	callAndLog(t, ctx, session, "open_file_in_editor", map[string]interface{}{
		"filePath": "internal/mcp/session.go",
	}, "S11-1 打开文件")
	callAndLog(t, ctx, session, "get_all_open_file_paths", map[string]interface{}{}, "S11-2 获取打开文件列表")

	// 阶段12: 高级分析
	t.Log("\n>>> 阶段12: 高级分析")
	callAndLog(t, ctx, session, "get_symbol_info", map[string]interface{}{
		"filePath": "internal/mcp/config.go", "line": 14, "column": 6,
	}, "S12-1 符号信息")
	callAndLog(t, ctx, session, "generate_inspection_kts_api", map[string]interface{}{
		"language": "Java",
	}, "S12-2 Inspection API")

	t.Log("\n========== 综合测试完成 ==========")
}
