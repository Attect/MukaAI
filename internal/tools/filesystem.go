package tools

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ==================== ReadFile Tool ====================

// ReadFileTool 读取文件内容工具
type ReadFileTool struct{}

func NewReadFileTool() *ReadFileTool {
	return &ReadFileTool{}
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

	// 检查文件是否存在
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorResult(fmt.Sprintf("file does not exist: %s", path)), nil
		}
		return NewErrorResultWithError(fmt.Sprintf("failed to stat file: %s", path), err), nil
	}

	if info.IsDir() {
		return NewErrorResult(fmt.Sprintf("path is a directory, not a file: %s", path)), nil
	}

	// 读取文件内容
	content, err := os.ReadFile(path)
	if err != nil {
		return NewErrorResultWithError(fmt.Sprintf("failed to read file: %s", path), err), nil
	}

	return NewSuccessResult(map[string]interface{}{
		"path":    path,
		"content": string(content),
		"size":    len(content),
	}), nil
}

// ==================== WriteFile Tool ====================

// WriteFileTool 写入文件工具
type WriteFileTool struct{}

func NewWriteFileTool() *WriteFileTool {
	return &WriteFileTool{}
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

	// 创建父目录（如果不存在）
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return NewErrorResultWithError(fmt.Sprintf("failed to create parent directory: %s", dir), err), nil
	}

	// 写入文件
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return NewErrorResultWithError(fmt.Sprintf("failed to write file: %s", path), err), nil
	}

	return NewSuccessResult(map[string]interface{}{
		"path":        path,
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
type ListDirectoryTool struct{}

func NewListDirectoryTool() *ListDirectoryTool {
	return &ListDirectoryTool{}
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

	// 检查目录是否存在
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorResult(fmt.Sprintf("directory does not exist: %s", path)), nil
		}
		return NewErrorResultWithError(fmt.Sprintf("failed to stat directory: %s", path), err), nil
	}

	if !info.IsDir() {
		return NewErrorResult(fmt.Sprintf("path is not a directory: %s", path)), nil
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
		err = filepath.WalkDir(path, func(walkPath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			// 跳过根目录本身
			if walkPath == path {
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
			return NewErrorResultWithError(fmt.Sprintf("failed to walk directory: %s", path), err), nil
		}
	} else {
		// 非递归，只列出直接内容
		entries, err := os.ReadDir(path)
		if err != nil {
			return NewErrorResultWithError(fmt.Sprintf("failed to read directory: %s", path), err), nil
		}

		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			files = append(files, FileInfo{
				Name:    entry.Name(),
				Path:    filepath.Join(path, entry.Name()),
				IsDir:   entry.IsDir(),
				Size:    info.Size(),
				Mode:    info.Mode().String(),
				ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}
	}

	return NewSuccessResult(map[string]interface{}{
		"path":      path,
		"count":     len(files),
		"entries":   files,
		"recursive": recursive,
	}), nil
}

// ==================== DeleteFile Tool ====================

// DeleteFileTool 删除文件工具
type DeleteFileTool struct{}

func NewDeleteFileTool() *DeleteFileTool {
	return &DeleteFileTool{}
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

	// 检查路径是否存在
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewErrorResult(fmt.Sprintf("path does not exist: %s", path)), nil
		}
		return NewErrorResultWithError(fmt.Sprintf("failed to stat path: %s", path), err), nil
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
			if err := os.RemoveAll(path); err != nil {
				return NewErrorResultWithError(fmt.Sprintf("failed to remove directory recursively: %s", path), err), nil
			}
		} else {
			if err := os.Remove(path); err != nil {
				// 如果目录非空，提示使用recursive
				if strings.Contains(err.Error(), "directory not empty") {
					return NewErrorResult(fmt.Sprintf("directory is not empty, use recursive=true to delete: %s", path)), nil
				}
				return NewErrorResultWithError(fmt.Sprintf("failed to remove directory: %s", path), err), nil
			}
		}
	} else {
		if err := os.Remove(path); err != nil {
			return NewErrorResultWithError(fmt.Sprintf("failed to remove file: %s", path), err), nil
		}
	}

	return NewSuccessResult(map[string]interface{}{
		"path":      path,
		"was_dir":   info.IsDir(),
		"recursive": recursive,
		"message":   "deleted successfully",
	}), nil
}

// ==================== CreateDirectory Tool ====================

// CreateDirectoryTool 创建目录工具
type CreateDirectoryTool struct{}

func NewCreateDirectoryTool() *CreateDirectoryTool {
	return &CreateDirectoryTool{}
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

	// 检查目录是否已存在
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return NewSuccessResult(map[string]interface{}{
				"path":    path,
				"created": false,
				"message": "directory already exists",
			}), nil
		}
		return NewErrorResult(fmt.Sprintf("path exists but is not a directory: %s", path)), nil
	}

	// 创建目录
	if err := os.MkdirAll(path, 0755); err != nil {
		return NewErrorResultWithError(fmt.Sprintf("failed to create directory: %s", path), err), nil
	}

	return NewSuccessResult(map[string]interface{}{
		"path":    path,
		"created": true,
		"message": "directory created successfully",
	}), nil
}

// ==================== 工具注册函数 ====================

// RegisterFilesystemTools 注册所有文件系统工具到指定注册中心
func RegisterFilesystemTools(registry *ToolRegistry) error {
	tools := []Tool{
		NewReadFileTool(),
		NewWriteFileTool(),
		NewListDirectoryTool(),
		NewDeleteFileTool(),
		NewCreateDirectoryTool(),
	}

	for _, tool := range tools {
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
