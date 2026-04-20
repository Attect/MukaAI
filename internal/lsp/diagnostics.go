package lsp

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Attect/MukaAI/internal/tools"
)

// ==================== diagnose_code 工具 ====================

// DiagnoseCodeTool 单文件代码诊断工具
// 通过LSP语言服务器获取文件的代码诊断信息
type DiagnoseCodeTool struct {
	manager *LSPManager
	workDir string
}

// NewDiagnoseCodeTool 创建单文件诊断工具
func NewDiagnoseCodeTool(manager *LSPManager, workDir string) *DiagnoseCodeTool {
	return &DiagnoseCodeTool{
		manager: manager,
		workDir: workDir,
	}
}

// SetWorkDir 设置工作目录（用于路径范围校验）
func (t *DiagnoseCodeTool) SetWorkDir(workDir string) {
	t.workDir = workDir
}

func (t *DiagnoseCodeTool) Name() string {
	return "diagnose_code"
}

func (t *DiagnoseCodeTool) Description() string {
	return "对指定代码文件进行LSP诊断分析。支持Go、TypeScript/JavaScript、Python三种语言。" +
		"返回文件中所有错误、警告和信息级别的诊断信息，包括行号、列号和详细描述。" +
		"当对应语言服务器不可用时，会返回降级信息。" +
		"路径必须是绝对路径。"
}

func (t *DiagnoseCodeTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{
		"path": {
			Type:        "string",
			Description: "要诊断的代码文件的绝对路径",
			Required:    true,
		},
		"content": {
			Type:        "string",
			Description: "可选的文件内容。如果提供，将用此内容替代磁盘文件内容进行诊断（用于诊断未保存的修改）",
		},
	}, []string{"path"})
}

func (t *DiagnoseCodeTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	// 解析参数
	pathVal, ok := params["path"]
	if !ok {
		return tools.NewErrorResult("missing required parameter: path"), nil
	}
	path, ok := pathVal.(string)
	if !ok {
		return tools.NewErrorResult("parameter 'path' must be a string"), nil
	}

	// 验证绝对路径
	if !filepath.IsAbs(path) {
		return tools.NewErrorResult(fmt.Sprintf("path must be absolute: %s", path)), nil
	}

	// 检查LSP是否启用
	if t.manager == nil || !t.manager.IsEnabled() {
		return tools.NewSuccessResult(t.buildDegradedResult(path, "LSP功能未启用")), nil
	}

	// 识别语言
	language := LanguageFromPath(path)
	if language == "" {
		return tools.NewSuccessResult(t.buildDegradedResult(path, "不支持的文件类型，支持: .go, .ts, .tsx, .js, .jsx, .py, .pyw")), nil
	}

	// 检查文件是否存在
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return tools.NewErrorResult(fmt.Sprintf("file does not exist: %s", path)), nil
	}

	// 获取诊断
	result, err := t.diagnose(ctx, path, language, params)
	if err != nil {
		return tools.NewSuccessResult(t.buildDegradedResult(path, fmt.Sprintf("诊断失败: %v", err))), nil
	}

	return tools.NewSuccessResult(result), nil
}

// diagnose 执行诊断逻辑
func (t *DiagnoseCodeTool) diagnose(ctx context.Context, path, language string, params map[string]interface{}) (map[string]interface{}, error) {
	// 设置超时上下文
	diagCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	diags, err := t.manager.GetDiagnosticsForLanguage(diagCtx, path, language)
	if err != nil {
		return nil, err
	}

	// 构建结果
	result := map[string]interface{}{
		"file":        path,
		"language":    LanguageDisplayName(language),
		"lsp_server":  language,
		"diagnostics": t.convertDiagnostics(diags),
		"summary":     t.buildSummary(diags),
	}

	return result, nil
}

// convertDiagnostics 转换诊断格式
func (t *DiagnoseCodeTool) convertDiagnostics(diags []Diagnostic) []map[string]interface{} {
	result := make([]map[string]interface{}, 0, len(diags))
	for _, d := range diags {
		item := map[string]interface{}{
			"severity": d.SeverityString(),
			"line":     d.Range.Start.Line + 1,      // 转为1-based
			"column":   d.Range.Start.Character + 1, // 转为1-based
			"message":  d.Message,
			"source":   d.Source,
		}
		if d.Code != nil {
			item["code"] = fmt.Sprintf("%v", d.Code)
		}
		result = append(result, item)
	}
	return result
}

// buildSummary 构建诊断摘要
func (t *DiagnoseCodeTool) buildSummary(diags []Diagnostic) map[string]interface{} {
	errors, warnings, info, hints := 0, 0, 0, 0
	for _, d := range diags {
		switch d.Severity {
		case 1:
			errors++
		case 2:
			warnings++
		case 3:
			info++
		case 4:
			hints++
		}
	}

	return map[string]interface{}{
		"total":    len(diags),
		"errors":   errors,
		"warnings": warnings,
		"info":     info,
		"hints":    hints,
	}
}

// buildDegradedResult 构建降级结果（语言服务器不可用时）
func (t *DiagnoseCodeTool) buildDegradedResult(path, reason string) map[string]interface{} {
	language := LanguageFromPath(path)
	return map[string]interface{}{
		"file":           path,
		"language":       LanguageDisplayName(language),
		"diagnostics":    []map[string]interface{}{},
		"summary":        map[string]interface{}{"total": 0, "errors": 0, "warnings": 0},
		"degraded":       true,
		"degrade_reason": reason,
		"lsp_server":     language,
	}
}

// ==================== get_diagnostics 工具 ====================

// GetDiagnosticsTool 多文件诊断工具
// 支持批量获取多个文件的诊断信息
type GetDiagnosticsTool struct {
	manager *LSPManager
	workDir string
}

// NewGetDiagnosticsTool 创建多文件诊断工具
func NewGetDiagnosticsTool(manager *LSPManager, workDir string) *GetDiagnosticsTool {
	return &GetDiagnosticsTool{
		manager: manager,
		workDir: workDir,
	}
}

// SetWorkDir 设置工作目录（用于路径范围校验）
func (t *GetDiagnosticsTool) SetWorkDir(workDir string) {
	t.workDir = workDir
}

func (t *GetDiagnosticsTool) Name() string {
	return "get_diagnostics"
}

func (t *GetDiagnosticsTool) Description() string {
	return "批量获取多个代码文件的LSP诊断信息。支持Go、TypeScript/JavaScript、Python三种语言。" +
		"如果不指定paths参数，将对工作目录下所有受支持的代码文件进行诊断。" +
		"返回每个文件的诊断信息和整体统计摘要。" +
		"路径必须是绝对路径。"
}

func (t *GetDiagnosticsTool) Parameters() map[string]interface{} {
	return tools.BuildSchema(map[string]*tools.ToolParameter{
		"paths": {
			Type:        "array",
			Description: "要诊断的文件绝对路径列表。不指定时自动扫描工作目录下所有支持的代码文件",
			Items: &tools.ToolParameter{
				Type: "string",
			},
		},
	}, []string{})
}

func (t *GetDiagnosticsTool) Execute(ctx context.Context, params map[string]interface{}) (*tools.ToolResult, error) {
	// 检查LSP是否启用
	if t.manager == nil || !t.manager.IsEnabled() {
		return tools.NewSuccessResult(map[string]interface{}{
			"files":          []interface{}{},
			"degraded":       true,
			"degrade_reason": "LSP功能未启用",
		}), nil
	}

	// 解析paths参数
	paths := t.resolvePaths(params)
	if len(paths) == 0 {
		return tools.NewSuccessResult(map[string]interface{}{
			"files":   []interface{}{},
			"summary": map[string]interface{}{"total_files": 0},
			"message": "没有找到需要诊断的文件",
		}), nil
	}

	// 对每个文件执行诊断
	type fileResult struct {
		file     string
		diags    []Diagnostic
		language string
		err      error
	}

	results := make([]fileResult, 0, len(paths))
	totalErrors, totalWarnings := 0, 0

	for _, p := range paths {
		language := LanguageFromPath(p)
		if language == "" {
			continue
		}

		diagCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		diags, err := t.manager.GetDiagnosticsForLanguage(diagCtx, p, language)
		cancel()

		if err != nil {
			results = append(results, fileResult{file: p, language: language, err: err})
			continue
		}

		for _, d := range diags {
			if d.Severity == 1 {
				totalErrors++
			} else if d.Severity == 2 {
				totalWarnings++
			}
		}

		results = append(results, fileResult{file: p, language: language, diags: diags})
	}

	// 构建输出
	diagTool := &DiagnoseCodeTool{manager: t.manager, workDir: t.workDir}
	fileOutputs := make([]map[string]interface{}, 0, len(results))
	for _, r := range results {
		if r.err != nil {
			fileOutputs = append(fileOutputs, diagTool.buildDegradedResult(r.file, r.err.Error()))
			continue
		}
		fileOutputs = append(fileOutputs, map[string]interface{}{
			"file":        r.file,
			"language":    LanguageDisplayName(r.language),
			"diagnostics": diagTool.convertDiagnostics(r.diags),
			"summary":     diagTool.buildSummary(r.diags),
			"lsp_server":  r.language,
		})
	}

	return tools.NewSuccessResult(map[string]interface{}{
		"files":   fileOutputs,
		"summary": map[string]interface{}{"total_files": len(fileOutputs), "total_errors": totalErrors, "total_warnings": totalWarnings},
	}), nil
}

// resolvePaths 解析要诊断的文件路径列表
func (t *GetDiagnosticsTool) resolvePaths(params map[string]interface{}) []string {
	// 如果指定了paths参数，直接使用
	if pathsVal, ok := params["paths"]; ok {
		switch pv := pathsVal.(type) {
		case []string:
			return t.filterSupported(pv)
		case []interface{}:
			paths := make([]string, 0, len(pv))
			for _, v := range pv {
				if s, ok := v.(string); ok {
					paths = append(paths, s)
				}
			}
			return t.filterSupported(paths)
		}
	}

	// 未指定paths时，扫描工作目录（限制最大文件数避免过慢）
	if t.workDir == "" {
		return nil
	}

	var paths []string
	filepath.WalkDir(t.workDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || len(paths) >= 50 {
			return nil
		}
		// 跳过隐藏目录和常见忽略目录
		name := d.Name()
		if d.IsDir() && (strings.HasPrefix(name, ".") || name == "node_modules" || name == "vendor" || name == "__pycache__") {
			return filepath.SkipDir
		}
		if !d.IsDir() && IsSupportedFile(path) {
			paths = append(paths, path)
		}
		return nil
	})

	return paths
}

// filterSupported 过滤出受LSP支持的文件
func (t *GetDiagnosticsTool) filterSupported(paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if filepath.IsAbs(p) && IsSupportedFile(p) {
			result = append(result, p)
		}
	}
	return result
}

// ==================== 工具注册函数 ====================

// RegisterLSPTools 注册LSP诊断工具到工具注册中心
func RegisterLSPTools(registry *tools.ToolRegistry, manager *LSPManager, workDir string) error {
	diagTool := NewDiagnoseCodeTool(manager, workDir)
	if err := registry.RegisterTool(diagTool); err != nil {
		return fmt.Errorf("failed to register diagnose_code tool: %w", err)
	}

	batchTool := NewGetDiagnosticsTool(manager, workDir)
	if err := registry.RegisterTool(batchTool); err != nil {
		return fmt.Errorf("failed to register get_diagnostics tool: %w", err)
	}

	return nil
}
