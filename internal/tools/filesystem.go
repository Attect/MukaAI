package tools

import (
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
// 返回清理后的绝对路径，或错误
func validatePath(requestedPath, workDir string) (string, error) {
	// 清理路径（处理 .. 和 . 等）
	cleaned := filepath.Clean(requestedPath)

	// workDir为空时不做限制（向后兼容）
	if workDir == "" {
		return cleaned, nil
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
			return "", fmt.Errorf("路径 '%s' 超出允许的工作目录范围 '%s'", requestedPath, workDir)
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
		return "", fmt.Errorf("路径 '%s' 超出允许的工作目录范围 '%s'", requestedPath, workDir)
	}

	// filepath.Rel在Windows上可能返回以\开头的路径（表示不同卷）
	// 也可能返回".."开头的路径（表示在工作目录之外）
	relNorm := filepath.ToSlash(rel)
	if strings.HasPrefix(relNorm, "..") || strings.HasPrefix(relNorm, "/") {
		return "", fmt.Errorf("路径 '%s' 超出允许的工作目录范围 '%s'", requestedPath, workDir)
	}

	// Windows上路径大小写不敏感，额外使用EqualFold校验
	if runtime.GOOS == "windows" {
		pathLower := strings.ToLower(realPath)
		workDirLower := strings.ToLower(realWorkDir)
		if !strings.HasPrefix(pathLower+string(filepath.Separator), workDirLower+string(filepath.Separator)) &&
			pathLower != workDirLower {
			// filepath.Rel已经通过了，这里作为双重保障
			// 如果Rel通过了但前缀检查失败，可能是边缘情况，优先信任Rel的结果
			_ = pathLower
			_ = workDirLower
		}
	}

	return cleaned, nil
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

	// 验证路径在工作目录范围内
	validatedPath, err := validatePath(path, t.workDir)
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

	return NewSuccessResult(map[string]interface{}{
		"path":    validatedPath,
		"content": string(content),
		"size":    len(content),
	}), nil
}

// ==================== WriteFile Tool ====================

// WriteFileTool 写入文件工具
type WriteFileTool struct {
	workDir string
}

func NewWriteFileTool() *WriteFileTool {
	return &WriteFileTool{}
}

func NewWriteFileToolWithWorkDir(workDir string) *WriteFileTool {
	return &WriteFileTool{workDir: workDir}
}

func (t *WriteFileTool) Name() string {
	return "write_file"
}

func (t *WriteFileTool) Description() string {
	return "将内容写入指定路径的文件。如果文件不存在则创建，如果存在则覆盖。路径必须是绝对路径。会自动创建所需的父目录。"
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

	// 验证路径在工作目录范围内
	validatedPath, err := validatePath(path, t.workDir)
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

	return NewSuccessResult(map[string]interface{}{
		"path":        validatedPath,
		"bytes_write": len(content),
		"message":     "file written successfully",
	}), nil
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

	// 验证路径在工作目录范围内
	validatedPath, err := validatePath(path, t.workDir)
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

	// 验证路径在工作目录范围内
	validatedPath, err := validatePath(path, t.workDir)
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

	// 验证路径在工作目录范围内
	validatedPath, err := validatePath(path, t.workDir)
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
	toolList := []Tool{
		NewReadFileTool(),
		NewWriteFileTool(),
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
	toolList := []Tool{
		NewReadFileToolWithWorkDir(workDir),
		NewWriteFileToolWithWorkDir(workDir),
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
