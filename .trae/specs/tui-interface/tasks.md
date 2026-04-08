# Tasks

## Phase 1: 项目依赖与基础结构

- [x] Task 1: 添加 Bubble Tea 依赖
  - [x] SubTask 1.1: 在 go.mod 中添加 bubbletea v2 依赖
  - [x] SubTask 1.2: 添加 bubbles UI 组件库依赖
  - [x] SubTask 1.3: 添加 lipgloss 样式库依赖
  - [x] SubTask 1.4: 运行 go mod tidy 确保依赖正确

- [x] Task 2: 创建 TUI 模块目录结构
  - [x] SubTask 2.1: 创建 internal/tui/ 目录
  - [x] SubTask 2.2: 创建 internal/tui/components/ 子目录
  - [x] SubTask 2.3: 创建基础文件框架（app.go, messages.go, styles.go）

## Phase 2: Bubble Tea 框架集成

- [ ] Task 3: 实现主应用 Model
  - [ ] SubTask 3.1: 创建 internal/tui/app.go，定义 AppModel 结构体
  - [ ] SubTask 3.2: 实现 Init() 方法，初始化应用状态
  - [ ] SubTask 3.3: 实现 Update() 方法，处理消息和事件
  - [ ] SubTask 3.4: 实现 View() 方法，渲染界面

- [ ] Task 4: 定义消息类型
  - [ ] SubTask 4.1: 创建 internal/tui/messages.go
  - [ ] SubTask 4.2: 定义流式消息类型（StreamThinkingMsg, StreamContentMsg 等）
  - [ ] SubTask 4.3: 定义用户交互消息（UserInputMsg, CommandMsg 等）
  - [ ] SubTask 4.4: 定义状态更新消息（TokenUpdateMsg, StatusChangeMsg 等）

## Phase 3: UI 组件开发

- [ ] Task 5: 实现状态栏组件
  - [ ] SubTask 5.1: 创建 internal/tui/components/statusbar.go
  - [ ] SubTask 5.2: 实现工作目录显示
  - [ ] SubTask 5.3: 实现 token 用量显示
  - [ ] SubTask 5.4: 实现推理次数显示
  - [ ] SubTask 5.5: 实现样式美化

- [ ] Task 6: 实现输入组件
  - [ ] SubTask 6.1: 创建 internal/tui/components/input.go
  - [ ] SubTask 6.2: 集成 bubbles/textarea 组件
  - [ ] SubTask 6.3: 实现单行/多行输入模式切换
  - [ ] SubTask 6.4: 实现输入历史记录
  - [ ] SubTask 6.5: 实现内置命令解析（/cd, /conversations 等）

- [ ] Task 7: 实现对话显示组件
  - [ ] SubTask 7.1: 创建 internal/tui/components/chat.go
  - [ ] SubTask 7.2: 集成 bubbles/viewport 组件实现滚动
  - [ ] SubTask 7.3: 实现消息渲染逻辑
  - [ ] SubTask 7.4: 实现自动滚动到底部
  - [ ] SubTask 7.5: 实现消息样式区分（用户、模型、工具）

- [ ] Task 8: 实现对话列表弹窗组件
  - [ ] SubTask 8.1: 创建 internal/tui/components/dialog.go
  - [ ] SubTask 8.2: 实现对话列表显示
  - [ ] SubTask 8.3: 实现状态标注（活动、等待、结束）
  - [ ] SubTask 8.4: 实现时间排序
  - [ ] SubTask 8.5: 实现对话选择和切换

## Phase 4: 样式系统

- [ ] Task 9: 实现样式定义
  - [ ] SubTask 9.1: 创建 internal/tui/styles.go
  - [ ] SubTask 9.2: 定义颜色主题
  - [ ] SubTask 9.3: 定义消息样式（用户、思考、正文、工具、错误）
  - [ ] SubTask 9.4: 定义状态样式（活动、等待、结束）
  - [ ] SubTask 9.5: 定义布局样式（边框、间距、对齐）

## Phase 5: 流式输出处理

- [ ] Task 10: 实现流式消息处理
  - [ ] SubTask 10.1: 在 internal/agent/core.go 中添加 StreamHandler 接口
  - [ ] SubTask 10.2: 实现 OnThinking() 回调方法
  - [ ] SubTask 10.3: 实现 OnContent() 回调方法
  - [ ] SubTask 10.4: 实现 OnToolCall() 回调方法
  - [ ] SubTask 10.5: 实现 OnToolResult() 回调方法
  - [ ] SubTask 10.6: 实现 OnComplete() 回调方法

- [ ] Task 11: 实现实时 UI 更新
  - [ ] SubTask 11.1: 实现流式消息到 Bubble Tea 消息的转换
  - [ ] SubTask 11.2: 实现消息缓冲和批量更新机制
  - [ ] SubTask 11.3: 实现思考内容块的实时更新
  - [ ] SubTask 11.4: 实现正文内容块的实时更新
  - [ ] SubTask 11.5: 实现工具调用块的实时更新

## Phase 6: 工具调用格式化

- [ ] Task 12: 实现工具调用格式化器
  - [ ] SubTask 12.1: 创建 internal/tui/formatter.go
  - [ ] SubTask 12.2: 实现工具调用 JSON 解析
  - [ ] SubTask 12.3: 实现参数格式化显示
  - [ ] SubTask 12.4: 实现工具结果格式化显示
  - [ ] SubTask 12.5: 实现折叠/展开功能

## Phase 7: Token 统计

- [ ] Task 13: 实现 Token 统计功能
  - [ ] SubTask 13.1: 在消息结构中添加 token 用量字段
  - [ ] SubTask 13.2: 实现单次对话 token 显示
  - [ ] SubTask 13.3: 实现总 token 用量累计
  - [ ] SubTask 13.4: 实现推理次数统计
  - [ ] SubTask 13.5: 在状态栏中显示统计数据

## Phase 8: 工作目录管理

- [ ] Task 14: 实现工作目录管理
  - [ ] SubTask 14.1: 实现当前目录显示
  - [ ] SubTask 14.2: 实现 /cd 命令解析
  - [ ] SubTask 14.3: 实现目录切换逻辑
  - [ ] SubTask 14.4: 实现目录验证和错误处理
  - [ ] SubTask 14.5: 更新状态栏显示

## Phase 9: 子对话管理

- [ ] Task 15: 实现子对话管理
  - [ ] SubTask 15.1: 定义对话状态类型（active, waiting, finished）
  - [ ] SubTask 15.2: 实现对话创建和状态更新
  - [ ] SubTask 15.3: 实现对话列表显示
  - [ ] SubTask 15.4: 实现对话切换功能
  - [ ] SubTask 15.5: 实现父子对话关系管理

## Phase 10: 命令行入口

- [ ] Task 16: 实现 TUI 启动命令
  - [ ] SubTask 16.1: 在 cmd/agentplus/main.go 中添加 tui 子命令
  - [ ] SubTask 16.2: 实现命令行参数解析（--dir, --load）
  - [ ] SubTask 16.3: 初始化 TUI 应用
  - [ ] SubTask 16.4: 启动 Bubble Tea 程序
  - [ ] SubTask 16.5: 实现优雅退出

## Phase 11: 内置命令

- [ ] Task 17: 实现内置命令系统
  - [ ] SubTask 17.1: 实现命令解析框架
  - [ ] SubTask 17.2: 实现 /cd 命令
  - [ ] SubTask 17.3: 实现 /conversations 命令
  - [ ] SubTask 17.4: 实现 /clear 命令
  - [ ] SubTask 17.5: 实现 /save 命令
  - [ ] SubTask 17.6: 实现 /help 命令
  - [ ] SubTask 17.7: 实现 /exit 命令

## Phase 12: 快捷键支持

- [ ] Task 18: 实现快捷键系统
  - [ ] SubTask 18.1: 实现 Enter 提交输入
  - [ ] SubTask 18.2: 实现 Ctrl+Enter 多行提交
  - [ ] SubTask 18.3: 实现 Tab 切换输入模式
  - [ ] SubTask 18.4: 实现 Ctrl+L 显示对话列表
  - [ ] SubTask 18.5: 实现 Ctrl+C/Esc 退出

## Phase 13: 测试与优化

- [ ] Task 19: 编写单元测试
  - [ ] SubTask 19.1: 编写消息处理测试
  - [ ] SubTask 19.2: 编写工具格式化测试
  - [ ] SubTask 19.3: 编写命令解析测试
  - [ ] SubTask 19.4: 编写状态管理测试

- [ ] Task 20: 集成测试与优化
  - [ ] SubTask 20.1: 测试流式输出显示效果
  - [ ] SubTask 20.2: 测试大量消息的滚动性能
  - [ ] SubTask 20.3: 测试子对话管理功能
  - [ ] SubTask 20.4: 优化内存占用
  - [ ] SubTask 20.5: 优化渲染性能

# Task Dependencies

- Task 2 依赖 Task 1（需要依赖库）
- Task 3, Task 4 依赖 Task 2（需要目录结构）
- Task 5, Task 6, Task 7, Task 8 依赖 Task 3, Task 4（需要主 Model）
- Task 9 可独立执行（样式定义）
- Task 10 依赖 Task 3（需要 Model 结构）
- Task 11 依赖 Task 10（需要流式处理接口）
- Task 12 可独立执行（格式化器）
- Task 13 依赖 Task 5, Task 7（需要状态栏和对话组件）
- Task 14 依赖 Task 5（需要状态栏）
- Task 15 依赖 Task 8（需要对话列表组件）
- Task 16 依赖 Task 3, Task 10（需要完整 TUI 功能）
- Task 17 依赖 Task 6（需要输入组件）
- Task 18 依赖 Task 3（需要 Update 方法）
- Task 19, Task 20 依赖所有前置任务

# 并行执行建议

以下任务可以并行执行：
- Task 5, Task 6, Task 7, Task 8（UI 组件开发）
- Task 9, Task 12（样式和格式化器）
- Task 13, Task 14, Task 15（功能模块）
- Task 17, Task 18（命令和快捷键）
- Task 19, Task 20（测试阶段）
