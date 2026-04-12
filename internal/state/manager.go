package state

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// CleanupConfig 清理配置，控制过期状态文件的自动清理行为
type CleanupConfig struct {
	RetentionDays int           // 保留天数，超过此天数的已完成/已取消/失败任务将被清理
	CheckInterval time.Duration // 清理检查间隔
	Enabled       bool          // 是否启用自动清理
}

// DefaultCleanupConfig 返回默认的清理配置
// 默认保留30天，每24小时检查一次，启用自动清理
func DefaultCleanupConfig() CleanupConfig {
	return CleanupConfig{
		RetentionDays: 30,
		CheckInterval: 24 * time.Hour,
		Enabled:       true,
	}
}

// StateManager 管理任务状态的创建、更新、加载和保存
// 该管理器是并发安全的，使用读写锁保护状态访问
type StateManager struct {
	mu            sync.RWMutex          // 读写锁，保护并发访问
	stateDir      string                // 状态文件存储目录
	states        map[string]*TaskState // 内存中的状态缓存
	autoSave      bool                  // 是否自动保存
	cleanupCancel context.CancelFunc    // 用于取消清理goroutine
	cleanupDone   chan struct{}         // 清理goroutine退出信号
	cleanupConfig CleanupConfig         // 清理配置
}

// NewStateManager 创建一个新的状态管理器
// 参数：
//   - stateDir: 状态文件存储目录
//   - autoSave: 是否在每次更新后自动保存到文件
//
// 返回：
//   - *StateManager: 状态管理器实例
//   - error: 错误信息
func NewStateManager(stateDir string, autoSave bool) (*StateManager, error) {
	// 确保状态目录存在
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory %s: %w", stateDir, err)
	}

	return &StateManager{
		stateDir:      stateDir,
		states:        make(map[string]*TaskState),
		autoSave:      autoSave,
		cleanupConfig: DefaultCleanupConfig(),
	}, nil
}

// NewStateManagerWithCleanup 创建一个带清理配置的状态管理器
// 参数：
//   - stateDir: 状态文件存储目录
//   - autoSave: 是否在每次更新后自动保存到文件
//   - cleanupConfig: 清理配置
//
// 返回：
//   - *StateManager: 状态管理器实例
//   - error: 错误信息
func NewStateManagerWithCleanup(stateDir string, autoSave bool, cleanupConfig CleanupConfig) (*StateManager, error) {
	// 确保状态目录存在
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory %s: %w", stateDir, err)
	}

	// 验证清理配置的合理性
	if cleanupConfig.Enabled {
		if cleanupConfig.RetentionDays <= 0 {
			cleanupConfig.RetentionDays = 30
		}
		if cleanupConfig.CheckInterval <= 0 {
			cleanupConfig.CheckInterval = 24 * time.Hour
		}
	}

	return &StateManager{
		stateDir:      stateDir,
		states:        make(map[string]*TaskState),
		autoSave:      autoSave,
		cleanupConfig: cleanupConfig,
	}, nil
}

// StartCleanup 启动后台清理goroutine
// 该方法启动一个后台goroutine，按照配置的间隔定期扫描并清理过期状态文件
// 参数：
//   - ctx: 上下文，用于控制清理goroutine的生命周期
func (sm *StateManager) StartCleanup(ctx context.Context) {
	sm.mu.Lock()
	if !sm.cleanupConfig.Enabled {
		sm.mu.Unlock()
		return
	}

	// 如果已有清理goroutine在运行，先停止
	if sm.cleanupCancel != nil {
		sm.mu.Unlock()
		sm.StopCleanup()
		sm.mu.Lock()
	}

	cleanupCtx, cancel := context.WithCancel(ctx)
	sm.cleanupCancel = cancel
	sm.cleanupDone = make(chan struct{})
	done := sm.cleanupDone

	config := sm.cleanupConfig
	sm.mu.Unlock()

	go func() {
		defer close(done)

		// 启动时先执行一次清理
		sm.performCleanup(config.RetentionDays)

		ticker := time.NewTicker(config.CheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-cleanupCtx.Done():
				log.Printf("[StateManager] 清理goroutine收到停止信号，退出")
				return
			case <-ticker.C:
				sm.performCleanup(config.RetentionDays)
			}
		}
	}()

	log.Printf("[StateManager] 自动清理已启动，保留天数: %d，检查间隔: %v", config.RetentionDays, config.CheckInterval)
}

// StopCleanup 停止后台清理goroutine，并等待其退出
func (sm *StateManager) StopCleanup() {
	sm.mu.Lock()
	cancel := sm.cleanupCancel
	done := sm.cleanupDone
	sm.cleanupCancel = nil
	sm.cleanupDone = nil
	sm.mu.Unlock()

	if cancel != nil {
		cancel()
		// 等待goroutine退出
		if done != nil {
			<-done
		}
		log.Printf("[StateManager] 清理goroutine已停止")
	}
}

// CleanupNow 立即执行一次清理，返回清理的文件数量
// 该方法使用默认的保留天数（30天）进行清理
// 返回：
//   - int: 清理的文件数量
//   - error: 错误信息
func (sm *StateManager) CleanupNow() (int, error) {
	sm.mu.RLock()
	retentionDays := sm.cleanupConfig.RetentionDays
	sm.mu.RUnlock()
	return sm.performCleanup(retentionDays)
}

// performCleanup 执行实际的清理操作
// 扫描磁盘目录中的task-*.yaml文件，删除超过保留天数的已完成/已取消/失败任务
// 永不删除in_progress状态的任务
// 参数：
//   - retentionDays: 保留天数
//
// 返回：
//   - int: 清理的文件数量
//   - error: 聚合的错误信息
func (sm *StateManager) performCleanup(retentionDays int) (int, error) {
	pattern := filepath.Join(sm.stateDir, "task-*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return 0, fmt.Errorf("扫描状态文件失败: %w", err)
	}

	if len(files) == 0 {
		return 0, nil
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	cleaned := 0
	var firstErr error

	for _, file := range files {
		// 从文件加载状态以检查任务状态和更新时间
		state, err := LoadYAML(file)
		if err != nil {
			// 无法解析的文件，记录日志但跳过
			log.Printf("[StateManager] 无法解析状态文件 %s: %v，跳过", file, err)
			continue
		}

		// 永不清理进行中的任务
		if state.Task.Status == "in_progress" {
			continue
		}

		// 只清理已完成、已取消、失败且超过保留期限的任务
		if state.Task.Status == "completed" || state.Task.Status == "cancelled" || state.Task.Status == "failed" {
			if state.Task.UpdatedAt.Before(cutoff) {
				if err := os.Remove(file); err != nil {
					log.Printf("[StateManager] 删除状态文件 %s 失败: %v", file, err)
					if firstErr == nil {
						firstErr = fmt.Errorf("删除状态文件 %s 失败: %w", file, err)
					}
					continue
				}

				// 从内存缓存中也删除
				sm.mu.Lock()
				delete(sm.states, state.Task.ID)
				sm.mu.Unlock()

				cleaned++
				log.Printf("[StateManager] 已清理过期状态文件: %s (任务ID: %s, 状态: %s, 更新时间: %s)",
					file, state.Task.ID, state.Task.Status, state.Task.UpdatedAt.Format("2006-01-02"))
			}
		}
	}

	if cleaned > 0 {
		log.Printf("[StateManager] 本次清理完成，共清理 %d 个过期状态文件", cleaned)
	}

	return cleaned, firstErr
}

// ListTasksFromDisk 扫描磁盘目录，列出所有task-*.yaml文件对应的任务ID
// 与ListTasks()不同，此方法直接扫描磁盘而非查询内存缓存
// 返回：
//   - []string: 任务ID列表
//   - error: 错误信息
func (sm *StateManager) ListTasksFromDisk() ([]string, error) {
	pattern := filepath.Join(sm.stateDir, "task-*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("扫描状态文件失败: %w", err)
	}

	ids := make([]string, 0, len(files))
	for _, file := range files {
		// 从文件名提取任务ID: task-{id}.yaml -> {id}
		base := filepath.Base(file)
		// 去掉 "task-" 前缀和 ".yaml" 后缀
		id := base
		if len(id) > 5 && id[:5] == "task-" {
			id = id[5:]
		}
		if len(id) > 5 && id[len(id)-5:] == ".yaml" {
			id = id[:len(id)-5]
		}
		if id != "" {
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// CreateTask 创建一个新的任务状态
// 参数：
//   - id: 任务唯一标识符
//   - goal: 任务目标描述
//
// 返回：
//   - *TaskState: 创建的任务状态
//   - error: 错误信息
func (sm *StateManager) CreateTask(id, goal string) (*TaskState, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 检查任务是否已存在
	if _, exists := sm.states[id]; exists {
		return nil, fmt.Errorf("task with id %s already exists", id)
	}

	// 创建新任务状态
	state := NewTaskState(id, goal)
	sm.states[id] = state

	// 如果启用自动保存，立即保存到文件
	if sm.autoSave {
		if err := sm.saveToFile(state); err != nil {
			delete(sm.states, id)
			return nil, fmt.Errorf("failed to save new task: %w", err)
		}
	}

	return state, nil
}

// Load 从文件加载任务状态
// 参数：
//   - id: 任务唯一标识符
//
// 返回：
//   - *TaskState: 加载的任务状态
//   - error: 错误信息
func (sm *StateManager) Load(id string) (*TaskState, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 先检查内存缓存
	if state, exists := sm.states[id]; exists {
		return state, nil
	}

	// 从文件加载
	filePath := sm.getFilePath(id)
	state, err := LoadYAML(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to load task %s: %w", id, err)
	}

	// 缓存到内存
	sm.states[id] = state
	return state, nil
}

// Save 保存任务状态到文件
// 参数：
//   - id: 任务唯一标识符
//
// 返回：
//   - error: 错误信息
func (sm *StateManager) Save(id string) error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[id]
	if !exists {
		return fmt.Errorf("task with id %s not found", id)
	}

	return sm.saveToFile(state)
}

// saveToFile 内部方法：将状态保存到文件（调用前必须持有锁）
func (sm *StateManager) saveToFile(state *TaskState) error {
	filePath := sm.getFilePath(state.Task.ID)
	return SaveYAML(state, filePath)
}

// UpdateProgress 更新任务进度
// 参数：
//   - id: 任务唯一标识符
//   - phase: 当前阶段
//   - completedStep: 刚完成的步骤（可选，传空字符串则不添加）
//
// 返回：
//   - error: 错误信息
func (sm *StateManager) UpdateProgress(id, phase, completedStep string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[id]
	if !exists {
		return fmt.Errorf("task with id %s not found", id)
	}

	// 更新阶段
	if phase != "" {
		state.SetCurrentPhase(phase)
	}

	// 添加完成的步骤
	if completedStep != "" {
		state.AddCompletedStep(completedStep)
		state.RemovePendingStep(completedStep)
	}

	// 自动保存
	if sm.autoSave {
		return sm.saveToFile(state)
	}

	return nil
}

// AddDecision 添加决策记录
// 参数：
//   - id: 任务唯一标识符
//   - decision: 决策内容
//
// 返回：
//   - error: 错误信息
func (sm *StateManager) AddDecision(id, decision string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[id]
	if !exists {
		return fmt.Errorf("task with id %s not found", id)
	}

	state.AddDecision(decision)

	if sm.autoSave {
		return sm.saveToFile(state)
	}

	return nil
}

// CompleteStep 完成一个步骤
// 参数：
//   - id: 任务唯一标识符
//   - step: 步骤描述
//
// 返回：
//   - error: 错误信息
func (sm *StateManager) CompleteStep(id, step string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[id]
	if !exists {
		return fmt.Errorf("task with id %s not found", id)
	}

	state.AddCompletedStep(step)
	state.RemovePendingStep(step)

	if sm.autoSave {
		return sm.saveToFile(state)
	}

	return nil
}

// SwitchAgent 切换活动Agent
// 参数：
//   - id: 任务唯一标识符
//   - role: 新的Agent角色
//   - summary: 前一个Agent的执行摘要
//   - duration: 前一个Agent的执行时长
//
// 返回：
//   - error: 错误信息
func (sm *StateManager) SwitchAgent(id, role, summary, duration string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[id]
	if !exists {
		return fmt.Errorf("task with id %s not found", id)
	}

	// 记录前一个Agent的历史
	if state.Agents.Active != "" {
		state.AddAgentRecord(state.Agents.Active, summary, duration)
	}

	// 切换到新Agent
	state.SetActiveAgent(role)

	if sm.autoSave {
		return sm.saveToFile(state)
	}

	return nil
}

// GetState 获取任务状态（只读）
// 参数：
//   - id: 任务唯一标识符
//
// 返回：
//   - *TaskState: 任务状态（只读，不要直接修改）
//   - error: 错误信息
func (sm *StateManager) GetState(id string) (*TaskState, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[id]
	if !exists {
		return nil, fmt.Errorf("task with id %s not found", id)
	}

	return state, nil
}

// GetYAMLSummary 获取任务的YAML摘要
// 参数：
//   - id: 任务唯一标识符
//
// 返回：
//   - string: 摘要字符串
//   - error: 错误信息
func (sm *StateManager) GetYAMLSummary(id string) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	state, exists := sm.states[id]
	if !exists {
		return "", fmt.Errorf("task with id %s not found", id)
	}

	return GetYAMLSummary(state)
}

// SetPendingSteps 设置待完成步骤列表
// 参数：
//   - id: 任务唯一标识符
//   - steps: 待完成步骤列表
//
// 返回：
//   - error: 错误信息
func (sm *StateManager) SetPendingSteps(id string, steps []string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[id]
	if !exists {
		return fmt.Errorf("task with id %s not found", id)
	}

	state.Progress.PendingSteps = steps
	state.Task.UpdatedAt = time.Now()

	if sm.autoSave {
		return sm.saveToFile(state)
	}

	return nil
}

// AddConstraint 添加约束条件
// 参数：
//   - id: 任务唯一标识符
//   - constraint: 约束条件
//
// 返回：
//   - error: 错误信息
func (sm *StateManager) AddConstraint(id, constraint string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[id]
	if !exists {
		return fmt.Errorf("task with id %s not found", id)
	}

	state.AddConstraint(constraint)

	if sm.autoSave {
		return sm.saveToFile(state)
	}

	return nil
}

// AddFile 添加相关文件信息
// 参数：
//   - id: 任务唯一标识符
//   - path: 文件路径
//   - description: 文件描述
//   - status: 文件状态
//
// 返回：
//   - error: 错误信息
func (sm *StateManager) AddFile(id, path, description, status string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[id]
	if !exists {
		return fmt.Errorf("task with id %s not found", id)
	}

	state.AddFile(path, description, status)

	if sm.autoSave {
		return sm.saveToFile(state)
	}

	return nil
}

// UpdateTaskStatus 更新任务状态
// 参数：
//   - id: 任务唯一标识符
//   - status: 新状态（pending, in_progress, completed, failed）
//
// 返回：
//   - error: 错误信息
func (sm *StateManager) UpdateTaskStatus(id, status string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	state, exists := sm.states[id]
	if !exists {
		return fmt.Errorf("task with id %s not found", id)
	}

	state.UpdateStatus(status)

	if sm.autoSave {
		return sm.saveToFile(state)
	}

	return nil
}

// getFilePath 获取任务状态文件的完整路径
// 参数：
//   - id: 任务唯一标识符
//
// 返回：
//   - string: 文件完整路径
func (sm *StateManager) getFilePath(id string) string {
	return filepath.Join(sm.stateDir, fmt.Sprintf("task-%s.yaml", id))
}

// ListTasks 列出所有已知任务ID
// 返回：
//   - []string: 任务ID列表
//   - error: 错误信息
func (sm *StateManager) ListTasks() ([]string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// 从内存缓存中获取
	ids := make([]string, 0, len(sm.states))
	for id := range sm.states {
		ids = append(ids, id)
	}

	return ids, nil
}

// DeleteTask 删除任务状态
// 参数：
//   - id: 任务唯一标识符
//
// 返回：
//   - error: 错误信息
func (sm *StateManager) DeleteTask(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// 从内存中删除
	delete(sm.states, id)

	// 删除文件
	filePath := sm.getFilePath(id)
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete task file %s: %w", filePath, err)
	}

	return nil
}
