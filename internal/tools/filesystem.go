package tools

import (
	"github.com/Attect/MukaAI/internal/tools/syntax"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// ==================== 路径校验辅助函数 ====================

// validatePath 验证请求的路径是否在允许的工作目录范围内
// workDir为空时不做限制（向后兼容，与命令白名单设计一致）
// 同时检查目标路径是否为系统敏感路径（拒绝/警告）
// 返回清理后的绝对路径、敏感路径警告列表，或错误
func validatePath(requestedPath, workDir string) (string, []string, error) {
	// 清理路径（处理 .. 和 . 等）
	cleaned := filepath.Clean(requestedPath)

	// workDir为空时不做限制（向后兼容）
	if workDir == "" {
		// 即使不限制工作目录，仍然执行敏感路径检查
		return checkSensitiveAndReturn(cleaned)
	}

	absWorkDir := filepath.Clean(workDir)

	// 如果请求的路径不是绝对路径，尝试相对于workDir解析
	// 区分真正的相对路径（如 subdir/file.txt）和Unix风格绝对路径（如 /tmp/xxx）
	// 在Windows上，/tmp/xxx 不是绝对路径，但它是Unix绝对路径，不应被接受
	if !filepath.IsAbs(cleaned) {
		// 检查是否是Unix风格的绝对路径（以/开头）
		// filepath.Clean在Windows上不会将以/开头的路径视为绝对路径
		if strings.HasPrefix(requestedPath, "/") {
			// 这是Unix风格的绝对路径，在Windows上无法使用，直接拒绝
			return "", nil, fmt.Errorf("路径 '%s' 超出允许的工作目录范围 '%s'", requestedPath, workDir)
		}
		// 真正的相对路径，拼接workDir
		cleaned = filepath.Join(absWorkDir, cleaned)
	}

	// 解析符号链接获取真实路径，防止通过符号链接逃逸
	realPath, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		// 如果路径不存在（如write_file目标），使用清理后的路径
		realPath = cleaned
	}
	realWorkDir, err := filepath.EvalSymlinks(absWorkDir)
	if err != nil {
		realWorkDir = absWorkDir
	}

	// 使用filepath.Rel进行可靠的路径范围校验
	// rel为空字符串表示两路径相同，rel不以".."开头表示在范围内
	rel, err := filepath.Rel(realWorkDir, realPath)
	if err != nil {
		return "", nil, fmt.Errorf("路径 '%s' 超出允许的工作目录范围 '%s'", requestedPath, workDir)
	}

	// filepath.Rel在Windows上可能返回以\开头的路径（表示不同卷）
	// 也可能返回".."开头的路径（表示在工作目录之外）
	relNorm := filepath.ToSlash(rel)
	if strings.HasPrefix(relNorm, "..") || strings.HasPrefix(relNorm, "/") {
		return "", nil, fmt.Errorf("路径 '%s' 超出允许的工作目录范围 '%s'", requestedPath, workDir)
	}

	// Windows上路径大小写不敏感，额外使用EqualFold校验
	if runtime.GOOS == "windows" {
		pathLower := strings.ToLower(realPath)
		workDirLower := strings.ToLower(realWorkDir)
		if !strings.HasPrefix(pathLower+string(filepath.Separator), workDirLower+string(filepath.Separator)) &&
			pathLower != workDirLower {
			return "", nil, fmt.Errorf("路径 '%s' 超出允许的工作目录范围 '%s'", requestedPath, workDir)
		}
	}

	return checkSensitiveAndReturn(cleaned)
}

// checkSensitiveAndReturn 对清理后的路径执行敏感路径检查，返回路径和可能的警告
func checkSensitiveAndReturn(cleanedPath string) (string, []string, error) {
	var warnings []string
	checkResult := CheckSensitivePath(cleanedPath)
	switch checkResult.Level {
	case PathCheckDeny:
		return "", nil, fmt.Errorf("敏感路径拒绝: %s", checkResult.Reason)
	case PathCheckWarn:
		warnings = append(warnings, checkResult.Reason)
	}
	return cleanedPath, warnings, nil
}

// ==================== ReadFile Tool ====================

// ReadFileTool 读取文件内容工具
type ReadFileTool struct {
	workDir string
}

func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{}
}

func NewReadFileToolWithWorkDir(workDir string) *ReadFileTool {
	return &ReadFileTool{workDir: workDir}
}

func (t *ReadFileTool) Name() string {
	return "read_file"
}

func (t *ReadFileTool) Description() string {
	return "读取指定路径的文件内容。支持读取文本文件，返回文件内容字符串。路径必须是绝对路径。"
}

func (t *ReadFileTool) Parameters() map[string]interface{} {
	return BuildSchema(map[string]*ToolParameter{
		"path": {
			Type:        "string",
			Description: "文件的绝对路径",
			Required:    true,
		},
		"encoding": {
			Type:        "string",
			Description: "文件编码，默认为utf-8",
			Default:     "utf-8",
		},
	}, []string{"path"})
}

func (t *ReadFileTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	pathVal, ok := params["path"]
	if !ok {
		return NewErrorResult("missing required parameter: path"), nil
	}

	path, ok := pathVal.(string)
	if !ok {
		return NewErrorResult("parameter 'path' must be a string"), nil
	}

	// 验证是否为绝对路径
	if !filepath.IsAbs(path) {
		return NewErrorResult(fmt.Sprintf("path must be absolute: %s", path)), nil
	}

	// 验证路径在工作目录范围内（含敏感路径检查）
	validatedPath, warnings, err := validatePath(path, t.workDir)
	if err != nil {
		return NewErrorResult(err.Error()), nil
	}

	// 检查文件是否存在
	info, err := os.Stat(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorResult(fmt.Sprintf("file does not exist: %s", validatedPath)), nil
		}
		return NewErrorResultWithError(fmt.Sprintf("failed to stat file: %s", validatedPath), err), nil
	}

	if info.IsDir() {
		return NewErrorResult(fmt.Sprintf("path is a directory, not a file: %s", validatedPath)), nil
	}

	// 读取文件内容
	content, err := os.ReadFile(validatedPath)
	if err != nil {
		return NewErrorResultWithError(fmt.Sprintf("failed to read file: %s", validatedPath), err), nil
	}

	result := map[string]interface{}{
		"path":    validatedPath,
		"content": string(content),
		"size":    len(content),
	}
	if len(warnings) > 0 {
		result["warnings"] = warnings
	}
	return NewSuccessResult(result), nil
}

// ==================== WriteFile Tool ====================

// WriteFileTool 写入文件工具
type WriteFileTool struct {
	workDir    string
	dispatcher *syntax.Dispatcher
}

func NewWriteFileTool() *WriteFileTool {
	return &WriteFileTool{}
}

func NewWriteFileToolWithWorkDir(workDir string) *WriteFileTool {
	return &WriteFileTool{workDir: workDir}
}

// NewWriteFileToolWithWorkDirAndDispatcher 创建带语法检查调度器的写入文件工具
func NewWriteFileToolWithWorkDirAndDispatcher(workDir string, dispatcher *syntax.Dispatcher) *WriteFileTool {
	return &WriteFileTool{workDir: workDir, dispatcher: dispatcher}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "将内容写入指定路径的文件。如果文件不存在则创建，如果存在则覆盖。路径必须是绝对路径。会自动创建所需的父目录。" +
		"写入完成后会自动根据文件扩展名进行语法检查（支持JSON/YAML/XML/HTML/Go/JS/TS/Python/Java/Kotlin/Rust等），检查结果会附加在返回数据中。如果发现语法错误，会在返回消息中提示，请根据错误信息修正文件。" +
		"\n**重要：必须使用此工具进行文件创建/写入操作，不应通过execute_command等命令行方式绕过文件操作工具。**"
}

func (t *WriteFileTool) Parameters() map[string]interface{} {
	return BuildSchema(map[string]*ToolParameter{
		"path": {
			Type:        "string",
			Description: "文件的绝对路径",
			Required:    true,
		},
		"content": {
			Type:        "string",
			Description: "要写入的文件内容",
			Required:    true,
		},
	}, []string{"path", "content"})
}

func (t *WriteFileTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	pathVal, ok := params["path"]
	if !ok {
		return NewErrorResult("missing required parameter: path"), nil
	}

	path, ok := pathVal.(string)
	if !ok {
		return NewErrorResult("parameter 'path' must be a string"), nil
	}

	contentVal, ok := params["content"]
	if !ok {
		return NewErrorResult("missing required parameter: content"), nil
	}

	content, ok := contentVal.(string)
	if !ok {
		return NewErrorResult("parameter 'content' must be a string"), nil
	}

	// 验证是否为绝对路径
	if !filepath.IsAbs(path) {
		return NewErrorResult(fmt.Sprintf("path must be absolute: %s", path)), nil
	}

	// 验证路径在工作目录范围内（含敏感路径检查）
	validatedPath, warnings, err := validatePath(path, t.workDir)
	if err != nil {
		return NewErrorResult(err.Error()), nil
	}

	// 创建父目录（如果不存在）
	// 父目录路径由validatedPath派生，不需要额外校验（必然在workDir内）
	dir := filepath.Dir(validatedPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewErrorResultWithError(fmt.Sprintf("failed to create parent directory: %s", dir), err), nil
	}

	// 写入文件
	if err := os.WriteFile(validatedPath, []byte(content), 0644); err != nil {
		return NewErrorResultWithError(fmt.Sprintf("failed to write file: %s", validatedPath), err), nil
	}

	result := map[string]interface{}{
		"path":        validatedPath,
		"bytes_write": len(content),
		"message":     "file written successfully",
	}
	if len(warnings) > 0 {
		result["warnings"] = warnings
	}

	// 写入成功后执行语法检查（仅当dispatcher可用时）
	if t.dispatcher != nil {
		syntaxResult := t.dispatcher.Check(content, validatedPath, int64(len(content)))
		if syntaxResult != nil {
			result["syntax_check"] = syntaxResult
			// 有语法错误时修改消息
			if syntaxResult.HasErrors {
				result["message"] = "file written successfully, but syntax errors detected. Please fix them immediately by using edit_file or write_file tool."
			} else if syntaxResult.Degraded {
				result["message"] = fmt.Sprintf("file written successfully (syntax check skipped: %s)", syntaxResult.DegradedReason)
			}
		}
	}

	return NewSuccessResult(result), nil
}

// ==================== EditFile Tool ====================

// EditFileTool 编辑文件工具
// 支持两种模式：line_replace（行号替换）和 string_replace（字符串替换）
// 编辑完成后自动进行语法检查
type EditFileTool struct {
	workDir    string
	dispatcher *syntax.Dispatcher
}

// NewEditFileToolWithWorkDir 创建编辑文件工具
func NewEditFileToolWithWorkDir(workDir string) *EditFileTool {
	return &EditFileTool{workDir: workDir}
}

// NewEditFileToolWithWorkDirAndDispatcher 创建带语法检查调度器的编辑文件工具
func NewEditFileToolWithWorkDirAndDispatcher(workDir string, dispatcher *syntax.Dispatcher) *EditFileTool {
	return &EditFileTool{workDir: workDir, dispatcher: dispatcher}
}

func (t *EditFileTool) Name() string {
	return "edit_file"
}

func (t *EditFileTool) Description() string {
	return "编辑指定文件的内容。支持两种编辑模式：" +
		"1. line_replace模式：通过指定起始行(start_line)和结束行(end_line)替换指定范围的行。" +
		"2. string_replace模式：通过指定原始字符串(old_string)和新字符串(new_string)进行精确替换。设置replace_all=true可替换所有匹配项。" +
		"\n编辑完成后会自动进行语法检查，如果发现错误会在返回结果中提示。" +
		"\n**重要：必须使用此工具或write_file工具进行文件编辑操作，不应通过execute_command等命令行方式绕过文件操作工具。**" +
		"\n路径必须是绝对路径。"
}

func (t *EditFileTool) Parameters() map[string]interface{} {
	return BuildSchema(map[string]*ToolParameter{
		"path": {
			Type:        "string",
			Description: "文件的绝对路径",
			Required:    true,
		},
		"mode": {
			Type:        "string",
			Description: "编辑模式：line_replace（行号替换）或 string_replace（字符串替换）",
			Required:    true,
			Enum:        []string{"line_replace", "string_replace"},
		},
		"start_line": {
			Type:        "integer",
			Description: "起始行号（1-based），line_replace模式必需",
		},
		"end_line": {
			Type:        "integer",
			Description: "结束行号（1-based，包含），line_replace模式必需",
		},
		"old_string": {
			Type:        "string",
			Description: "被替换的原始字符串，string_replace模式必需",
		},
		"new_string": {
			Type:        "string",
			Description: "替换后的新字符串，默认为空（即删除操作）",
		},
		"replace_all": {
			Type:        "boolean",
			Description: "string_replace模式下是否替换所有匹配项，默认false",
			Default:     false,
		},
	}, []string{"path", "mode"})
}

func (t *EditFileTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	// 解析必需参数
	pathVal, ok := params["path"]
	if !ok {
		return NewErrorResult("missing required parameter: path"), nil
	}
	path, ok := pathVal.(string)
	if !ok {
		return NewErrorResult("parameter 'path' must be a string"), nil
	}

	modeVal, ok := params["mode"]
	if !ok {
		return NewErrorResult("missing required parameter: mode"), nil
	}
	mode, ok := modeVal.(string)
	if !ok {
		return NewErrorResult("parameter 'mode' must be a string"), nil
	}

	// 验证模式
	if mode != "line_replace" && mode != "string_replace" {
		return NewErrorResult(fmt.Sprintf("invalid mode '%s', must be 'line_replace' or 'string_replace'", mode)), nil
	}

	// 验证路径
	if !filepath.IsAbs(path) {
		return NewErrorResult(fmt.Sprintf("path must be absolute: %s", path)), nil
	}

	validatedPath, warnings, err := validatePath(path, t.workDir)
	if err != nil {
		return NewErrorResult(err.Error()), nil
	}

	info, err := os.Stat(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorResult(fmt.Sprintf("file does not exist: %s", validatedPath)), nil
		}
		return NewErrorResultWithError(fmt.Sprintf("failed to stat file: %s", validatedPath), err), nil
	}
	if info.IsDir() {
		return NewErrorResult(fmt.Sprintf("path is a directory, not a file: %s", validatedPath)), nil
	}

	// 读取文件内容
	contentBytes, err := os.ReadFile(validatedPath)
	if err != nil {
		return NewErrorResultWithError(fmt.Sprintf("failed to read file: %s", validatedPath), err), nil
	}
	content := string(contentBytes)

	// 解析new_string（默认为空字符串）
	newString := ""
	if nsVal, ok := params["new_string"]; ok {
		if ns, ok := nsVal.(string); ok {
			newString = ns
		}
	}

	var newContent string
	var replacements int
	var linesAffected []int

	switch mode {
	case "line_replace":
		newContent, linesAffected, err = t.executeLineReplace(content, params)
		if err != nil {
			return NewErrorResult(err.Error()), nil
		}
		replacements = 1

	case "string_replace":
		// 解析old_string
		oldStringVal, ok := params["old_string"]
		if !ok {
			return NewErrorResult("missing required parameter: old_string for string_replace mode"), nil
		}
		oldString, ok := oldStringVal.(string)
		if !ok {
			return NewErrorResult("parameter 'old_string' must be a string"), nil
		}
		if oldString == "" {
			return NewErrorResult("parameter 'old_string' must not be empty"), nil
		}

		// 解析replace_all
		replaceAll := false
		if raVal, ok := params["replace_all"]; ok {
			if ra, ok := raVal.(bool); ok {
				replaceAll = ra
			}
		}

		newContent, replacements, linesAffected, err = t.executeStringReplace(content, oldString, newString, replaceAll)
		if err != nil {
			return NewErrorResult(err.Error()), nil
		}
	}

	// 写入修改后的内容
	if err := os.WriteFile(validatedPath, []byte(newContent), 0644); err != nil {
		return NewErrorResultWithError(fmt.Sprintf("failed to write edited file: %s", validatedPath), err), nil
	}

	// 构建返回结果
	result := map[string]interface{}{
		"path":              validatedPath,
		"mode":              mode,
		"replacements_made": replacements,
		"bytes_written":     len(newContent),
		"message":           "file edited successfully",
	}
	if len(linesAffected) > 0 {
		result["lines_affected"] = linesAffected
	}
	if len(warnings) > 0 {
		result["warnings"] = warnings
	}

	// 编辑完成后执行语法检查
	if t.dispatcher != nil {
		syntaxResult := t.dispatcher.Check(newContent, validatedPath, int64(len(newContent)))
		if syntaxResult != nil {
			result["syntax_check"] = syntaxResult
			if syntaxResult.HasErrors {
				result["message"] = "file edited successfully, but syntax errors detected. Please fix them immediately by using edit_file or write_file tool."
			} else if syntaxResult.Degraded {
				result["message"] = fmt.Sprintf("file edited successfully (syntax check skipped: %s)", syntaxResult.DegradedReason)
			}
		}
	}

	return NewSuccessResult(result), nil
}

// executeLineReplace 执行行号替换模式
func (t *EditFileTool) executeLineReplace(content string, params map[string]interface{}) (string, []int, error) {
	// 解析start_line
	startLineVal, ok := params["start_line"]
	if !ok {
		return "", nil, fmt.Errorf("missing required parameter: start_line for line_replace mode")
	}
	startLine, ok := startLineVal.(float64)
	if !ok {
		return "", nil, fmt.Errorf("parameter 'start_line' must be a number")
	}

	// 解析end_line
	endLineVal, ok := params["end_line"]
	if !ok {
		return "", nil, fmt.Errorf("missing required parameter: end_line for line_replace mode")
	}
	endLine, ok := endLineVal.(float64)
	if !ok {
		return "", nil, fmt.Errorf("parameter 'end_line' must be a number")
	}

	// 验证行号
	if startLine < 1 {
		return "", nil, fmt.Errorf("start_line must be >= 1")
	}
	if startLine > endLine {
		return "", nil, fmt.Errorf("start_line must be <= end_line")
	}

	// 解析new_string
	newString := ""
	if nsVal, ok := params["new_string"]; ok {
		if ns, ok := nsVal.(string); ok {
			newString = ns
		}
	}

	// 分割行
	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// 如果内容为空且只有一行空内容
	if content == "" {
		lines = []string{}
		totalLines = 0
	}

	if totalLines == 0 {
		return "", nil, fmt.Errorf("cannot perform line_replace on empty file")
	}

	sLine := int(startLine)
	eLine := int(endLine)

	if sLine > totalLines {
		return "", nil, fmt.Errorf("start_line exceeds file line count (file has %d lines, start_line=%d)", totalLines, sLine)
	}
	if eLine > totalLines {
		return "", nil, fmt.Errorf("end_line exceeds file line count (file has %d lines, end_line=%d)", totalLines, eLine)
	}

	// 构建受影响行号列表
	linesAffected := make([]int, 0, eLine-sLine+1)
	for i := sLine; i <= eLine; i++ {
		linesAffected = append(linesAffected, i)
	}

	// 构建新内容
	var newLines []string
	// 保留 start_line 之前的行
	newLines = append(newLines, lines[:sLine-1]...)
	// 插入新内容
	if newString != "" {
		newLines = append(newLines, strings.Split(newString, "\n")...)
	}
	// 保留 end_line 之后的行
	if eLine < totalLines {
		newLines = append(newLines, lines[eLine:]...)
	}

	return strings.Join(newLines, "\n"), linesAffected, nil
}

// executeStringReplace 执行字符串替换模式
func (t *EditFileTool) executeStringReplace(content string, oldString, newString string, replaceAll bool) (string, int, []int, error) {
	// 统计匹配次数
	count := strings.Count(content, oldString)

	if count == 0 {
		return "", 0, nil, fmt.Errorf("old_string not found in file")
	}

	if count > 1 && !replaceAll {
		return "", 0, nil, fmt.Errorf("found %d matches for old_string. Provide more context or use replace_all=true", count)
	}

	var newContent string
	replacements := count

	if replaceAll {
		newContent = strings.ReplaceAll(content, oldString, newString)
	} else {
		newContent = strings.Replace(content, oldString, newString, 1)
	}

	// 计算受影响的行号
	linesAffected := findAffectedLines(content, oldString, replaceAll, count)

	return newContent, replacements, linesAffected, nil
}

// findAffectedLines 查找受替换影响的行号
func findAffectedLines(content, oldString string, replaceAll bool, count int) []int {
	var affected []int

	// 使用搜索偏移量在内容中查找每次出现的位置
	// 根据位置计算所在行号
	searchStart := 0
	for {
		idx := strings.Index(content[searchStart:], oldString)
		if idx < 0 {
			break
		}

		// 计算绝对位置
		absPos := searchStart + idx

		// 计算行号：统计绝对位置之前的换行符数量
		lineNum := 1
		for i := 0; i < absPos && i < len(content); i++ {
			if content[i] == '\n' {
				lineNum++
			}
		}
		affected = append(affected, lineNum)

		if !replaceAll {
			break
		}

		searchStart = absPos + len(oldString)
	}

	return affected
}

// ==================== ListDirectory Tool ====================

// FileInfo 文件信息结构
type FileInfo struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	IsDir   bool   `json:"is_dir"`
	Size    int64  `json:"size"`
	Mode    string `json:"mode"`
	ModTime string `json:"mod_time"`
}

// ListDirectoryTool 列出目录内容工具
type ListDirectoryTool struct {
	workDir string
}

func NewListDirectoryTool() *ListDirectoryTool {
	return &ListDirectoryTool{}
}

func NewListDirectoryToolWithWorkDir(workDir string) *ListDirectoryTool {
	return &ListDirectoryTool{workDir: workDir}
}

func (t *ListDirectoryTool) Name() string {
	return "list_directory"
}

func (t *ListDirectoryTool) Description() string {
	return "列出指定目录的内容。返回目录中所有文件和子目录的列表，包含名称、类型、大小等信息。路径必须是绝对路径。"
}

func (t *ListDirectoryTool) Parameters() map[string]interface{} {
	return BuildSchema(map[string]*ToolParameter{
		"path": {
			Type:        "string",
			Description: "目录的绝对路径",
			Required:    true,
		},
		"recursive": {
			Type:        "boolean",
			Description: "是否递归列出子目录内容，默认为false",
			Default:     false,
		},
	}, []string{"path"})
}

func (t *ListDirectoryTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	pathVal, ok := params["path"]
	if !ok {
		return NewErrorResult("missing required parameter: path"), nil
	}

	path, ok := pathVal.(string)
	if !ok {
		return NewErrorResult("parameter 'path' must be a string"), nil
	}

	// 验证是否为绝对路径
	if !filepath.IsAbs(path) {
		return NewErrorResult(fmt.Sprintf("path must be absolute: %s", path)), nil
	}

	// 验证路径在工作目录范围内（含敏感路径检查）
	validatedPath, _, err := validatePath(path, t.workDir)
	if err != nil {
		return NewErrorResult(err.Error()), nil
	}

	// 检查目录是否存在
	info, err := os.Stat(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorResult(fmt.Sprintf("directory does not exist: %s", validatedPath)), nil
		}
		return NewErrorResultWithError(fmt.Sprintf("failed to stat directory: %s", validatedPath), err), nil
	}

	if !info.IsDir() {
		return NewErrorResult(fmt.Sprintf("path is not a directory: %s", validatedPath)), nil
	}

	// 获取递归参数
	recursive := false
	if recVal, ok := params["recursive"]; ok {
		if rec, ok := recVal.(bool); ok {
			recursive = rec
		}
	}

	var files []FileInfo

	if recursive {
		// 递归遍历
		err = filepath.WalkDir(validatedPath, func(walkPath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// 跳过根目录本身
			if walkPath == validatedPath {
				return nil
			}

			info, err := d.Info()
			if err != nil {
				return err
			}

			files = append(files, FileInfo{
				Name:    d.Name(),
				Path:    walkPath,
				IsDir:   d.IsDir(),
				Size:    info.Size(),
				Mode:    info.Mode().String(),
				ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			})
			return nil
		})
		if err != nil {
			return NewErrorResultWithError(fmt.Sprintf("failed to walk directory: %s", validatedPath), err), nil
		}
	} else {
		// 非递归，只列出直接内容
		entries, err := os.ReadDir(validatedPath)
		if err != nil {
			return NewErrorResultWithError(fmt.Sprintf("failed to read directory: %s", validatedPath), err), nil
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			files = append(files, FileInfo{
				Name:    entry.Name(),
				Path:    filepath.Join(validatedPath, entry.Name()),
				IsDir:   entry.IsDir(),
				Size:    info.Size(),
				Mode:    info.Mode().String(),
				ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
	}

	return NewSuccessResult(map[string]interface{}{
		"path":      validatedPath,
		"count":     len(files),
		"entries":   files,
		"recursive": recursive,
	}), nil
}

// ==================== DeleteFile Tool ====================

// DeleteFileTool 删除文件工具
type DeleteFileTool struct {
	workDir string
}

func NewDeleteFileTool() *DeleteFileTool {
	return &DeleteFileTool{}
}

func NewDeleteFileToolWithWorkDir(workDir string) *DeleteFileTool {
	return &DeleteFileTool{workDir: workDir}
}

func (t *DeleteFileTool) Name() string {
	return "delete_file"
}

func (t *DeleteFileTool) Description() string {
	return "删除指定路径的文件或目录。如果是目录，可以选择是否递归删除。路径必须是绝对路径。删除操作不可恢复，请谨慎使用。"
}

func (t *DeleteFileTool) Parameters() map[string]interface{} {
	return BuildSchema(map[string]*ToolParameter{
		"path": {
			Type:        "string",
			Description: "要删除的文件或目录的绝对路径",
			Required:    true,
		},
		"recursive": {
			Type:        "boolean",
			Description: "如果是目录，是否递归删除所有内容，默认为false",
			Default:     false,
		},
	}, []string{"path"})
}

func (t *DeleteFileTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	pathVal, ok := params["path"]
	if !ok {
		return NewErrorResult("missing required parameter: path"), nil
	}

	path, ok := pathVal.(string)
	if !ok {
		return NewErrorResult("parameter 'path' must be a string"), nil
	}

	// 验证是否为绝对路径
	if !filepath.IsAbs(path) {
		return NewErrorResult(fmt.Sprintf("path must be absolute: %s", path)), nil
	}

	// 验证路径在工作目录范围内（含敏感路径检查）
	validatedPath, _, err := validatePath(path, t.workDir)
	if err != nil {
		return NewErrorResult(err.Error()), nil
	}

	// 检查路径是否存在
	info, err := os.Stat(validatedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorResult(fmt.Sprintf("path does not exist: %s", validatedPath)), nil
		}
		return NewErrorResultWithError(fmt.Sprintf("failed to stat path: %s", validatedPath), err), nil
	}

	// 获取递归参数
	recursive := false
	if recVal, ok := params["recursive"]; ok {
		if rec, ok := recVal.(bool); ok {
			recursive = rec
		}
	}

	// 执行删除
	if info.IsDir() {
		if recursive {
			if err := os.RemoveAll(validatedPath); err != nil {
				return NewErrorResultWithError(fmt.Sprintf("failed to remove directory recursively: %s", validatedPath), err), nil
			}
		} else {
			if err := os.Remove(validatedPath); err != nil {
				// 如果目录非空，提示使用recursive
				if strings.Contains(err.Error(), "directory not empty") {
					return NewErrorResult(fmt.Sprintf("directory is not empty, use recursive=true to delete: %s", validatedPath)), nil
				}
				return NewErrorResultWithError(fmt.Sprintf("failed to remove directory: %s", validatedPath), err), nil
			}
		}
	} else {
		if err := os.Remove(validatedPath); err != nil {
			return NewErrorResultWithError(fmt.Sprintf("failed to remove file: %s", validatedPath), err), nil
		}
	}

	return NewSuccessResult(map[string]interface{}{
		"path":      validatedPath,
		"was_dir":   info.IsDir(),
		"recursive": recursive,
		"message":   "deleted successfully",
	}), nil
}

// ==================== CreateDirectory Tool ====================

// CreateDirectoryTool 创建目录工具
type CreateDirectoryTool struct {
	workDir string
}

func NewCreateDirectoryTool() *CreateDirectoryTool {
	return &CreateDirectoryTool{}
}

func NewCreateDirectoryToolWithWorkDir(workDir string) *CreateDirectoryTool {
	return &CreateDirectoryTool{workDir: workDir}
}

func (t *CreateDirectoryTool) Name() string {
	return "create_directory"
}

func (t *CreateDirectoryTool) Description() string {
	return "创建指定路径的目录。可以创建多级目录（类似mkdir -p）。路径必须是绝对路径。"
}

func (t *CreateDirectoryTool) Parameters() map[string]interface{} {
	return BuildSchema(map[string]*ToolParameter{
		"path": {
			Type:        "string",
			Description: "要创建的目录的绝对路径",
			Required:    true,
		},
	}, []string{"path"})
}

func (t *CreateDirectoryTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	pathVal, ok := params["path"]
	if !ok {
		return NewErrorResult("missing required parameter: path"), nil
	}

	path, ok := pathVal.(string)
	if !ok {
		return NewErrorResult("parameter 'path' must be a string"), nil
	}

	// 验证是否为绝对路径
	if !filepath.IsAbs(path) {
		return NewErrorResult(fmt.Sprintf("path must be absolute: %s", path)), nil
	}

	// 验证路径在工作目录范围内（含敏感路径检查）
	validatedPath, _, err := validatePath(path, t.workDir)
	if err != nil {
		return NewErrorResult(err.Error()), nil
	}

	// 检查目录是否已存在
	info, err := os.Stat(validatedPath)
	if err == nil {
		if info.IsDir() {
			return NewSuccessResult(map[string]interface{}{
				"path":    validatedPath,
				"created": false,
				"message": "directory already exists",
			}), nil
		}
		return NewErrorResult(fmt.Sprintf("path exists but is not a directory: %s", validatedPath)), nil
	}

	// 创建目录
	if err := os.MkdirAll(validatedPath, 0755); err != nil {
		return NewErrorResultWithError(fmt.Sprintf("failed to create directory: %s", validatedPath), err), nil
	}

	return NewSuccessResult(map[string]interface{}{
		"path":    validatedPath,
		"created": true,
		"message": "directory created successfully",
	}), nil
}

// ==================== 工具注册函数 ====================

// RegisterFilesystemTools 注册所有文件系统工具到指定注册中心（无workDir限制，向后兼容）
func RegisterFilesystemTools(registry *ToolRegistry) error {
	// 初始化语法检查调度器
	dispatcher := syntax.NewDispatcher()
	syntax.RegisterAllCheckers(dispatcher)

	toolList := []Tool{
		NewReadFileTool(),
		NewWriteFileToolWithWorkDirAndDispatcher("", dispatcher),
		NewEditFileToolWithWorkDirAndDispatcher("", dispatcher),
		NewListDirectoryTool(),
		NewDeleteFileTool(),
		NewCreateDirectoryTool(),
	}

	for _, tool := range toolList {
		if err := registry.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
		}
	}
	return nil
}

// RegisterFilesystemToolsWithWorkDir 注册带工作目录限制的文件系统工具
// workDir: 允许操作的工作目录范围，为空时不做限制
func RegisterFilesystemToolsWithWorkDir(registry *ToolRegistry, workDir string) error {
	// 初始化语法检查调度器
	dispatcher := syntax.NewDispatcher()
	syntax.RegisterAllCheckers(dispatcher)

	toolList := []Tool{
		NewReadFileToolWithWorkDir(workDir),
		NewWriteFileToolWithWorkDirAndDispatcher(workDir, dispatcher),
		NewEditFileToolWithWorkDirAndDispatcher(workDir, dispatcher),
		NewListDirectoryToolWithWorkDir(workDir),
		NewDeleteFileToolWithWorkDir(workDir),
		NewCreateDirectoryToolWithWorkDir(workDir),
	}

	for _, tool := range toolList {
		if err := registry.RegisterTool(tool); err != nil {
			return fmt.Errorf("failed to register tool %s: %w", tool.Name(), err)
		}
	}
	return nil
}

// RegisterDefaultFilesystemTools 注册文件系统工具到默认注册中心
func RegisterDefaultFilesystemTools() error {
	return RegisterFilesystemTools(defaultRegistry)
}
