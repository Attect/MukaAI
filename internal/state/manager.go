package state

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StateManager 管理任务状态的创建、更新、加载和保存
// 该管理器是并发安全的，使用读写锁保护状态访问
type StateManager struct {
	mu       sync.RWMutex          // 读写锁，保护并发访问
	stateDir string                // 状态文件存储目录
	states   map[string]*TaskState // 内存中的状态缓存
	autoSave bool                  // 是否自动保存
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
		stateDir: stateDir,
		states:   make(map[string]*TaskState),
		autoSave: autoSave,
	}, nil
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
