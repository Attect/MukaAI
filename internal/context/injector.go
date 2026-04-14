package context

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Attect/MukaAI/internal/model"
)

// Injector 上下文注入器
// 负责将查询结果组装为模型可理解的项目上下文，并注入到消息历史中
type Injector struct {
	indexer   *Indexer // 索引器引用
	maxTokens int      // 最大Token预算
	query     *QueryEngine
}

// NewInjector 创建新的上下文注入器
// indexer: 索引器实例
// maxTokens: 最大Token预算（注入上下文最多占用的Token数）
func NewInjector(indexer *Indexer, maxTokens int) *Injector {
	if maxTokens <= 0 {
		maxTokens = 40000 // 默认40K token预算
	}
	return &Injector{
		indexer:   indexer,
		maxTokens: maxTokens,
		query:     NewQueryEngine(indexer),
	}
}

// InjectContext 注入项目上下文到消息历史
// taskDesc: 任务描述
// messages: 当前的消息历史
// 返回注入上下文后的消息历史（在系统消息后插入上下文消息）
func (inj *Injector) InjectContext(taskDesc string, messages []model.Message) []model.Message {
	if !inj.indexer.IsReady() {
		return messages
	}

	// 查询相关文件（带文件提示）
	files := inj.query.QueryWithFileHints(taskDesc, 10)
	if len(files) == 0 {
		return messages
	}

	// 组装上下文文本
	contextText := inj.buildContextText(files, taskDesc)

	// 检查Token预算
	tokenCount := EstimateTokenCount(contextText)
	if tokenCount > inj.maxTokens {
		// 超出预算，逐步截断
		contextText = inj.truncateToBudget(files, taskDesc)
	}

	if contextText == "" {
		return messages
	}

	// 将上下文注入到系统消息之后，使用user角色
	// 注意：不能使用system角色，因为llama.cpp的Jinja聊天模板
	// 通常要求system消息只能在对话开头且仅有一条，多个system消息会导致500错误
	result := make([]model.Message, 0, len(messages)+1)
	injected := false
	for _, msg := range messages {
		result = append(result, msg)
		if msg.Role == model.RoleSystem && !injected {
			// 在系统消息之后插入上下文消息（使用user角色避免模板错误）
			result = append(result, model.Message{
				Role:    model.RoleUser,
				Content: "[Project Context]\n" + contextText,
			})
			injected = true
		}
	}

	// 如果没有系统消息，在开头插入
	if !injected {
		result = append([]model.Message{{
			Role:    model.RoleUser,
			Content: "[Project Context]\n" + contextText,
		}}, result...)
	}

	return result
}

// BuildContextString 仅构建上下文字符串（不注入到消息历史）
// 用于日志记录和调试
func (inj *Injector) BuildContextString(taskDesc string, topN int) (string, []FileEntry) {
	if !inj.indexer.IsReady() {
		return "", nil
	}

	files := inj.query.QueryWithFileHints(taskDesc, topN)
	if len(files) == 0 {
		return "", nil
	}

	contextText := inj.buildContextText(files, taskDesc)
	return contextText, files
}

// buildContextText 组装项目上下文文本
func (inj *Injector) buildContextText(files []FileEntry, taskDesc string) string {
	var sb strings.Builder

	// 头部信息
	sb.WriteString("=== Project Context (Auto-generated) ===\n")
	sb.WriteString(fmt.Sprintf("Project: %s\n", inj.indexer.GetProjectName()))
	sb.WriteString(fmt.Sprintf("Files: %d files indexed\n\n", inj.indexer.GetFileCount()))
	sb.WriteString("--- Related Files ---\n\n")

	for _, file := range files {
		// 读取文件内容
		_, truncated, err := ReadFileContent(file.AbsPath)
		if err != nil {
			// 文件读取失败，仅列出文件信息
			sb.WriteString(fmt.Sprintf("## %s (%s)\n[File unreadable: %s]\n\n---\n\n",
				file.Path, file.Language, err.Error()))
			continue
		}

		// 文件内容为空，跳过
		if strings.TrimSpace(truncated) == "" {
			continue
		}

		// 格式化输出
		sb.WriteString(fmt.Sprintf("## %s (%s)\n", file.Path, file.Language))

		// 如果有符号信息，列出符号列表
		if len(file.Symbols) > 0 {
			sb.WriteString("// Symbols: ")
			symbolNames := make([]string, 0, len(file.Symbols))
			for _, sym := range file.Symbols {
				symbolNames = append(symbolNames, sym.Name)
			}
			// 限制符号列表长度
			if len(symbolNames) > 15 {
				symbolNames = symbolNames[:15]
				symbolNames = append(symbolNames, "...")
			}
			sb.WriteString(strings.Join(symbolNames, ", "))
			sb.WriteString("\n")
		}

		sb.WriteString(truncated)
		sb.WriteString("\n\n---\n\n")
	}

	sb.WriteString("=== End Project Context ===\n")

	return sb.String()
}

// truncateToBudget 截断上下文到Token预算内
// 按相关性从低到高逐步移除文件内容
func (inj *Injector) truncateToBudget(files []FileEntry, taskDesc string) string {
	// 从后向前（相关性最低的）逐步移除
	for i := len(files) - 1; i >= 1; i-- {
		contextText := inj.buildContextText(files[:i], taskDesc)
		if EstimateTokenCount(contextText) <= inj.maxTokens {
			return contextText
		}
	}

	// 只剩一个文件仍然超预算，截断该文件内容
	if len(files) > 0 {
		return inj.truncateSingleFile(files[0])
	}

	return ""
}

// truncateSingleFile 单个文件的截断版本（更激进的截断）
func (inj *Injector) truncateSingleFile(file FileEntry) string {
	var sb strings.Builder
	sb.WriteString("=== Project Context (Auto-generated) ===\n")
	sb.WriteString(fmt.Sprintf("Project: %s\n", inj.indexer.GetProjectName()))
	sb.WriteString(fmt.Sprintf("Files: %d files indexed\n\n", inj.indexer.GetFileCount()))
	sb.WriteString("--- Related Files ---\n\n")
	sb.WriteString(fmt.Sprintf("## %s (%s)\n", file.Path, file.Language))

	// 更激进的截断：只保留前100行
	_, truncated, err := ReadFileContent(file.AbsPath)
	if err != nil {
		return ""
	}

	lines := strings.Split(truncated, "\n")
	if len(lines) > 100 {
		lines = lines[:100]
		lines = append(lines, "... (truncated)")
	}
	sb.WriteString(strings.Join(lines, "\n"))
	sb.WriteString("\n\n=== End Project Context ===\n")

	return sb.String()
}

// InjectIntoHistory 将上下文注入到HistoryManager
// 这是一个便捷方法，直接操作HistoryManager
// 返回注入的文件列表（用于日志）
func (inj *Injector) InjectIntoHistory(taskDesc string, addMessage func(msg model.Message)) []FileEntry {
	if !inj.indexer.IsReady() {
		return nil
	}

	files := inj.query.QueryWithFileHints(taskDesc, 10)
	if len(files) == 0 {
		return nil
	}

	contextText := inj.buildContextText(files, taskDesc)
	tokenCount := EstimateTokenCount(contextText)
	if tokenCount > inj.maxTokens {
		contextText = inj.truncateToBudget(files, taskDesc)
	}

	if contextText == "" {
		return nil
	}

	// 使用user角色而非system角色，避免llama.cpp聊天模板的"system消息必须在开头"约束
	addMessage(model.Message{
		Role:    model.RoleUser,
		Content: "[Project Context]\n" + contextText,
	})

	return files
}

// SetMaxTokens 设置最大Token预算
func (inj *Injector) SetMaxTokens(maxTokens int) {
	if maxTokens > 0 {
		inj.maxTokens = maxTokens
	}
}

// GetMaxTokens 获取最大Token预算
func (inj *Injector) GetMaxTokens() int {
	return inj.maxTokens
}

// NewInjectorFromContextSize 根据模型上下文窗口大小创建注入器
// contextSize: 模型的上下文窗口大小（token数）
// 注入器预算为上下文窗口的20%
func NewInjectorFromContextSize(indexer *Indexer, contextSize int) *Injector {
	maxTokens := contextSize / 5 // 20%
	if maxTokens <= 0 {
		maxTokens = 40000
	}
	return NewInjector(indexer, maxTokens)
}

// resolveWorkDir 获取工作目录（用于相对路径计算）
func resolveWorkDir(workDir string) string {
	if workDir == "" {
		wd, _ := filepath.Abs(".")
		return wd
	}
	abs, _ := filepath.Abs(workDir)
	return abs
}
