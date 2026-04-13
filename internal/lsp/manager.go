package lsp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// maxCrashRestart 语言服务器崩溃后最大重启次数
const maxCrashRestart = 3

// LanguageServerConfig 语言服务器配置
type LanguageServerConfig struct {
	Language string   // "go", "typescript", "python"
	Command  string   // "gopls", "typescript-language-server", "pylsp"
	Args     []string // 启动参数
}

// LSPManager 语言服务器管理器
// 负责管理多个语言服务器的生命周期，包括：
// - 懒启动（首次请求时启动）
// - 空闲超时自动关闭
// - 崩溃自动重启（最多3次）
// - 优雅关闭
type LSPManager struct {
	clients map[string]*clientEntry // language -> client entry
	configs map[string]LanguageServerConfig
	rootURI string
	rootDir string
	mu      sync.RWMutex

	enabled bool // 是否启用LSP
}

// clientEntry 客户端条目，包含空闲计时器
type clientEntry struct {
	client    *LSPClient
	lastUsed  time.Time
	idleTimer *time.Timer
}

// NewManager 创建新的LSP管理器
// rootDir: 项目根目录（用于构造file:// URI）
func NewManager(rootDir string) *LSPManager {
	absDir, err := filepath.Abs(rootDir)
	if err != nil {
		absDir = rootDir
	}

	rootURI := pathToURI(absDir)

	return &LSPManager{
		clients: make(map[string]*clientEntry),
		configs: make(map[string]LanguageServerConfig),
		rootURI: rootURI,
		rootDir: absDir,
		enabled: true, // 默认启用
	}
}

// SetEnabled 设置是否启用LSP
func (m *LSPManager) SetEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.enabled = enabled
}

// IsEnabled 返回是否启用LSP
func (m *LSPManager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.enabled
}

// Configure 配置语言服务器
func (m *LSPManager) Configure(language string, config LanguageServerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configs[language] = config
}

// ConfigureFromMap 批量配置语言服务器
func (m *LSPManager) ConfigureFromMap(configs map[string]LanguageServerConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for lang, cfg := range configs {
		m.configs[lang] = cfg
	}
}

// GetDiagnostics 获取指定文件的诊断信息
// 根据文件扩展名自动路由到对应的语言服务器
// 如果对应服务器未启动，会自动启动
func (m *LSPManager) GetDiagnostics(ctx context.Context, filePath string) ([]Diagnostic, error) {
	language := LanguageFromPath(filePath)
	if language == "" {
		return nil, fmt.Errorf("unsupported file type: %s", filePath)
	}

	return m.GetDiagnosticsForLanguage(ctx, filePath, language)
}

// GetDiagnosticsForLanguage 获取指定语言服务器的诊断信息
func (m *LSPManager) GetDiagnosticsForLanguage(ctx context.Context, filePath, language string) ([]Diagnostic, error) {
	if !m.IsEnabled() {
		return nil, fmt.Errorf("LSP is disabled")
	}

	// 获取或启动语言服务器
	client, err := m.getOrCreateClient(ctx, language)
	if err != nil {
		return nil, err
	}

	// 读取文件内容
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	uri := pathToURI(filePath)

	// 发送didOpen
	if err := client.DidOpen(ctx, uri, language, string(content)); err != nil {
		return nil, fmt.Errorf("didOpen failed: %w", err)
	}

	// 等待诊断通知到达（语言服务器需要时间分析代码）
	// 使用渐进式等待策略
	if err := m.waitForDiagnostics(ctx, client, uri); err != nil {
		return nil, err
	}

	// 获取诊断结果
	diags, err := client.Diagnostics(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("failed to get diagnostics: %w", err)
	}

	return diags, nil
}

// waitForDiagnostics 等待诊断结果到达
// 使用渐进式等待：先等短时间，如果已有结果立即返回
func (m *LSPManager) waitForDiagnostics(ctx context.Context, client *LSPClient, uri string) error {
	// 渐进式等待策略：50ms, 100ms, 200ms, 500ms, 1s
	waitTimes := []time.Duration{50 * time.Millisecond, 100 * time.Millisecond, 200 * time.Millisecond, 500 * time.Millisecond, 1 * time.Second}

	for _, wait := range waitTimes {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}

		diags, _ := client.Diagnostics(ctx, uri)
		if len(diags) > 0 {
			return nil
		}
	}

	// 即使没有诊断结果也返回（文件可能没有问题）
	// 最后再等2秒给服务器更多时间
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(2 * time.Second):
	}

	return nil
}

// getOrCreateClient 获取或创建语言服务器客户端
func (m *LSPManager) getOrCreateClient(ctx context.Context, language string) (*LSPClient, error) {
	m.mu.RLock()
	if entry, ok := m.clients[language]; ok && entry.client.IsRunning() {
		entry.lastUsed = time.Now()
		m.mu.RUnlock()
		return entry.client, nil
	}
	m.mu.RUnlock()

	// 需要创建新客户端
	m.mu.Lock()
	defer m.mu.Unlock()

	// 双重检查
	if entry, ok := m.clients[language]; ok && entry.client.IsRunning() {
		entry.lastUsed = time.Now()
		return entry.client, nil
	}

	config, ok := m.configs[language]
	if !ok {
		return nil, fmt.Errorf("no language server configured for: %s", language)
	}

	// 检查崩溃次数
	if oldEntry, ok := m.clients[language]; ok {
		if oldEntry.client.CrashCount() >= maxCrashRestart {
			return nil, fmt.Errorf("language server for %s has crashed too many times (max %d)", language, maxCrashRestart)
		}
	}

	client := NewClient(config.Command, config.Args, m.rootURI)

	if err := client.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start language server %s: %w", language, err)
	}

	if err := client.Initialize(ctx); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("failed to initialize language server %s: %w", language, err)
	}

	// 启动空闲超时计时器（10分钟）
	idleTimer := time.AfterFunc(10*time.Minute, func() {
		m.stopIdleClient(language)
	})

	m.clients[language] = &clientEntry{
		client:    client,
		lastUsed:  time.Now(),
		idleTimer: idleTimer,
	}

	// 启动崩溃监控
	go m.monitorCrash(language, client)

	return client, nil
}

// stopIdleClient 停止空闲的语言服务器
func (m *LSPManager) stopIdleClient(language string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry, ok := m.clients[language]
	if !ok {
		return
	}

	// 检查是否真的空闲（超过10分钟未使用）
	if time.Since(entry.lastUsed) < 10*time.Minute {
		// 还在使用中，重置计时器
		entry.idleTimer.Reset(10 * time.Minute)
		return
	}

	log.Printf("[LSP] Stopping idle language server: %s", language)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = entry.client.Shutdown(ctx)
	delete(m.clients, language)
}

// monitorCrash 监控语言服务器崩溃并自动重启
func (m *LSPManager) monitorCrash(language string, client *LSPClient) {
	// 等待进程退出
	err := client.cmd.Wait()
	if !client.running {
		return // 正常关闭
	}

	log.Printf("[LSP] Language server %s crashed: %v", language, err)

	client.running = false
	client.IncrementCrash()

	m.mu.Lock()
	// 清理旧条目（但不重置崩溃计数，因为client对象引用不变）
	if entry, ok := m.clients[language]; ok && entry.client == client {
		if entry.idleTimer != nil {
			entry.idleTimer.Stop()
		}
		delete(m.clients, language)
	}
	m.mu.Unlock()

	if client.CrashCount() >= maxCrashRestart {
		log.Printf("[LSP] Language server %s has crashed %d times, not restarting", language, client.CrashCount())
		return
	}

	log.Printf("[LSP] Attempting to restart language server %s (attempt %d/%d)",
		language, client.CrashCount(), maxCrashRestart)

	// 自动重启
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, restartErr := m.getOrCreateClient(ctx, language)
	if restartErr != nil {
		log.Printf("[LSP] Failed to restart language server %s: %v", language, restartErr)
	}
}

// ShutdownAll 关闭所有语言服务器
func (m *LSPManager) ShutdownAll() {
	m.mu.Lock()
	defer m.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for language, entry := range m.clients {
		if entry.idleTimer != nil {
			entry.idleTimer.Stop()
		}
		log.Printf("[LSP] Shutting down language server: %s", language)
		_ = entry.client.Shutdown(ctx)
	}

	m.clients = make(map[string]*clientEntry)
}

// ==================== 辅助函数 ====================

// pathToURI 将文件路径转换为file:// URI
func pathToURI(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}
	// 在Windows上将反斜杠替换为正斜杠
	absPath = filepath.ToSlash(absPath)
	return "file:///" + strings.TrimPrefix(absPath, "/")
}
