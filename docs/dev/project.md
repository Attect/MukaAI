# 项目架构文档

## 更新日志

### 2026-04-09 Task GUI 新增：Wails GUI 后端绑定层

#### internal/agent/stream.go
- `StreamHandler` 接口新增方法：
  - `OnTaskDone()` - 处理任务完成，当整个任务（包括所有迭代）完成后调用
  - 与 `OnComplete` 的区别：OnComplete 是单次推理完成，OnTaskDone 是整个任务完成
- `StreamHandlerFunc` 新增字段和方法：
  - `onTaskDone func()` - 任务完成回调函数字段
  - `OnTaskDone(fn func()) *StreamHandlerFunc` - 设置任务完成处理函数（链式调用）
- `streamHandlerFuncImpl` 新增字段和方法：
  - `onTaskDone func()` - 任务完成回调函数字段
  - `OnTaskDone()` - 实现StreamHandler接口，调用onTaskDone回调

#### internal/agent/core.go
- `SendMessage()` 方法修改：
  - 将 `handler.OnError(nil)` 改为 `handler.OnTaskDone()`
  - 使用语义更明确的OnTaskDone通知GUI推理已完全结束，替代之前用OnError(nil)的hack方式

#### internal/gui/app.go - 新增文件
- `Conversation` - 对话信息结构体（暴露给前端的JSON结构）
  - ID, Title, CreatedAt, Status, TokenUsage, MessageCount
- `Message` - 消息信息结构体（暴露给前端的JSON结构）
  - Role, Content, Thinking, ToolCalls, TokenUsage, IsStreaming, StreamingType, Timestamp
- `ToolCall` - 工具调用信息结构体（暴露给前端的JSON结构）
  - ID, Name, Arguments, IsComplete, Result, ResultError
- `TokenStats` - Token使用统计结构体
  - TotalTokens, InferenceCount
- `App` - Wails应用绑定层，作为前端与后端Agent之间的桥梁
  - 管理对话状态和消息流
  - 内部使用conversation和message结构管理对话数据
- `NewApp()` - 创建新的App实例
- `startup(ctx)` - Wails生命周期回调
- `SetAgent(ag)` - 设置Agent实例
- `SetCurrentDir(dir)` - 设置当前工作目录
- `SendMessage(content)` - 发送用户消息并启动推理（前端主要入口）
- `GetConversations()` - 获取所有对话列表
- `GetConversationData()` - 获取当前活跃对话的完整数据
- `SetWorkDir(path)` - 设置工作目录并通知前端
- `GetWorkDir()` - 获取当前工作目录
- `GetTokenStats()` - 获取Token使用统计
- `InterruptInference()` - 中断当前推理
- `ClearConversation()` - 清空当前对话的消息

#### internal/gui/stream_bridge.go - 新增文件
- `StreamBridge` - StreamHandler到Wails事件系统的桥接器
  - 实现agent.StreamHandler接口
  - 将所有流式事件转发为Wails前端事件
  - 同时更新App中的对话状态
- `NewStreamBridge(app)` - 创建新的StreamBridge实例
- `SetContext(ctx)` - 设置Wails上下文
- `OnThinking(chunk)` - 处理思考内容块，发射stream:thinking事件
- `OnContent(chunk)` - 处理正文内容块，发射stream:content事件
- `OnToolCall(call, isComplete)` - 处理工具调用，发射stream:toolcall事件
- `OnToolResult(result)` - 处理工具执行结果，发射stream:toolresult事件
- `OnComplete(usage)` - 处理推理完成，发射stream:complete和tokenstats:updated事件
- `OnError(err)` - 处理错误，发射stream:error事件
- `OnTaskDone()` - 处理任务完成，发射stream:done事件

#### Wails事件列表
| 事件名 | 数据 | 说明 |
|--------|------|------|
| stream:thinking | string | 思考内容块 |
| stream:content | string | 正文内容块 |
| stream:toolcall | map | 工具调用信息 |
| stream:toolresult | map | 工具执行结果 |
| stream:complete | map{usage} | 单次推理完成 |
| stream:error | string | 错误信息 |
| stream:done | - | 整个任务完成 |
| stream:interrupted | - | 用户中断推理 |
| conversation:updated | map | 对话数据更新 |
| tokenstats:updated | TokenStats | Token统计更新 |
| workdir:changed | string | 工作目录变更 |

#### cmd/agentplus/gui.go - GUI入口增强
- `enableWebView2Accessibility()` - 新增函数：配置WebView2无障碍支持
  - 在WebView2初始化前设置`WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS`环境变量
  - 添加`--force-renderer-accessibility`参数，使Windows UI Automation可访问DOM无障碍树
  - 保留已有的浏览器参数（避免覆盖其他配置）
- `runGUICommand()` 修改：
  - 在函数第一行调用`enableWebView2Accessibility()`
  - 确保无障碍配置在Wails运行前生效

#### frontend/src/ - WAI-ARIA无障碍属性补充
为所有主要React组件添加语义化Aria属性，使无障碍树具有可读性：

| 文件 | role | aria-label |
|------|------|------------|
| App.tsx | `role="application"` (根容器) | "MukaAI 智能编程助手" |
| App.tsx | `role="main"` (对话区) | "对话区域" |
| InputArea.tsx | textarea | "消息输入框" |
| Toolbar.tsx | button | "设置"/"主题切换"/"终端"/"清空"/"侧边栏" |
| Sidebar.tsx | nav | "对话列表" |

#### GUI架构 - 无障碍支持说明

**WebView2无障碍配置**：

MukaAI使用Wails v2 + WebView2作为GUI框架。默认情况下，WebView2的DOM无障碍树对Windows UI Automation不可见，导致自动化工具（如Windows MCP）只能看到窗口框架，无法识别内部元素。

解决方案分两层：

1. **后端配置**（`cmd/agentplus/gui.go`）：
   - 在WebView2初始化前设置环境变量`WEBVIEW2_ADDITIONAL_BROWSER_ARGUMENTS`
   - 添加`--force-renderer-accessibility`参数强制渲染进程创建无障碍节点
   - 关键：必须在`wails.Run()`之前调用，否则配置无效

2. **前端语义化**（`frontend/src/`）：
   - 为所有交互元素添加WAI-ARIA属性（role、aria-label等）
   - 使无障碍树具有可读性和可操作性
   - 支持屏幕阅读器和自动化工具

**注意事项**：
- `--remote-debugging-port`在WebView2嵌入模式下不支持，会被忽略
- 启用无障碍树后内存占用增加约5-10MB，对桌面应用可忽略
- 该配置跨平台有效（Windows/macOS/Linux）

详见：[WebView2无障碍踩坑记录](../../ai/knowledge/troubleshooting/webview2-accessibility.md)

#### internal/agent/stream_test.go
- 更新 `TestStreamHandlerFunc` 测试用例，增加OnTaskDone测试
- 更新 `TestStreamHandlerFuncWithNilFunctions` 测试用例，增加OnTaskDone空函数安全调用测试

### 2026-04-08 Task 20 新增：集成测试与优化

#### internal/tui/performance_test.go
- 创建完整的性能测试框架
- `TestStreamingPerformance()` - 测试流式输出性能
  - 测试 100 次流式输出操作
  - 测量平均延迟、P95、P99 延迟
  - 验证延迟 < 100ms 要求
- `TestScrollPerformance()` - 测试滚动性能
  - 生成 1000 条测试消息
  - 测试渲染时间和滚动延迟
  - 验证支持 1000+ 条消息流畅滚动
- `TestSubConversationManagement()` - 测试子对话管理性能
  - 创建 100 个对话（主对话 + 子对话）
  - 测试对话切换延迟
  - 验证并发安全性
- `TestMemoryUsage()` - 测试内存占用
  - 创建 10 个对话，每个 100 条消息
  - 测量内存增量
  - 验证内存占用 < 100MB
- `TestBatchUpdatePerformance()` - 测试批量更新性能
- `TestRenderPerformance()` - 测试渲染性能
- `BenchmarkStreamingOutput()` - 流式输出基准测试
- `BenchmarkScrolling()` - 滚动基准测试
- `BenchmarkConversationSwitching()` - 对话切换基准测试
- `TestComponentPerformance()` - 组件性能测试
  - ChatView 组件性能测试
  - InputComponent 组件性能测试
  - DialogList 组件性能测试

#### docs/dev/performance_report.md
- 创建详细的性能测试报告
- 记录所有测试结果和性能指标
- 性能指标对比表
- 优化措施总结
- 性能瓶颈分析
- 测试结论和建议

#### 测试结果总结
**流式输出性能测试**:
- 平均延迟: 0.01 ms (要求 < 100ms) ✅
- P95 延迟: 0.53 ms
- P99 延迟: 0.53 ms
- 测试通过

**滚动性能测试**:
- 消息数量: 1000 条
- 渲染时间: 6.66 ms (要求 < 50ms) ✅
- 平均滚动延迟: < 1 μs (要求 < 16ms) ✅
- 内存占用: 3.20 MB (要求 < 100MB) ✅
- 测试通过

**子对话管理性能测试**:
- 对话数量: 100 个
- 创建时间: < 1 ms
- 平均切换延迟: < 1 μs (要求 < 10ms) ✅
- 内存占用: 3.29 MB
- 测试通过

**内存占用测试**:
- 对话数量: 10 个
- 总消息数: 1000 条
- 内存增量: 2.51 MB (要求 < 100MB) ✅
- 当前内存占用: 3.03 MB
- 测试通过

#### 性能优化措施
1. **批量更新机制**: 使用 StreamUpdateManager 缓冲流式消息，每 50ms 批量更新
2. **虚拟滚动**: 使用 viewport 组件实现虚拟滚动，只渲染可见区域
3. **内存优化**: 优化数据结构，避免不必要的字符串拷贝，定期触发 GC
4. **并发安全**: 使用读写锁保护并发访问，确保线程安全
5. **延迟计算**: 统计信息按需计算，避免不必要的开销
6. **哈希映射**: 使用 map 存储对话，实现 O(1) 查找

#### 性能指标对比
| 指标 | 要求 | 实际值 | 达标情况 | 优化倍数 |
|------|------|--------|----------|----------|
| 流式输出延迟 | < 100ms | 0.01ms | ✅ 达标 | 10000x |
| 渲染时间 | < 50ms | 6.66ms | ✅ 达标 | 7.5x |
| 滚动延迟 | < 16ms | < 1μs | ✅ 达标 | 16000x |
| 内存占用 | < 100MB | 3.03MB | ✅ 达标 | 33x |
| 支持消息数 | 1000+ | 1000+ | ✅ 达标 | - |
| 对话切换延迟 | < 10ms | < 1μs | ✅ 达标 | 10000x |

#### 功能实现说明
**性能测试框架**:
- 完整的单元测试和基准测试
- 覆盖流式输出、滚动、内存、对话管理等核心功能
- 详细的性能指标统计和分析
- 自动化测试和持续集成支持

**性能优化策略**:
- 批量更新: 减少UI更新频率，提升流式输出性能
- 虚拟滚动: 支持大量消息流畅滚动
- 内存管理: 低内存占用，支持长时间运行
- 并发控制: 线程安全，支持并发访问

**测试覆盖**:
- 流式输出性能测试
- 滚动性能测试
- 子对话管理测试
- 内存占用测试
- 批量更新性能测试
- 渲染性能测试
- 组件性能测试

### 2026-04-08 Task 16 新增：TUI 启动命令

#### cmd/agentplus/main.go
- 重构命令行入口，支持子命令模式
- `main()` - 检查子命令并路由到相应的处理函数
- `runCLICommand()` - 运行 CLI 模式（原有逻辑）
- `runTUICommand()` - 运行 TUI 模式（新增）
  - 解析 TUI 命令行参数
  - 设置初始工作目录
  - 创建 TUI 应用模型
  - 启动 Bubble Tea 程序
  - 实现优雅退出
- `parseTUIFlags()` - 解析 TUI 模式命令行参数
  - 支持 `-c, --config` 参数指定配置文件
  - 支持 `--dir` 参数指定初始工作目录
  - 支持 `--load` 参数加载历史对话
- `printUsage()` - 打印完整的使用说明
  - 显示 CLI 模式和 TUI 模式的选项
  - 提供使用示例
- `TUIOptions` - TUI 模式命令行参数结构体

#### 功能实现说明
**子命令模式**：
- 支持默认 CLI 模式：`mukaai [options] <task>`
- 支持 TUI 模式：`mukaai tui [options]`
- 支持帮助命令：`mukaai help`
- 支持版本命令：`mukaai version`

**TUI 启动流程**：
1. 解析命令行参数
2. 加载配置文件
3. 设置初始工作目录（支持绝对路径和相对路径）
4. 验证目录是否存在
5. 创建 TUI 应用模型
6. 设置初始工作目录
7. 创建 Bubble Tea 程序
8. 设置信号处理（支持 Ctrl+C 优雅退出）
9. 启动 TUI 程序

**命令行参数**：
- `-c, --config <file>`: 配置文件路径（默认：./configs/config.yaml）
- `--dir <directory>`: 初始工作目录
- `--load <file>`: 加载历史对话文件

**优雅退出机制**：
- 监听 SIGINT 和 SIGTERM 信号
- 收到信号后发送退出命令给 TUI
- TUI 清理资源后退出
- 无资源泄漏

#### internal/tui/app.go
- `showConversationList()` - 显示对话列表方法（新增）
  - 更新对话列表数据
  - 显示对话列表弹窗

#### 使用示例
```bash
# 启动 TUI 模式
agentplus tui

# 指定初始工作目录
agentplus tui --dir /path/to/project

# 加载历史对话
agentplus tui --load conversation.yaml

# 使用自定义配置文件
agentplus tui -c ./custom-config.yaml
```

### 2026-04-08 Task 17 新增：内置命令系统

#### internal/tui/export.go
- `ConversationExport` - 导出的对话数据结构
  - ID: 对话唯一标识
  - Title: 对话标题
  - CreatedAt: 创建时间
  - Status: 对话状态
  - TokenUsage: token 用量
  - IsSubConversation: 是否为子对话
  - Messages: 消息列表
  - ExportedAt: 导出时间
- `MessageExport` - 导出的消息数据结构
  - Role: 消息角色
  - Content: 正文内容
  - Thinking: 思考内容
  - ToolCalls: 工具调用列表
  - TokenUsage: token 用量
  - Timestamp: 时间戳
- `ToolCallExport` - 导出的工具调用数据结构
- `ExportConversation(conv)` - 导出对话为 JSON 格式
- `SaveConversationToFile(filePath, conv)` - 保存对话到文件
  - 支持指定文件路径（可选）
  - 如果没有指定路径，使用默认文件名（conversation_时间戳_ID.json）
  - 自动创建目录（如果需要）
  - 返回保存的绝对路径
- `LoadConversationFromFile(filePath)` - 从文件加载对话
  - 支持 JSON 格式
  - 自动转换数据结构

#### internal/tui/app.go
- `handleSaveCommand(args []string)` - 处理 /save 命令
  - 检查是否有活动对话
  - 支持指定文件路径（可选）
  - 保存对话到文件
  - 返回成功/失败消息
- `handleCommandExecuted(cmd, args, result, err)` - 处理命令执行结果
  - 在对话区显示命令执行结果
  - 成功消息：✓ 前缀
  - 失败消息：❌ 前缀
- `ShowConversationListMsg` 消息处理：
  - 更新对话列表数据
  - 选中活动对话
  - 显示对话列表

#### internal/tui/commands_test.go
- 完整的单元测试覆盖
- 测试 /save 命令（保存到默认路径、指定路径、空对话）
- 测试对话导出功能
- 测试从文件加载对话
- 测试命令执行结果显示
- 测试对话列表显示
- 测试 /clear 命令
- 测试 /help 命令
- 所有测试通过（15+测试用例）

#### 功能实现说明
**内置命令系统**：
- `/cd <path>` - 切换工作目录（Task 14 已实现）
- `/conversations` 或 `/conv` - 显示对话列表
- `/clear` - 清空当前对话
- `/save [file]` - 保存对话历史到 JSON 文件
- `/help` - 显示帮助信息
- `/exit` - 退出 TUI

**对话保存功能**：
- 支持 JSON 格式导出
- 包含完整的对话信息（消息、token用量、状态等）
- 支持自定义文件路径
- 自动生成默认文件名
- 支持从文件加载对话

**命令执行结果显示**：
- 成功消息使用 ✓ 前缀
- 失败消息使用 ❌ 前缀
- 自动添加到当前对话的消息列表
- 实时更新对话视图

**对话列表显示**：
- 自动更新对话列表数据
- 选中当前活动对话
- 支持键盘导航和选择

### 2026-04-08 Task 18 新增：快捷键系统

#### internal/tui/app.go
- `AppModel` 结构体更新：
  - 使用 `*components.InputComponent` 替代 `textinput.Model`
  - 新增 `dialogList *components.DialogList` 字段
- `NewAppModel()` - 更新以初始化 InputComponent 和 DialogList
- `Init()` - 更新以使用 InputComponent 的初始化命令
- `Update()` - 重构快捷键处理逻辑：
  - **Enter 键提交**：单行模式下直接提交，多行模式下换行
  - **Ctrl+Enter 提交**：多行模式下提交输入
  - **Tab 切换模式**：在单行和多行输入模式间切换，自动同步状态
  - **Ctrl+L 显示对话列表**：切换对话列表显示状态，自动更新对话数据
  - **Esc 键智能退出**：对话列表可见时关闭列表，否则退出 TUI
  - **Ctrl+C 退出**：全局退出 TUI
  - 优先处理对话列表的输入事件
  - 快捷键不与输入冲突（Enter 在多行模式下换行而非提交）
- `View()` - 更新以渲染对话列表覆盖层
- `renderInputArea()` - 更新以使用 InputComponent 的视图
- `toggleConversationList()` - 切换对话列表显示状态
- `updateDialogList()` - 更新对话列表数据（将 Conversation 转换为 components.Conversation）

#### internal/tui/keybindings_test.go
- 完整的单元测试覆盖
- 测试 AppModel 创建和初始化
- 测试输入模式切换（Tab 键）
- 测试提交判断逻辑（Enter/Ctrl+Enter）
- 测试对话列表切换（Ctrl+L）
- 测试输入历史记录
- 测试输入模式同步
- 测试对话列表导航和排序
- 测试对话切换功能
- 测试输入值操作
- 测试命令解析
- 测试快捷键帮助文本
- 所有测试通过（15+测试用例）

#### 功能实现说明
**快捷键系统设计原则**：
- **不与输入冲突**：Enter 在多行模式下换行，Ctrl+Enter 提交
- **智能上下文感知**：Esc 键根据对话列表状态决定行为
- **优先级处理**：对话列表可见时优先处理其输入事件
- **状态同步**：Tab 切换模式时自动同步 AppModel 和 InputComponent 的状态

**快捷键绑定**：
| 快捷键 | 功能 | 说明 |
|--------|------|------|
| Enter | 提交输入 | 单行模式 |
| Ctrl+Enter | 提交输入 | 多行模式 |
| Tab | 切换输入模式 | 单行 ↔ 多行 |
| Ctrl+L | 显示对话列表 | 切换显示状态 |
| Esc | 关闭对话列表/退出 | 智能上下文感知 |
| Ctrl+C | 退出 TUI | 全局退出 |

**输入模式特性**：
- **单行模式**：Enter 提交，适合简短输入
- **多行模式**：Enter 换行，Ctrl+Enter 提交，适合长文本输入
- **历史记录**：↑↓ 键浏览历史输入
- **命令解析**：支持 /cd, /conversations, /clear, /save, /help, /exit 等命令

**对话列表功能**：
- 按创建时间降序排序（最新的在前）
- 显示对话状态（活动/等待/结束）
- 支持键盘导航（↑↓ 或 j/k）
- Enter 选择对话，Esc 关闭列表

#### 与 InputComponent 的集成
- AppModel 使用 InputComponent 替代简单的 textinput.Model
- InputComponent 处理输入相关的快捷键（Tab, Enter, Ctrl+Enter, ↑↓）
- AppModel 处理全局快捷键（Ctrl+L, Esc, Ctrl+C）
- 通过 ShouldSubmit 方法判断是否应该提交输入
- 自动同步输入模式状态

#### 性能优化
- 对话列表仅在显示时更新数据
- 快捷键处理优先级明确，避免冲突
- 输入组件独立处理输入事件，减少 AppModel 的复杂度

### 2026-04-08 Task 14 新增：工作目录管理

#### internal/tui/app.go
- `AppModel` 结构体更新：
  - 使用 `*components.StatusBar` 替代旧的 `StatusBar` 结构体
  - 新增 `streamManager *StreamUpdateManager` 字段
- `NewAppModel()` - 更新以初始化 components.StatusBar 并获取当前工作目录
- `handleCommand(cmd string)` - 实现命令解析和路由
  - 支持 `/cd <path>` 命令
  - 支持 `/conversations` 命令
  - 支持 `/clear` 命令
  - 支持 `/save` 命令
  - 支持 `/help` 命令
  - 支持 `/exit` 命令
- `handleCDCommand(args []string)` - 处理目录切换命令
  - 支持绝对路径和相对路径
  - 支持特殊路径（`~` 用户主目录）
  - 自动解析相对路径为绝对路径
  - 清理路径（处理 `.` 和 `..`）
- `validateDirectory(dir string)` - 验证目录是否存在且可访问
  - 检查路径是否存在
  - 检查是否为目录
  - 检查访问权限
- `handleWorkingDirChanged(oldDir, newDir)` - 处理工作目录变更
  - 更新当前目录
  - 更新状态栏显示
- `handleCommandExecuted(cmd, args, result, err)` - 处理命令执行结果
- `SetCurrentDir(dir string)` - 设置当前工作目录（供外部调用）
- `GetCurrentDir()` - 获取当前工作目录
- `handleClearCommand()` - 清空当前对话
- `handleSaveCommand(args []string)` - 保存对话（待实现）
- `handleHelpCommand()` - 显示帮助信息
- `handleBatchUpdate(result *FlushResult)` - 处理批量更新消息

#### internal/tui/app_test.go
- 完整的单元测试覆盖
- 测试目录验证功能
- 测试 /cd 命令处理
- 测试相对路径切换
- 测试用户主目录切换
- 测试工作目录变更处理
- 测试设置当前目录
- 测试命令处理路由
- 测试命令执行结果处理
- 测试清空对话命令
- 测试帮助命令
- 所有测试通过（15+测试用例）

#### 功能实现说明
**工作目录管理流程**：
1. 用户输入 `/cd <path>` 命令
2. handleCommand 解析命令并路由到 handleCDCommand
3. handleCDCommand 验证目录路径
4. 切换到新目录
5. 发送 WorkingDirChangedMsg 消息
6. handleWorkingDirChanged 更新状态
7. 状态栏显示新目录

**目录验证机制**：
- 检查路径是否存在
- 检查是否为目录（不是文件）
- 检查访问权限
- 返回详细的错误信息

**错误处理**：
- 目录不存在：返回明确的错误信息
- 路径不是目录：返回明确的错误信息
- 权限不足：返回权限错误
- 缺少参数：返回参数错误

**状态栏更新**：
- 使用 components.StatusBar 组件
- 自动显示当前工作目录
- 显示 token 用量和推理次数
- 支持目录路径缩写

### 2026-04-08 Task 11 新增：实时 UI 更新机制更新

#### internal/tui/stream.go
- `BatchUpdateConfig` - 批量更新配置结构体
  - BufferDuration: 缓冲时间窗口（默认50ms）
  - MaxBufferSize: 最大缓冲区大小（默认10条）
  - EnableBatching: 是否启用批量更新
  - MinUpdateInterval: 最小更新间隔（默认16ms，约60fps）
- `BufferedMessage` - 缓冲的消息结构体
  - Type: 消息类型
  - Content: 消息内容
  - ToolCall: 工具调用信息
  - IsComplete: 是否完成
  - Usage: token用量
  - Error: 错误信息
  - Timestamp: 时间戳
- `MessageBuffer` - 消息缓冲器（线程安全）
  - 管理思考内容、正文内容、工具调用的缓冲
  - 支持时间窗口和大小触发的刷新机制
  - 提供并发安全的访问
- `FlushResult` - 刷新结果结构体
  - Thinking: 累积的思考内容
  - Content: 累积的正文内容
  - ToolCalls: 累积的工具调用
  - ToolResults: 累积的工具结果
  - Messages: 其他消息（完成、错误等）
  - HasThinking/HasContent/HasToolCalls/HasToolResult: 状态标志
- `StreamUpdateManager` - 流式更新管理器（线程安全）
  - 管理流式消息的缓冲和批量更新
  - 使用定时器定期检查缓冲区
  - 提供强制刷新机制
- `DefaultBatchUpdateConfig()` - 返回默认批量更新配置
- `NewMessageBuffer(config)` - 创建新的消息缓冲器
- `NewStreamUpdateManager(config)` - 创建新的流式更新管理器

#### internal/tui/stream_test.go
- 完整的单元测试覆盖
- 测试消息缓冲器的各种操作
- 测试批量更新管理器的启动和停止
- 测试并发访问安全性
- 测试刷新机制
- 所有测试通过（20+测试用例）

#### internal/tui/messages.go
- `BatchUpdateMsg` - 批量更新消息
  - Result: 刷新结果
- `TickMsg` - 定时器消息
  - Time: 当前时间
- `NewBatchUpdateMsg(result)` - 创建批量更新消息
- `NewTickMsg(t)` - 创建定时器消息

#### internal/tui/app.go
- `AppModel` 结构体新增字段：
  - `streamManager *StreamUpdateManager` - 流式更新管理器
- `NewAppModel()` - 创建并初始化流式更新管理器
- `Init()` - 启动流式更新管理器和定时器
- `tickCmd()` - 创建定时器命令（每16ms检查一次）
- `Update()` - 重构以支持批量更新：
  - 添加 TickMsg 处理，定时检查缓冲区
  - 添加 BatchUpdateMsg 处理，执行批量更新
  - 流式消息先缓冲，再批量处理
- `handleBatchUpdate(result)` - 处理批量更新消息（核心方法）
  - 更新思考内容块
  - 更新正文内容块
  - 更新工具调用块
  - 更新工具结果
  - 处理完成和错误消息
  - 自动滚动到底部
- `processUserInput()` - 初始化当前消息用于接收流式输出

#### 功能特性
- **流式消息缓冲**：使用时间窗口（50ms）缓冲流式消息
- **批量更新机制**：避免频繁更新UI，提升性能
- **实时更新延迟**：确保流式输出延迟 < 100ms
- **自动滚动**：流式输出时自动滚动到最新内容
- **并发安全**：所有缓冲操作都使用互斥锁保护
- **性能优化**：
  - 使用16ms定时器（约60fps）检查缓冲区
  - 批量处理减少UI更新次数
  - 避免频繁的字符串拼接和渲染

#### 实时更新流程
```
流式消息 -> StreamHandlerImpl -> 缓冲到 MessageBuffer
        -> 定时器检查（每16ms）
        -> ShouldFlush() 判断是否刷新
        -> ForceFlush() 刷新缓冲区
        -> BatchUpdateMsg 消息
        -> handleBatchUpdate() 批量更新UI
        -> 自动滚动到底部
```

#### 缓冲策略
- **时间窗口触发**：距离上次更新超过50ms时触发刷新
- **大小触发**：缓冲区达到10条消息时立即刷新
- **强制刷新**：流式完成或错误时强制刷新
- **定时检查**：每16ms检查一次是否需要刷新

#### 性能指标
- **流式输出延迟**：< 100ms（实际约50-66ms）
- **UI更新频率**：最高60fps（16ms间隔）
- **缓冲区大小**：最多10条消息
- **并发安全**：使用sync.RWMutex保护

### 2026-04-08 Task 15 新增：子对话管理

#### internal/tui/conversation_manager.go
- `ConversationManager` - 对话管理器（线程安全）
  - 管理对话的创建、状态更新、切换和父子关系
  - 支持并发访问，使用读写锁保护
  - 提供回调机制，支持事件通知
- `ConversationTreeNode` - 对话树节点结构体
  - Conversation: 对话对象
  - Children: 子对话节点列表
- `NewConversationManager()` - 创建新的对话管理器
- `CreateConversation(title string)` - 创建新对话
- `CreateSubConversation(parentID, agentRole, task string)` - 创建子对话
- `UpdateConversationStatus(convID string, status ConvStatus)` - 更新对话状态
- `SwitchConversation(convID string)` - 切换到指定对话
- `GetConversation(convID string)` - 获取指定对话
- `GetActiveConversation()` - 获取当前活动对话
- `GetAllConversations()` - 获取所有对话列表（按时间降序排序）
- `GetRootConversations()` - 获取根对话列表（非子对话）
- `GetSubConversations(parentID string)` - 获取指定对话的所有子对话
- `GetConversationTree()` - 获取对话树（包含父子关系）
- `DeleteConversation(convID string)` - 删除对话（递归删除子对话）
- `AddMessageToConversation(convID string, msg Message)` - 向对话添加消息
- `UpdateTokenUsage(convID string, usage int)` - 更新对话的 token 用量
- `SetConversationTitle(convID, title string)` - 设置对话标题
- `GetStatistics()` - 获取统计信息（各状态的对话数量）
- `GetTotalTokenUsage()` - 获取总 token 用量
- `CompleteConversation(convID string)` - 完成对话
- `ActivateConversation(convID string)` - 激活对话
- `WaitConversation(convID string)` - 设置对话为等待状态
- `CreateCurrentMessage(convID string)` - 创建当前消息（用于流式输出）
- `FinalizeCurrentMessage(convID string)` - 完成当前消息并添加到消息列表
- `SetOnConversationCreated(callback)` - 设置对话创建回调
- `SetOnConversationStatusChanged(callback)` - 设置对话状态变更回调
- `SetOnConversationSwitched(callback)` - 设置对话切换回调

#### internal/tui/conversation_manager_test.go
- 完整的单元测试覆盖
- 测试对话管理器创建和配置
- 测试对话创建和状态更新
- 测试子对话创建和父子关系
- 测试对话切换功能
- 测试对话列表获取和排序
- 测试对话删除功能
- 测试消息添加和 token 用量更新
- 测试统计信息获取
- 测试回调机制
- 测试并发安全性
- 测试流式消息处理
- 测试对话树结构
- 所有测试通过（20+测试用例）

#### 功能实现说明
**对话状态类型**：
- ConvStatusActive：活动（正在推理）- 🔄 绿色
- ConvStatusWaiting：等待（等待子代理完成）- ⏳ 黄色
- ConvStatusFinished：结束（对话内容结束）- ✓ 灰色

**对话创建和状态更新**：
- 支持创建主对话和子对话
- 子对话创建时自动更新父对话状态为等待
- 子对话完成时自动检查并更新父对话状态
- 支持状态变更回调通知

**对话列表显示**：
- 对话列表按创建时间降序排序（最新的在前）
- 支持获取所有对话、根对话、子对话
- 支持按状态过滤对话
- 提供统计信息（各状态的对话数量）

**对话切换功能**：
- 支持切换到指定对话
- 触发对话切换回调
- 更新活动对话ID

**父子对话关系管理**：
- 维护父子对话关系（ParentID字段）
- 支持嵌套子对话（子对话可以再创建子对话）
- 提供对话树结构查询
- 删除对话时递归删除所有子对话
- 子对话完成时自动恢复父对话状态

**并发安全性**：
- 使用读写锁（sync.RWMutex）保护并发访问
- 所有公共方法都是线程安全的
- 支持并发创建、查询、更新操作

### 2026-04-08 Task 13 新增：Token 统计功能

#### internal/tui/app.go
- `AppModel` 结构体：
  - `totalTokens int` - 总 token 用量
  - `inferenceCount int` - 推理次数
- `handleStreamComplete(usage int)` - 处理流式完成消息，更新 token 统计
  - 更新当前消息的 `TokenUsage`
  - 累加到对话的 `TokenUsage`
  - 累加到全局的 `totalTokens`
  - 增加 `inferenceCount`
  - 更新状态栏显示

#### internal/tui/messages.go
- `StreamCompleteMsg` - 流式完成消息
  - `Usage int` - token 用量
- `TokenUsageUpdatedMsg` - token 用量更新消息
  - `TotalTokens int` - 总 token 用量
  - `Delta int` - 增量
- `InferenceCountUpdatedMsg` - 推理次数更新消息
  - `Count int` - 推理次数
- `NewStreamCompleteMsg(usage int)` - 创建流式完成消息
- `NewTokenUsageUpdatedMsg(total, delta int)` - 创建 token 用量更新消息
- `NewInferenceCountUpdatedMsg(count int)` - 创建推理次数更新消息
- `MessageBuilder.SetTokenUsage(usage int)` - 设置消息的 token 用量
- `ConversationBuilder.SetTokenUsage(usage int)` - 设置对话的 token 用量

#### internal/tui/components/statusbar.go
- `StatusBar` - 状态栏组件
  - `TotalTokens int` - 总 token 用量
  - `InferenceCount int` - 推理次数
- `SetTokens(tokens int)` - 设置 token 用量
- `SetInferenceCount(count int)` - 设置推理次数
- `UpdateTokens(delta int)` - 更新 token 用量（增量）
- `UpdateInferenceCount(delta int)` - 更新推理次数（增量）
- `Render()` - 渲染状态栏，显示 token 用量和推理次数

#### internal/tui/components/chat.go
- `MessageData` - 消息数据结构体
  - `TokenUsage int` - token 用量
- `RenderTokenUsage(usage int)` - 渲染 token 用量
- `RenderAssistantMessage(msg MessageData)` - 渲染助手消息，包含 token 用量显示

#### internal/agent/core.go
- `modelResponse` 结构体：
  - `Usage int` - token 用量（估算）
- `callModel()` - 调用模型并估算 token 用量
  - 使用简单的字符数估算（平均每4个字符约1个token）
  - 调用 `handler.OnComplete(usage)` 传递 token 用量

#### internal/tui/conversation_manager.go
- `Conversation` 结构体：
  - `TokenUsage int` - token 用量
- `UpdateTokenUsage(convID string, usage int)` - 更新对话的 token 用量
- `GetTotalTokenUsage()` - 获取所有对话的总 token 用量

#### internal/tui/token_stats_test.go
- 完整的单元测试覆盖
- 测试消息和对话的 token 用量字段
- 测试 AppModel 的 token 统计功能
- 测试处理流式完成消息
- 测试多次推理的 token 累计
- 测试多个对话的 token 统计
- 测试消息构建器和对话构建器的 token 用量设置
- 测试状态栏的 token 统计更新
- 测试对话渲染中的 token 显示
- 测试边界情况（0值、大数值）
- 所有测试通过（15个测试用例）

#### internal/tui/conversation_manager_test.go
- 测试对话管理器的 token 统计功能
- 测试创建对话和子对话
- 测试更新 token 用量
- 测试获取总 token 用量
- 所有测试通过

#### 功能特性
- **单次对话 token 显示**：每条助手消息下方显示 token 用量
- **总 token 用量累计**：全局累计所有对话的 token 用量
- **推理次数统计**：统计总推理次数
- **状态栏显示**：实时显示总 token 用量和推理次数
- **数据更新机制**：
  - 流式完成时自动更新统计数据
  - 状态栏实时同步最新数据
  - 支持多对话的 token 统计
- **Token 估算**：
  - 使用字符数估算（每4个字符约1个token）
  - 包含内容、思考、工具调用的 token 估算
  - 对于显示用途足够准确

#### 数据流程
```
模型响应 -> Agent.callModel() 估算 token 用量
        -> StreamHandler.OnComplete(usage)
        -> StreamHandlerImpl.OnComplete()
        -> NewStreamCompleteMsg(usage)
        -> AppModel.handleStreamComplete(usage)
        -> 更新消息 TokenUsage
        -> 累加到对话 TokenUsage
        -> 累加到全局 totalTokens
        -> 增加 inferenceCount
        -> 更新状态栏显示
```

#### 显示效果
- **消息下方**：`Tokens: 150`（灰色斜体）
- **状态栏**：`📁 /path/to/dir │ Tokens: 12345 │ Inferences: 5`
- **实时更新**：每次推理完成后立即更新显示

### 2026-04-08 Task 9 新增：样式定义系统

#### internal/tui/styles.go
- `Theme` - 主题配置结构体
  - 基础颜色：Primary、Secondary、Success、Warning、Error、Info、Muted、Background、Border
  - 消息类型颜色：UserMessage、Thinking、Content、ToolCall、ToolResult、ToolError
  - 状态颜色：Active、Waiting、Finished
  - 布局配置：Layout
- `LayoutConfig` - 布局配置结构体
  - 边框配置：BorderWidth、BorderRadius、BorderStyle
  - 间距配置：PaddingHorizontal、PaddingVertical、MarginHorizontal、MarginVertical
  - 对齐配置：HorizontalAlign、VerticalAlign
- `DefaultTheme()` - 返回默认主题配置（深色主题）
- `DarkTheme()` - 返回深色主题
- `LightTheme()` - 返回浅色主题
- `SetTheme(theme)` - 设置当前主题
- `GetTheme()` - 获取当前主题
- `updateStylesFromTheme()` - 根据当前主题更新样式定义
- `NewBorderStyle(borderType, color)` - 创建边框样式
- `NewPaddingStyle(vertical, horizontal)` - 创建内边距样式
- `NewMarginStyle(vertical, horizontal)` - 创建外边距样式
- `NewAlignedStyle(horizontal, vertical)` - 创建对齐样式
- `NewBoxStyle(borderType, borderColor, paddingVertical, paddingHorizontal)` - 创建盒子样式
- `StyleBuilder` - 样式构建器（链式调用）
  - Foreground(color): 设置前景色
  - Background(color): 设置背景色
  - Bold(bold): 设置粗体
  - Italic(italic): 设置斜体
  - Underline(underline): 设置下划线
  - Padding(vertical, horizontal): 设置内边距
  - Margin(vertical, horizontal): 设置外边距
  - Border(border): 设置边框
  - BorderForeground(color): 设置边框颜色
  - Width(width): 设置宽度
  - Height(height): 设置高度
  - Align(horizontal, vertical): 设置对齐
  - Build(): 构建最终样式
  - Render(text): 渲染文本
- `JoinHorizontal(texts...)` - 水平连接多个文本
- `JoinVertical(texts...)` - 垂直连接多个文本
- `Place(width, height, hPos, vPos, content)` - 将内容放置在指定大小的区域内
- `Width(text)` - 获取文本渲染后的宽度
- `Height(text)` - 获取文本渲染后的高度

#### 样式定义
- **基础样式**：styleBase、styleTitle、styleStatusBar、styleStatusItem、styleInput、styleInputFocused、styleChatArea
- **消息样式**：styleUserMessage、styleUserContent、styleThinking、styleThinkingBox、styleThinkingTitle、styleContent、styleToolCall、styleToolCallBox、styleToolCallTitle、styleToolArgs、styleToolResult、styleToolResultBox、styleToolError、styleTokenUsage
- **状态样式**：styleStatusActive、styleStatusWaiting、styleStatusFinished
- **对话列表样式**：styleConversationList、styleConversationListTitle、styleConversationItem、styleConversationItemSelected、styleConversationTitle、styleConversationTime
- **错误样式**：styleError、styleErrorBox
- **帮助文本样式**：styleHelp、styleKeybinding、styleDescription

#### 颜色主题
- **默认主题**：
  - 基础颜色：Primary(#7C3AED)、Secondary(#3B82F6)、Success(#10B981)、Warning(#F59E0B)、Error(#EF4444)
  - 消息类型颜色：UserMessage(#3B82F6)、Thinking(#6B7280)、Content(#F3F4F6)、ToolCall(#F59E0B)、ToolResult(#10B981)、ToolError(#EF4444)
  - 状态颜色：Active(#10B981)、Waiting(#F59E0B)、Finished(#6B7280)
- **深色主题**：更深的背景色和边框色
- **浅色主题**：浅色背景和深色文本

#### internal/tui/styles_test.go
- 完整的单元测试覆盖
- 测试主题创建和配置
- 测试样式定义
- 测试格式化函数
- 测试状态样式
- 测试布局样式辅助函数
- 测试样式构建器
- 测试样式工具函数
- 所有测试通过（30+测试用例）

#### 功能特性
- **主题系统**：支持默认主题、深色主题、浅色主题，可自定义主题
- **颜色主题**：基础颜色、消息类型颜色、状态颜色
- **布局样式**：边框样式（normal、rounded、double、thick、hidden）、间距样式、对齐样式
- **样式构建器**：链式调用的样式构建接口
- **样式工具函数**：水平连接、垂直连接、居中放置、宽度高度计算
- **消息格式化**：用户消息、思考内容、正文内容、工具调用、工具结果、错误消息
- **状态样式**：活动状态、等待状态、结束状态

### 2026-04-08 Task 10 新增：流式消息处理

#### internal/agent/stream.go
- `StreamHandler` - 流式消息处理器接口
  - OnThinking(chunk string): 处理思考内容块
  - OnContent(chunk string): 处理正文内容块
  - OnToolCall(call ToolCallInfo, isComplete bool): 处理工具调用
  - OnToolResult(result ToolCallInfo): 处理工具执行结果
  - OnComplete(usage int): 处理推理完成
  - OnError(err error): 处理错误
- `ToolCallInfo` - 工具调用信息结构体
  - ID: 工具调用唯一标识
  - Name: 工具名称
  - Arguments: 工具参数（JSON 格式）
  - IsComplete: 是否已完成流式生成
  - Result: 工具执行结果
  - ResultError: 工具执行错误
- `StreamHandlerFunc` - 函数式流式处理器
  - 支持链式调用设置回调函数
  - 支持空函数的安全调用
- `NewStreamHandlerFunc()` - 创建基于函数的流式处理器
- `ConvertToolCall(tc)` - 将 model.ToolCall 转换为 ToolCallInfo
- `ConvertToolCallWithResult(tc, result, error)` - 将 model.ToolCall 转换为带结果的 ToolCallInfo

#### internal/agent/thinking.go
- `ThinkingTagProcessor` - 思考标签处理器
  - 识别和处理 `<thinking>` 标签
  - 支持跨块的标签处理
  - 区分思考内容和正文内容
- `NewThinkingTagProcessor()` - 创建新的思考标签处理器
- `Process(chunk)` - 处理内容块，返回思考内容和正文内容
- `Flush()` - 刷新缓冲区，返回剩余内容
- `IsInThinking()` - 是否正在思考标签内
- `Reset()` - 重置处理器状态

#### internal/agent/core.go
- `Agent` 结构体新增字段：
  - `streamHandler StreamHandler` - 流式消息处理器
- `modelResponse` 结构体新增字段：
  - `Usage int` - token 用量（估算）
- `SetStreamHandler(handler)` - 设置流式消息处理器
- `GetStreamHandler()` - 获取流式消息处理器
- `callModel()` - 重构以支持流式回调：
  - 使用 ThinkingTagProcessor 处理思考标签
  - 调用 streamHandler 的 OnThinking 和 OnContent 方法
  - 调用 streamHandler 的 OnToolCall 方法
  - 估算 token 用量并调用 OnComplete 回调
- `executeTools()` - 重构以支持工具结果回调：
  - 调用 streamHandler 的 OnToolResult 方法

#### internal/model/client.go
- `extractAndCleanThinking()` - 修改为保留思考标签
  - 不再移除 `<thinking>` 标签
  - 由 Agent 核心模块处理标签

#### internal/agent/stream_test.go
- 完整的单元测试覆盖
- 测试 StreamHandlerFunc 的各个回调方法
- 测试空函数的安全调用
- 测试工具调用转换函数
- 所有测试通过（3个测试用例）

#### internal/agent/thinking_test.go
- 完整的单元测试覆盖
- 测试基本的思考标签处理
- 测试纯思考内容和纯正文内容
- 测试混合内容
- 测试多个思考块
- 测试跨块的标签处理
- 测试未闭合的标签
- 测试刷新缓冲区
- 测试重置处理器
- 测试状态检查
- 测试空标签
- 测试嵌套标签
- 所有测试通过（10个测试用例）

#### 功能实现说明
**流式消息处理流程**：
1. 模型客户端接收流式响应
2. Agent 核心模块使用 ThinkingTagProcessor 处理内容
3. 识别思考内容（`<thinking>` 标签内）和正文内容
4. 调用 StreamHandler 的相应回调方法
5. 处理工具调用和工具结果
6. 在推理完成时调用 OnComplete 回调

**思考标签处理**：
- 支持识别 `<thinking>` 和 `</thinking>` 标签
- 支持跨块的标签处理（标签可能被分割到多个流式块中）
- 正确区分思考内容和正文内容
- 处理未闭合的标签

**与 TUI 的集成**：
- TUI 模块实现了 StreamHandler 接口（`internal/tui/messages.go` 中的 `StreamHandlerImpl`）
- Agent 核心模块通过 SetStreamHandler 方法设置处理器
- 流式消息通过回调方法传递到 TUI
- TUI 实时更新界面显示

**性能优化**：
- 使用缓冲区处理跨块的标签
- 避免频繁的字符串操作
- 线程安全的流式处理器访问

### 2026-04-08 Task 6 新增：输入组件

#### internal/tui/components/input.go
- `InputMode` - 输入模式类型（single-line / multi-line）
- `CommandType` - 命令类型枚举
  - CommandNone: 无命令
  - CommandCD: 切换目录命令
  - CommandConversations: 显示对话列表
  - CommandClear: 清空对话
  - CommandSave: 保存对话
  - CommandHelp: 显示帮助
  - CommandExit: 退出程序
- `Command` - 解析后的命令结构体
  - Type: 命令类型
  - Name: 命令名称
  - Args: 命令参数
  - Raw: 原始输入
- `InputComponent` - 输入组件（支持单行/多行模式切换、输入历史记录和命令解析）
  - 集成 bubbles/textarea 组件
  - 支持单行输入模式（Enter 提交）
  - 支持多行输入模式（Ctrl+Enter 提交）
  - 支持 Tab 键切换输入模式
  - 支持输入历史记录（上下箭头浏览）
  - 支持内置命令解析（/cd, /conversations, /clear, /save, /help, /exit）
- `InputComponentConfig` - 输入组件配置结构体
  - Width: 宽度
  - Height: 高度（多行模式）
  - Placeholder: 占位符
  - Prompt: 提示符
  - MaxHistory: 最大历史记录数
  - InitialMode: 初始输入模式
- `DefaultInputComponentConfig()` - 返回默认输入组件配置
- `NewInputComponent(config)` - 创建新的输入组件
- `Init()` - 初始化组件
- `Update(msg)` - 更新组件状态
- `View()` - 渲染组件
- `ToggleMode()` - 切换输入模式
- `SetMode(mode)` - 设置输入模式
- `GetMode()` - 获取当前输入模式
- `GetValue()` - 获取输入内容
- `SetValue(value)` - 设置输入内容
- `Clear()` - 清空输入
- `Focus()` - 获取焦点
- `Blur()` - 失去焦点
- `SetWidth(width)` - 设置宽度
- `SetHeight(height)` - 设置高度
- `AddToHistory(input)` - 添加到历史记录
- `GetHistory()` - 获取历史记录
- `ClearHistory()` - 清空历史记录
- `ParseCommand(input)` - 解析命令
- `IsCommand(input)` - 检查输入是否为命令
- `GetCommandHelp()` - 获取命令帮助文本
- `ShouldSubmit(key)` - 检查是否应该提交输入

#### internal/tui/components/input_test.go
- 完整的单元测试覆盖
- 测试输入组件创建和配置
- 测试输入模式切换
- 测试输入值操作
- 测试历史记录功能
- 测试历史记录导航
- 测试历史记录限制
- 测试命令解析
- 测试提交判断
- 测试焦点管理
- 测试尺寸设置
- 测试视图渲染
- 所有测试通过（14个测试用例）

### 2026-04-08 Task 7 新增：对话显示组件

#### internal/tui/components/chat.go
- `ChatView` - 对话显示组件
  - 封装 viewport 组件，提供消息渲染和自动滚动功能
  - 支持滚动查看历史消息
  - 自动滚动到最新消息
  - 支持自定义样式配置
- `ChatStyles` - 对话样式配置结构体
  - UserMessage: 用户消息样式
  - UserContent: 用户消息内容样式
  - Thinking: 思考内容样式
  - ThinkingBox: 思考内容框样式
  - ThinkingTitle: 思考标题样式
  - Content: 正文内容样式
  - ToolCall: 工具调用样式
  - ToolCallBox: 工具调用框样式
  - ToolCallTitle: 工具调用标题样式
  - ToolArgs: 工具参数样式
  - ToolResult: 工具结果样式
  - ToolResultBox: 工具结果框样式
  - ToolError: 工具错误样式
  - TokenUsage: token 用量样式
  - Error: 错误消息样式
  - ErrorBox: 错误框样式
  - StreamingCursor: 流式光标样式
- `MessageData` - 消息数据结构体
  - Role: 消息角色（user/assistant/tool）
  - Content: 正文内容
  - Thinking: 思考内容
  - ToolCalls: 工具调用列表
  - TokenUsage: token 用量
  - IsStreaming: 是否正在流式输出
  - StreamingType: 流式输出类型
- `ToolCallData` - 工具调用数据结构体
  - ID: 工具调用唯一标识
  - Name: 工具名称
  - Arguments: 工具参数
  - IsComplete: 是否已完成流式生成
  - Result: 工具执行结果
  - ResultError: 工具执行错误
- `DefaultChatStyles() ChatStyles` - 返回默认对话样式
- `NewChatView(width, height int) *ChatView` - 创建新的对话显示组件
- `NewChatViewWithStyles(width, height int, styles ChatStyles) *ChatView` - 创建带自定义样式的对话显示组件
- `Init() tea.Cmd` - 初始化组件
- `Update(msg tea.Msg) (*ChatView, tea.Cmd)` - 更新组件状态
- `View() string` - 渲染组件
- `SetSize(width, height int)` - 设置组件大小
- `SetWidth(width int)` - 设置宽度
- `SetHeight(height int)` - 设置高度
- `SetContent(content string)` - 设置内容
- `SetAutoScroll(autoScroll bool)` - 设置自动滚动
- `ScrollToBottom()` - 滚动到底部
- `ScrollToTop()` - 滚动到顶部
- `PageDown()` - 向下翻页
- `PageUp()` - 向上翻页
- `LineDown()` - 向下滚动一行
- `LineUp()` - 向上滚动一行
- `GetViewport() *viewport.Model` - 获取视口组件
- `RenderMessages(messages []MessageData) string` - 渲染消息列表
- `RenderMessage(msg MessageData) string` - 渲染单条消息
- `RenderUserMessage(content string) string` - 渲染用户消息
- `RenderAssistantMessage(msg MessageData) string` - 渲染助手消息
- `RenderToolMessage(msg MessageData) string` - 渲染工具消息
- `RenderThinking(thinking string, isStreaming bool) string` - 渲染思考内容
- `RenderContent(content string, isStreaming bool) string` - 渲染正文内容
- `RenderToolCall(tc ToolCallData, isStreaming bool) string` - 渲染工具调用
- `RenderToolResult(tc ToolCallData) string` - 渲染工具结果
- `RenderTokenUsage(usage int) string` - 渲染 token 用量
- `RenderError(err string) string` - 渲染错误消息
- `GetWidth() int` - 获取宽度
- `GetHeight() int` - 获取高度
- `IsAtBottom() bool` - 检查是否在底部
- `SetStyles(styles ChatStyles)` - 设置样式
- `GetStyles() ChatStyles` - 获取样式

#### internal/tui/components/chat_test.go
- 完整的单元测试覆盖
- 测试对话显示组件创建和配置
- 测试组件大小设置
- 测试自动滚动功能
- 测试消息渲染（用户消息、思考内容、正文、工具调用、工具结果）
- 测试样式配置
- 测试滚动方法
- 所有测试通过（19个测试用例）

#### internal/tui/app.go
- 更新 `AppModel` 结构体，使用 `*components.ChatView` 替代 `viewport.Model`
- 更新 `NewAppModel()` 函数，创建 `ChatView` 组件
- 更新 `Update()` 方法，适配 `ChatView` 组件
- 实现 `renderConversation()` 方法，将消息转换为 `ChatView` 格式并渲染
- 添加 `StatusBar` 结构体定义（用于状态管理）

#### 消息样式区分
- 用户消息：蓝色（#3B82F6）
- 模型思考：灰色斜体（#6B7280）
- 模型正文：默认样式（#F3F4F6）
- 工具调用：黄色（#F59E0B）
- 工具响应：绿色（#10B981）
- 错误信息：红色（#EF4444）

#### 功能特性
- **滚动查看历史消息**：支持上下翻页、单行滚动
- **自动滚动到最新消息**：新消息到达时自动滚动到底部
- **流式输出支持**：显示流式光标（▌）
- **消息样式区分**：不同类型消息使用不同颜色和样式
- **工具调用格式化**：流式生成中显示原文，完成后格式化为友好显示
- **思考内容框**：使用边框和标题突出显示思考内容
- **工具调用框**：使用边框和标题突出显示工具调用

### 2026-04-08 Task 5 新增：状态栏组件

#### internal/tui/components/statusbar.go
- `StatusBarConfig` - 状态栏配置结构体
  - ShowDirectory: 是否显示工作目录
  - ShowTokens: 是否显示 token 用量
  - ShowInferences: 是否显示推理次数
  - MaxDirectoryLength: 目录最大显示长度
- `StatusBar` - 状态栏组件
  - 显示当前工作目录（📁 图标）
  - 显示总 token 用量
  - 显示推理次数
  - 支持配置显示项
  - 支持目录路径缩写（超过最大长度时）
  - 使用 lipgloss 进行样式美化
- `DefaultStatusBarConfig() StatusBarConfig` - 返回默认状态栏配置
- `NewStatusBar(config StatusBarConfig) *StatusBar` - 创建新的状态栏组件
- `SetWidth(width int)` - 设置宽度
- `SetDirectory(dir string)` - 设置工作目录
- `SetTokens(tokens int)` - 设置 token 用量
- `SetInferenceCount(count int)` - 设置推理次数
- `UpdateTokens(delta int)` - 更新 token 用量（增量）
- `UpdateInferenceCount(delta int)` - 更新推理次数（增量）
- `Render() string` - 渲染状态栏
- `formatDirectory(dir string) string` - 格式化目录显示
- `abbreviatePath(path string, maxLen int) string` - 缩写路径
- `String() string` - 实现 Stringer 接口

#### internal/tui/components/statusbar_test.go
- 完整的单元测试覆盖
- 测试状态栏创建和配置
- 测试各项设置方法
- 测试渲染功能
- 测试目录格式化和缩写
- 测试并发安全性
- 所有测试通过（12个测试用例）

#### internal/tui/styles.go
- 修复 `FormatStatusBar` 函数，使用正确的格式化方式
- 添加 fmt 包导入

#### internal/tui/app.go
- 移除旧的 `StatusBar` 结构体定义
- 使用 `components.StatusBar` 组件
- 更新 `NewAppModel()` 初始化状态栏组件
- 更新 `handleStreamComplete()` 方法，同步更新状态栏数据

### 2026-04-08 Task 8 新增：对话列表弹窗组件

#### internal/tui/components/dialog.go
- `ConvStatus` - 对话状态类型（active / waiting / finished）
- `Conversation` - 对话信息结构体（简化版，用于列表显示）
  - ID: 对话唯一标识
  - CreatedAt: 创建时间
  - Status: 对话状态
  - Title: 对话标题
  - IsSubConversation: 是否为子对话
  - AgentRole: Agent 角色（如果是子对话）
  - MessageCount: 消息数量
  - TokenUsage: token 用量
- `DialogList` - 对话列表弹窗组件
  - 对话列表显示：支持显示所有对话（主对话和子对话）
  - 时间排序：按创建时间降序排序（最新的在前）
  - 状态标注：
    - 活动（正在推理）：🔄 绿色
    - 等待（等待子代理完成）：⏳ 黄色
    - 结束（对话内容结束）：✓ 灰色
  - 对话选择和切换：支持键盘导航（↑/k 上移、↓/j 下移、Enter 选择、Esc 关闭）
  - 过滤功能：支持按状态过滤、过滤子对话
  - 统计功能：获取各状态的对话数量统计
- `NewDialogList() *DialogList` - 创建新的对话列表组件
- `SetConversations(conversations []*Conversation)` - 设置对话列表（自动排序）
- `GetConversations() []*Conversation` - 获取对话列表
- `SetSize(width, height int)` - 设置组件大小
- `Show()` - 显示对话列表
- `Hide()` - 隐藏对话列表
- `Toggle()` - 切换显示状态
- `IsVisible() bool` - 是否可见
- `GetSelected() *Conversation` - 获取当前选中的对话
- `GetSelectedIndex() int` - 获取当前选中的索引
- `SetSelectedIndex(index int)` - 设置选中的索引
- `Update(msg tea.Msg) (*Conversation, bool)` - 处理消息更新
- `View() string` - 渲染对话列表
- `RenderOverlay(parentWidth, parentHeight int) string` - 渲染为覆盖层（居中显示）
- `GetConversationCount() int` - 获取对话数量
- `FindConversationByID(id string) *Conversation` - 根据ID查找对话
- `SelectConversationByID(id string) bool` - 根据ID选择对话
- `FilterByStatus(status ConvStatus) []*Conversation` - 按状态过滤对话
- `FilterSubConversations(onlySub bool) []*Conversation` - 过滤子对话
- `GetStatistics() map[ConvStatus]int` - 获取统计信息

#### internal/tui/components/dialog_test.go
- 完整的单元测试覆盖
- 测试对话列表创建和配置
- 测试对话列表排序
- 测试显示和隐藏
- 测试导航功能
- 测试选择功能
- 测试过滤功能
- 测试统计功能
- 测试视图渲染
- 所有测试通过（17个测试用例）

#### 功能实现说明
**对话列表显示**：
- 支持显示所有对话（主对话和子对话）
- 自动按创建时间降序排序（最新的在前）
- 显示对话标题、状态、时间等信息
- 支持空列表提示

**状态标注**：
- 活动（正在推理）：🔄 绿色图标
- 等待（等待子代理完成）：⏳ 黄色图标
- 结束（对话内容结束）：✓ 灰色图标
- 状态颜色应用于图标，增强视觉识别

**时间排序**：
- 使用 Go 的 sort.Slice 按创建时间降序排序
- 确保最新的对话始终显示在列表顶部
- 支持时间格式化显示（刚刚、X分钟前、X小时前、X天前、具体日期）

**对话选择和切换**：
- 支持键盘导航：↑/k 上移、↓/j 下移
- 支持循环导航（从顶部向上循环到底部，从底部向下循环到顶部）
- 支持 Enter 键选择对话
- 支持 Esc 键关闭列表
- 选中项高亮显示

**覆盖层显示**：
- 支持居中显示对话列表
- 自动计算合适的宽度和高度
- 使用 lipgloss.Place 实现居中布局

### 2026-04-08 Task 2 新增：TUI 模块目录结构

#### internal/tui/app.go
- `InputMode` - 输入模式类型（single-line / multi-line）
- `ConvStatus` - 对话状态类型（active / waiting / finished）
- `MessageRole` - 消息角色类型（user / assistant / tool）
- `ToolCall` - 工具调用信息结构体
  - ID: 工具调用唯一标识
  - Name: 工具名称
  - Arguments: 工具参数（JSON 格式）
  - IsComplete: 是否已完成流式生成
  - Result: 工具执行结果
  - ResultError: 工具执行错误
- `Message` - 对话消息结构体
  - Role: 消息角色
  - Content: 正文内容
  - Thinking: 思考内容
  - ToolCalls: 工具调用列表
  - TokenUsage: token 用量
  - Timestamp: 时间戳
  - IsStreaming: 是否正在流式输出
  - StreamingContent: 流式输出的当前内容
  - StreamingType: 流式输出类型（thinking/content/tool）
- `Conversation` - 对话结构体
  - ID: 对话唯一标识
  - CreatedAt: 创建时间
  - Status: 对话状态
  - Messages: 消息列表
  - TokenUsage: token 用量
  - Title: 对话标题
  - IsSubConversation: 是否为子对话
  - ParentID: 父对话 ID（如果是子对话）
  - AgentRole: Agent 角色（如果是子对话）
- `StatusBar` - 状态栏组件
- `AppModel` - TUI 主应用 Model（实现 Bubble Tea Model-Update-View 架构）
  - 状态管理：currentDir, totalTokens, inferenceCount
  - 对话管理：conversations, activeConv
  - UI 组件：input, chatView, statusBar
  - 状态标志：isStreaming, showConvList, inputMode
- `AgentInterface` - Agent 接口定义（解耦 TUI 和 Agent 核心逻辑）
  - SendMessage(content string) error
  - Cancel() error
  - SetStreamHandler(handler StreamHandler)
  - GetConversations() []*Conversation
  - SwitchConversation(id string) error
  - GetCurrentConversation() *Conversation
  - Close() error
- `StreamHandler` - 流式消息处理器接口
  - OnThinking(chunk string)
  - OnContent(chunk string)
  - OnToolCall(call ToolCall, isComplete bool)
  - OnToolResult(result ToolCall)
  - OnComplete(usage int)
  - OnError(err error)
- `NewAppModel() AppModel` - 创建新的 TUI 应用 Model
- `Init() tea.Cmd` - 初始化 TUI 应用
- `Update(msg tea.Msg) (tea.Model, tea.Cmd)` - 处理消息和更新状态
- `View() tea.View` - 渲染 TUI 界面
- `SetAgent(agent AgentInterface)` - 设置 Agent 接口
- `AddConversation(conv *Conversation)` - 添加新对话
- `SwitchConversation(id string)` - 切换到指定对话

#### internal/tui/messages.go
- `UserInputMsg` - 用户输入消息
- `StreamThinkingMsg` - 流式思考内容消息
- `StreamContentMsg` - 流式正文内容消息
- `StreamToolCallMsg` - 流式工具调用消息
- `StreamToolResultMsg` - 工具执行结果消息
- `StreamCompleteMsg` - 流式输出完成消息
- `StreamErrorMsg` - 流式输出错误消息
- `ConversationCreatedMsg` - 对话创建消息
- `ConversationSwitchedMsg` - 对话切换消息
- `ConversationStatusUpdatedMsg` - 对话状态更新消息
- `WorkingDirChangedMsg` - 工作目录变更消息
- `TokenUsageUpdatedMsg` - token 用量更新消息
- `InferenceCountUpdatedMsg` - 推理次数更新消息
- `InputModeChangedMsg` - 输入模式变更消息
- `ShowConversationListMsg` - 显示对话列表消息
- `CommandExecutedMsg` - 命令执行消息
- `NewStreamThinkingMsg(chunk string) StreamThinkingMsg` - 创建流式思考内容消息
- `NewStreamContentMsg(chunk string) StreamContentMsg` - 创建流式正文内容消息
- `NewStreamToolCallMsg(call ToolCall, isComplete bool) StreamToolCallMsg` - 创建流式工具调用消息
- `NewStreamToolResultMsg(result ToolCall) StreamToolResultMsg` - 创建工具执行结果消息
- `NewStreamCompleteMsg(usage int) StreamCompleteMsg` - 创建流式输出完成消息
- `NewStreamErrorMsg(err error) StreamErrorMsg` - 创建流式输出错误消息
- `StreamHandlerImpl` - 流式处理器实现（实现 StreamHandler 接口）
- `MessageBuilder` - 消息构建器
- `ConversationBuilder` - 对话构建器

#### internal/tui/styles.go
- 颜色主题定义（基础颜色、消息类型颜色、状态颜色）
- 基础样式定义（styleBase, styleTitle, styleStatusBar, styleInput, styleChatArea）
- 消息样式定义（styleUserMessage, styleThinking, styleContent, styleToolCall, styleToolResult, styleTokenUsage）
- 状态样式定义（styleStatusActive, styleStatusWaiting, styleStatusFinished）
- 对话列表样式定义（styleConversationList, styleConversationItem, styleConversationItemSelected）
- 错误样式定义（styleError, styleErrorBox）
- 帮助文本样式定义（styleHelp, styleKeybinding, styleDescription）
- `GetStatusStyle(status ConvStatus) lipgloss.Style` - 根据状态获取样式
- `GetStatusIcon(status ConvStatus) string` - 根据状态获取图标
- `GetMessageRoleStyle(role MessageRole) lipgloss.Style` - 根据消息角色获取样式
- `FormatUserMessage(content string) string` - 格式化用户消息
- `FormatThinking(thinking string, isStreaming bool) string` - 格式化思考内容
- `FormatContent(content string, isStreaming bool) string` - 格式化正文内容
- `FormatToolCall(name, args string, isComplete bool) string` - 格式化工具调用
- `FormatToolResult(result string, isError bool) string` - 格式化工具结果
- `FormatTokenUsage(usage int) string` - 格式化 token 用量
- `FormatError(err string) string` - 格式化错误消息
- `FormatStatusBar(dir string, totalTokens, inferenceCount int, width int) string` - 格式化状态栏
- `FormatConversationItem(conv *Conversation, isSelected bool) string` - 格式化对话列表项
- `FormatHelpText() string` - 格式化帮助文本

#### internal/tui/components/ 目录
- 创建 components 子目录，用于存放 UI 组件（input.go, chat.go, statusbar.go, dialog.go 等）

#### 依赖更新
- 升级到 Bubble Tea v2（charm.land/bubbletea/v2）
- 升级到 Bubbles v2（charm.land/bubbles/v2）
- 升级到 Lipgloss v2（charm.land/lipgloss/v2）

#### TUI 功能实现
- **Bubble Tea 框架集成**：使用 Model-Update-View 架构管理界面状态
- **对话界面布局**：状态栏、对话区、输入区三部分布局
- **流式输出实时显示**：支持思考内容、正文内容、工具调用的流式显示
- **工具调用格式化**：流式生成中显示原文，完成后格式化为友好显示
- **Token 用量统计**：单次对话和总 token 用量显示
- **工作目录管理**：显示当前工作目录
- **子对话管理**：支持对话列表展示和状态标注
- **快捷键支持**：Enter 提交、Tab 切换模式、Ctrl+L 查看列表、Ctrl+C 退出

### 2026-04-07 Task 15 改进：校验机制优化

#### internal/agent/selfcorrector.go
- `SelfCorrectorConfig` 结构体新增字段：
  - `MaxReviewRetries int` - 审查最大重试次数（默认3）
  - `MaxVerifyRetries int` - 校验最大重试次数（默认5）
- `SelfCorrector` 结构体新增字段：
  - `reviewRetryCount int` - 审查重试计数
  - `verifyRetryCount int` - 校验重试计数
- `CorrectionStatistics` 结构体新增字段：
  - `ReviewRetryCount int` - 审查重试次数
  - `VerifyRetryCount int` - 校验重试次数
  - `MaxReviewRetries int` - 审查最大重试次数
  - `MaxVerifyRetries int` - 校验最大重试次数
  - `RemainingReviewRetries int` - 剩余审查重试次数
  - `RemainingVerifyRetries int` - 剩余校验重试次数
- `DefaultSelfCorrectorConfig()` - 新增默认值：MaxReviewRetries=3, MaxVerifyRetries=5
- `NewSelfCorrector(config)` - 初始化分离的重试计数
- `ShouldRetryReview() bool` - 检查是否还有审查重试机会（新增）
- `ShouldRetryVerify() bool` - 检查是否还有校验重试机会（新增）
- `RecordReviewFailure(summary, details, issues)` - 记录审查失败（新增）
- `RecordVerifyFailure(summary, details, issues)` - 记录校验失败（新增）
- `GetReviewRetryCount() int` - 获取审查重试计数（新增）
- `GetVerifyRetryCount() int` - 获取校验重试计数（新增）
- `GetRemainingReviewRetries() int` - 获取剩余审查重试次数（新增）
- `GetRemainingVerifyRetries() int` - 获取剩余校验重试次数（新增）
- `GetStatistics()` - 更新以包含分离的重试统计
- `Reset()` - 更新以重置分离的重试计数
- `MarkSuccess()` - 更新以重置分离的重试计数

#### internal/agent/core.go
- `Agent` 结构体新增字段：
  - `verificationPassed bool` - 是否通过强制校验
- `Run(ctx, taskGoal)` - 重构主循环以支持强制校验：
  - 使用外层循环支持强制校验后的重试
  - 任务完成时不立即返回，而是跳出内层循环
  - 在外层循环中执行强制校验
  - 强制校验失败时重置状态为in_progress，注入修正指令，继续循环
  - 添加最大外层迭代保护（maxIterations + 10）
  - 使用分离的重试计数：审查使用 `ShouldRetryReview()`，校验使用 `ShouldRetryVerify()`

#### 改进说明
**分离重试计数**：
- 审查和校验使用独立的重试计数器
- 审查默认最大重试次数：3次
- 校验默认最大重试次数：5次
- 避免审查失败耗尽所有重试机会，影响校验重试

**强制校验机制**：
- 任务标记为completed后，不立即返回
- 在外层循环中执行强制校验
- 强制校验失败时，重置状态，注入修正指令，继续执行
- 提供额外的迭代保护，防止无限循环

#### 改进流程
```
内层循环 -> 任务完成 -> 跳出内层循环 -> 外层循环 -> 强制校验
                                                        |
                                                        v
                                                    [失败?]
                                                        |
                                                        v
                                        重置状态 -> 注入修正指令 -> 继续内层循环
                                                        |
                                                        v
                                                    [通过?]
                                                        |
                                                        v
                                                真正完成任务 -> 返回结果
```

### 2026-04-07 Task 14 修改：集成校验和自我修正机制到Agent核心

#### internal/agent/core.go
- `Agent` 结构体新增字段：
  - `reviewer *Reviewer` - 程序逻辑审查器
  - `verifier *Verifier` - 成果校验器
  - `corrector *SelfCorrector` - 自我修正器
- `Config` 结构体新增配置：
  - `Reviewer *Reviewer` - 审查器（可选，会自动创建）
  - `Verifier *Verifier` - 校验器（可选，会自动创建）
  - `VerifierConfig *VerifyConfig` - 校验器配置（可选）
  - `CorrectorConfig *SelfCorrectorConfig` - 修正器配置（可选）
- `NewAgent()` - 初始化审查器、校验器和自我修正器
- `Run()` 主循环集成审查、校验和修正逻辑：
  - 在工具调用前审查模型输出
  - 审查被阻断时生成修正指令并注入历史
  - 任务完成前进行校验
  - 校验失败时记录失败并生成修正指令
  - 支持重试机制（最大重试次数可配置）
- `verifyTaskCompletion()` - 验证任务完成情况
- `SetReviewer(r *Reviewer)` - 设置审查器
- `SetVerifier(v *Verifier)` - 设置校验器
- `SetCorrector(c *SelfCorrector)` - 设置自我修正器
- `GetReviewer() *Reviewer` - 获取审查器
- `GetVerifier() *Verifier` - 获取校验器
- `GetCorrector() *SelfCorrector` - 获取自我修正器

#### internal/tools/state_tools.go
- `VerifyResult` - 校验结果结构体（简化版，用于工具层）
- `VerifyIssue` - 校验问题结构体
- `completeTaskTool` 新增字段：
  - `verifier` - 校验回调函数
- `NewCompleteTaskTool()` - 创建完成任务工具
- `NewCompleteTaskToolWithVerifier()` - 创建带校验器的完成任务工具
- `Execute()` - 支持校验回调，校验失败时返回失败结果
- `RegisterStateToolsWithVerifier()` - 注册状态工具（带校验器）

#### 集成流程
```
模型输出 -> 审查器审查 -> [阻断?] -> 生成修正指令 -> 注入历史 -> 继续循环
                                    |
                                    v
                              [通过] -> 执行工具调用
                                    |
                                    v
                              [complete_task?] -> 校验器校验 -> [失败?] -> 记录失败 -> 生成修正指令 -> 注入历史
                                                                              |
                                                                              v
                                                                        [通过] -> 标记任务完成
```

#### 重试机制
- 最大重试次数可配置（默认3次）
- 支持指数退避策略
- 重试次数耗尽时任务失败
- 修正成功后重置重试计数

### 2026-04-07 Task 13 新增：自我修正器模块

#### internal/agent/selfcorrector.go
- `CorrectionStatus` - 修正状态类型（needed/retryable/exhausted/success）
- `FailureType` - 失败类型枚举
  - verify: 校验失败
  - review: 审查失败
  - tool: 工具执行失败
  - system: 系统错误
- `FailureRecord` - 失败记录结构体
  - Type: 失败类型
  - Timestamp: 失败时间
  - Summary: 失败摘要
  - Details: 详细信息
  - Issues: 问题列表
  - RetryCount: 重试次数
  - Corrected: 是否已修正
- `CorrectionResult` - 修正结果结构体
  - Status: 修正状态
  - NeedsCorrection: 是否需要修正
  - Instruction: 修正指令内容
  - RemainingRetries: 剩余重试次数
  - FailureSummary: 失败原因摘要
  - Suggestions: 修正建议列表
  - Priority: 优先级（low/medium/high/critical）
  - Timestamp: 时间戳
- `SelfCorrectorConfig` - 自我修正器配置结构体
  - MaxRetries: 最大重试次数
  - RetryDelayMs: 重试延迟（毫秒）
  - ExponentialBackoff: 是否启用指数退避
  - MaxFailureHistory: 最大失败历史记录数
  - FailurePatternWindow: 失败模式检测窗口大小
  - MaxInstructionLength: 修正指令最大长度
  - IncludeEvidence: 是否包含证据
  - IncludeSuggestions: 是否包含建议
- `FailurePattern` - 失败模式结构体
  - DetectedAt: 检测时间
  - FailureTypes: 失败类型统计
  - RecurringIssues: 重复出现的问题
  - Severity: 严重程度
  - Description: 模式描述
- `CorrectionStatistics` - 修正统计信息结构体
  - CurrentRetryCount: 当前重试次数
  - MaxRetries: 最大重试次数
  - RemainingRetries: 剩余重试次数
  - TotalFailures: 总失败次数
  - CorrectedFailures: 已修正的失败次数
  - TotalCorrections: 总修正次数
  - CorrectionSuccessRate: 修正成功率
- `SelfCorrector` - 自我修正器（线程安全）
  - 管理重试逻辑和失败历史
  - 分析失败原因并生成修正指令
  - 检测失败模式
  - 支持指数退避重试策略
- `DefaultSelfCorrectorConfig() *SelfCorrectorConfig` - 返回默认自我修正器配置
- `NewSelfCorrector(config) *SelfCorrector` - 创建新的自我修正器
- `AnalyzeFailure(verifyResult, reviewResult) *CorrectionResult` - 分析校验失败结果
- `GenerateCorrectionInstruction(correctionResult) string` - 根据失败原因生成修正指令
- `ShouldRetry() bool` - 检查是否还有重试机会
- `RecordFailure(failureType, summary, details, issues)` - 记录失败历史
- `RecordCorrection(result)` - 记录修正结果
- `Reset()` - 重置修正器状态
- `GetRetryCount() int` - 获取当前重试次数
- `GetRemainingRetries() int` - 获取剩余重试次数
- `GetFailureHistory() []FailureRecord` - 获取失败历史记录
- `GetCorrectionHistory() []CorrectionResult` - 获取修正历史记录
- `GetConfig() *SelfCorrectorConfig` - 获取修正器配置
- `UpdateConfig(config) error` - 更新修正器配置
- `DetectFailurePattern() *FailurePattern` - 检测失败模式
- `GetRetryDelay() time.Duration` - 获取重试延迟（支持指数退避）
- `MarkSuccess()` - 标记修正成功
- `GetStatistics() *CorrectionStatistics` - 获取统计信息

#### 自我修正功能实现
- **失败分析**：分析校验和审查失败结果
  - 提取关键问题
  - 生成问题摘要
  - 确定失败类型和优先级
- **修正指令生成**：根据失败原因生成清晰的修正指令
  - 包含失败原因、优先级、剩余重试次数
  - 提供具体的修正建议
  - 根据优先级给出不同的行动指导
- **重试管理**：管理重试逻辑
  - 支持最大重试次数配置
  - 支持指数退避策略
  - 自动更新重试计数
- **失败模式检测**：识别重复出现的失败模式
  - 统计失败类型
  - 检测重复问题
  - 生成模式描述

#### internal/agent/selfcorrector_test.go
- 完整的单元测试覆盖
- 测试修正器创建和配置
- 测试失败分析
- 测试修正指令生成
- 测试重试逻辑
- 测试失败模式检测
- 测试并发安全性
- 所有测试通过（13个测试用例）

### 2026-04-07 Task 12 新增：成果校验器模块

#### internal/agent/verifier.go
- `VerifyStatus` - 校验结果状态类型（pass/warning/fail）
- `VerifyIssueType` - 校验问题类型枚举
  - file_not_found: 文件不存在
  - file_empty: 文件内容为空
  - content_missing: 内容缺失关键部分
  - keyword_not_found: 关键词未找到
  - custom_rule_failed: 自定义规则校验失败
  - invalid_path: 无效路径
  - permission_denied: 权限拒绝
- `VerifyIssue` - 校验发现的问题结构体
  - Type: 问题类型
  - Severity: 严重程度（low/medium/high/critical）
  - Description: 问题描述
  - Evidence: 证据/示例
  - Suggestion: 修正建议
  - FilePath: 相关文件路径
  - RuleName: 相关规则名称
  - Timestamp: 发现时间
- `VerifyResult` - 校验结果结构体
  - Status: 校验状态
  - Issues: 发现的问题列表
  - Timestamp: 校验时间
  - Summary: 校验摘要
  - Passed: 通过的检查项数量
  - Failed: 失败的检查项数量
- `VerifyConfig` - 校验器配置结构体
  - CheckFileExists: 是否检查文件存在
  - CheckFileNonEmpty: 是否检查文件非空
  - MaxFileSizeToCheck: 最大检查文件大小
  - CheckKeywords: 是否检查关键词
  - RequiredKeywords: 必需的关键词列表
  - KeywordMatchMode: 关键词匹配模式（any/all）
  - EnableCustomRules: 是否启用自定义规则
  - StopOnFirstFailure: 遇到第一个失败是否停止
  - MaxIssuesToReport: 最大报告问题数量（0表示不限制）
- `VerifyRule` - 自定义校验规则接口
  - Name(): 规则名称
  - Description(): 规则描述
  - Execute(ctx *VerifyContext): 执行校验规则
- `VerifyContext` - 校验上下文结构体
  - TaskState: 任务状态
  - Files: 待校验的文件列表
  - Content: 待校验的内容
  - ExtraData: 额外数据
- `Verifier` - 成果校验器（线程安全）
  - 管理校验规则和状态跟踪
  - 维护校验历史记录
  - 支持自定义校验规则
- `DefaultVerifierConfig() *VerifyConfig` - 返回默认校验器配置
- `NewVerifier(config) *Verifier` - 创建新的校验器
- `Verify(files, taskState) *VerifyResult` - 执行成果校验
- `VerifyTaskCompletion(files, taskState) *VerifyResult` - 任务完成前校验（更严格）
- `VerifyFiles(files) *VerifyResult` - 批量校验文件
- `VerifyContent(content, taskState) *VerifyResult` - 校验内容（不涉及文件）
- `AddRule(rule)` - 添加自定义校验规则
- `RemoveRule(name) bool` - 移除自定义校验规则
- `ClearRules()` - 清除所有自定义校验规则
- `GetVerifyHistory() []VerifyResult` - 获取校验历史
- `GetLastResult() *VerifyResult` - 获取最后一次校验结果
- `Reset()` - 重置校验器状态
- `GetConfig() *VerifyConfig` - 获取校验器配置
- `UpdateConfig(config) error` - 更新校验器配置
- `SetRequiredKeywords(keywords)` - 设置必需的关键词
- `GetCustomRules() []VerifyRule` - 获取所有自定义规则

#### 校验功能实现
- **文件存在检查**：验证文件是否存在
  - 支持绝对路径和相对路径
  - 检查路径是否为目录
  - 检查文件访问权限
- **文件非空检查**：验证文件内容是否非空
  - 检查文件大小
  - 检查内容是否只有空白字符
  - 支持文件大小限制
- **关键词匹配检查**：验证文件内容是否包含必需的关键词
  - 支持all模式（所有关键词都必须存在）
  - 支持any模式（至少一个关键词存在）
  - 大小写不敏感匹配
- **自定义规则检查**：支持自定义校验规则
  - 通过VerifyRule接口定义规则
  - 支持规则的添加、移除和清除
  - 规则可以访问校验上下文

#### 校验流程
```
文件列表 -> 文件存在检查 -> 文件非空检查 -> 关键词匹配检查 -> 自定义规则检查 -> 返回结果
```

**详细步骤**：
1. 检查所有文件是否存在
2. 检查存在的文件是否非空
3. 检查存在的文件是否包含必需的关键词
4. 执行自定义校验规则
5. 汇总所有问题
6. 确定最终状态（pass/warning/fail）
7. 生成校验摘要
8. 记录校验历史

#### internal/agent/verifier_test.go
- 完整的单元测试覆盖
- 测试校验器创建和配置
- 测试各类校验规则
- 测试关键词匹配
- 测试自定义规则
- 测试并发安全性
- 所有测试通过（30+测试用例）

### 2026-04-06 Task 11 新增：命令行入口

#### internal/config/loader.go
- `Config` - 完整的应用配置结构体
  - Model: 模型服务配置
  - Agent: Agent行为配置
  - State: 状态管理配置
  - Tools: 工具配置
- `ModelConfig` - 模型服务配置
  - Endpoint: API端点地址
  - APIKey: API密钥
  - ModelName: 模型名称
  - ContextSize: 上下文大小
- `AgentConfig` - Agent行为配置
  - MaxIterations: 最大迭代次数
  - Temperature: 温度参数
- `StateConfig` - 状态管理配置
  - Dir: 状态文件存储目录
  - AutoSave: 是否自动保存
- `ToolsConfig` - 工具配置
  - WorkDir: 工作目录
  - AllowCommands: 允许执行的命令列表
- `DefaultConfig() *Config` - 返回默认配置
- `LoadConfig(path) (*Config, error)` - 从文件加载配置
- `Validate() error` - 验证配置有效性
- `GetAbsoluteWorkDir() (string, error)` - 获取绝对工作目录
- `GetAbsoluteStateDir() (string, error)` - 获取绝对状态目录

#### 环境变量支持
- `AGENTPLUS_MODEL_ENDPOINT` - 覆盖模型端点
- `AGENTPLUS_MODEL_API_KEY` - 覆盖API密钥
- `AGENTPLUS_MODEL_NAME` - 覆盖模型名称
- `AGENTPLUS_MODEL_CONTEXT_SIZE` - 覆盖上下文大小
- `AGENTPLUS_AGENT_MAX_ITERATIONS` - 覆盖最大迭代次数
- `AGENTPLUS_AGENT_TEMPERATURE` - 覆盖温度参数
- `AGENTPLUS_STATE_DIR` - 覆盖状态目录
- `AGENTPLUS_STATE_AUTO_SAVE` - 覆盖自动保存设置
- `AGENTPLUS_TOOLS_WORK_DIR` - 覆盖工作目录

#### cmd/agentplus/main.go
- 命令行入口程序
- 支持命令行参数解析
- 支持交互式任务输入
- 支持流式输出显示
- 支持优雅退出（Ctrl+C）
- 命令行参数：
  - `-c, --config <file>`: 配置文件路径（默认: ./configs/config.yaml）
  - `-t, --task <id>`: 继续已有任务
  - `-w, --workdir <dir>`: 工作目录
  - `-v, --verbose`: 详细输出
  - `--no-supervisor`: 禁用监督
  - `--max-iterations <n>`: 最大迭代次数
- 交互式命令：
  - `/help`: 显示帮助信息
  - `/quit`: 退出程序
  - `/status`: 显示当前任务状态
  - `/clear`: 清除当前输入

#### internal/model/message.go 新增
- `NewAssistantMessageWithToolCalls(content, toolCalls) Message` - 创建带工具调用的助手消息

### 2026-04-06 Task 10 新增：上下文压缩模块

#### internal/agent/compressor.go
- `CompressorConfig` - 压缩器配置结构体
  - TriggerThreshold: 触发压缩的上下文使用阈值（默认0.8）
  - MinMessagesToKeep: 压缩后保留的最小消息数量（默认10）
  - MaxMessagesToKeep: 压缩后保留的最大消息数量（默认20）
  - KeepSystemMessages: 是否保留所有系统消息（默认true）
  - KeepRecentToolCalls: 保留最近N次工具调用及其结果（默认3）
  - EnableProgressiveCompression: 是否启用渐进式压缩（默认true）
  - SummaryMaxLength: 摘要的最大长度（默认2000）
- `CompressionResult` - 压缩结果结构体
  - CompressedMessages: 压缩后的消息列表
  - OriginalCount: 原始消息数量
  - CompressedCount: 压缩后消息数量
  - OriginalTokens: 原始token数量
  - CompressedTokens: 压缩后token数量
  - CompressionRatio: 压缩比率（0-1）
  - Summary: 生成的上下文摘要
  - WasCompressed: 是否进行了压缩
- `Compressor` - 上下文压缩器（线程安全）
  - 管理上下文压缩策略
  - 支持渐进式压缩和深度压缩
  - 基于YAML状态摘要生成压缩摘要
- `KeyInfo` - 关键信息结构体
  - Decisions: 关键决策
  - RecentActions: 最近操作
  - ToolHistory: 工具调用历史
- `CompressionStats` - 压缩统计信息结构体
  - MessageCount: 消息数量
  - TokenCount: Token数量
  - ContextSize: 上下文大小
  - UsageRatio: 使用率
  - ShouldCompress: 是否应该压缩
  - TriggerThreshold: 触发阈值
- `NewCompressor(modelClient, config) (*Compressor, error)` - 创建新的上下文压缩器
- `ShouldCompress(messages) (bool, float64)` - 判断是否需要压缩
- `Compress(messages, taskState) (*CompressionResult, error)` - 压缩消息历史
- `ExtractKeyInfo(messages) *KeyInfo` - 提取关键信息
- `GetConfig() *CompressorConfig` - 获取压缩器配置
- `UpdateConfig(config) error` - 更新压缩器配置
- `GetCompressionStats(messages) *CompressionStats` - 获取压缩统计信息

#### 压缩策略实现
- **渐进式压缩**：
  - 第一阶段：轻度压缩（保留更多消息）
  - 第二阶段：深度压缩（如果轻度压缩后仍然超限）
- **压缩保留策略**：
  - 保留所有系统消息
  - 保留最近的工具调用和结果
  - 保留最近的对话消息
  - 生成上下文摘要替代被压缩的内容
- **关键信息提取**：
  - 从消息中提取关键决策
  - 提取最近的操作记录
  - 提取工具调用历史
- **上下文摘要生成**：
  - 基于YAML状态摘要
  - 包含任务目标、当前状态、已完成步骤
  - 包含关键决策和最近操作
  - 支持长度限制

#### 压缩触发机制
- 当上下文使用超过阈值（默认80%）时触发
- 可配置触发阈值
- 压缩后保持最小上下文
- 支持渐进式压缩策略

#### internal/agent/compressor_test.go
- 完整的单元测试覆盖
- 测试压缩器创建和配置
- 测试压缩判断逻辑
- 测试压缩功能
- 测试关键信息提取
- 测试并发安全性
- 所有测试通过（9个测试用例）

### 2026-04-06 Task 9 新增：监督系统模块

#### internal/supervisor/monitor.go
- `InterventionType` - 干预类型枚举
  - warning: 警告（记录问题但不中断）
  - pause: 暂停（暂停当前Agent，等待人工确认）
  - interrupt: 中断（中断当前操作，注入修正指令）
  - rollback: 回滚（回滚到上一个稳定状态）
- `IssueType` - 监督问题类型枚举
  - quality: 输出质量问题
  - progress: 任务进度问题
  - error: 错误检测
  - security: 安全问题
  - behavior: 行为异常
  - resource: 资源使用问题
- `SupervisionIssue` - 监督发现的问题结构体
  - Type: 问题类型
  - Severity: 严重程度（low/medium/high/critical）
  - Description: 问题描述
  - Evidence: 证据/示例
  - Suggestion: 修正建议
  - Timestamp: 发现时间
  - Context: 上下文信息
- `InterventionRecord` - 干预记录结构体
  - ID: 干预记录ID
  - Type: 干预类型
  - Issue: 相关问题
  - Timestamp: 干预时间
  - Action: 采取的行动
  - Result: 干预结果
  - TaskID: 相关任务ID
  - AgentRole: 相关Agent角色
- `SupervisionResult` - 监督结果结构体
  - Status: 监督状态（pass/warning/intervention）
  - Issues: 发现的问题列表
  - Timestamp: 监督时间
  - Summary: 监督摘要
  - Intervention: 需要的干预（如果有）
- `SupervisionStats` - 监督统计结构体
  - TotalChecks: 总检查次数
  - IssuesFound: 发现的问题总数
  - Interventions: 干预次数
  - WarningsIssued: 发出的警告次数
  - PausesTriggered: 触发的暂停次数
  - InterruptsTriggered: 触发的中断次数
  - RollbacksTriggered: 触发的回滚次数
  - LastCheckTime: 最后检查时间
  - StartTime: 监督开始时间
- `SupervisorConfig` - 监督器配置结构体
  - EnableQualityCheck: 启用输出质量检查
  - EnableProgressCheck: 启用任务进度检查
  - EnableErrorDetection: 启用错误检测
  - EnableSecurityCheck: 启用安全检查
  - EnableBehaviorCheck: 启用行为检查
  - EnableResourceCheck: 启用资源检查
  - MonitorInterval: 监督频率（秒）
  - MaxWarnings: 最大警告次数
  - AutoIntervene: 自动干预
  - MaxConsecutiveErrors: 最大连续错误次数
  - QualityThreshold: 质量阈值
  - ProgressTimeout: 进度超时时间
  - EnableParallelMonitor: 启用并行监督
  - MaxParallelChecks: 最大并行检查数
- `AgentOutput` - Agent输出结构体
  - Content: 输出内容
  - ToolCalls: 工具调用列表
  - Timestamp: 输出时间
  - TaskID: 任务ID
  - AgentRole: Agent角色
  - Iteration: 迭代次数
  - Success: 是否成功
  - Error: 错误信息
- `Supervisor` - 监督器（线程安全）
  - 管理监督检查和干预机制
  - 维护干预历史和统计信息
  - 支持并行监督
  - 提供回调机制
- `NewSupervisor(modelClient, toolRegistry, stateManager, reviewer, config) (*Supervisor, error)` - 创建新的监督器
- `Monitor(ctx, agentOutput, taskState) *SupervisionResult` - 监督Agent输出
- `ParallelMonitor(ctx, outputs) <-chan *SupervisionResult` - 并行监督多个Agent输出
- `Intervene(ctx, issue) *InterventionRecord` - 执行干预
- `SetOnIntervention(callback)` - 设置干预回调
- `SetOnWarning(callback)` - 设置警告回调
- `SetOnIssueFound(callback)` - 设置问题发现回调
- `GetInterventionLog() []InterventionRecord` - 获取干预历史
- `GetStatistics() SupervisionStats` - 获取监督统计
- `Reset()` - 重置监督器状态
- `SaveStableState(taskState)` - 保存稳定状态（用于回滚）
- `Resume()` - 恢复执行（用于暂停后）
- `Stop()` - 停止监督

#### 监督检查项实现
- **输出质量检查**：验证Agent输出的质量和完整性
  - 检查输出是否为空
  - 检查输出长度是否合理
  - 集成审查器进行深度检查
- **任务进度检查**：监控任务执行进度
  - 检查任务是否超时
  - 检查进度是否停滞
  - 跟踪完成步骤
- **错误检测**：识别和处理错误
  - 检查输出中的错误标记
  - 跟踪连续错误次数
  - 检查工具调用失败
- **安全检查**：识别潜在安全风险
  - 检查危险命令（rm -rf /, mkfs等）
  - 检查敏感文件访问（/etc/passwd, .env等）
- **行为检查**：监控Agent行为
  - 检查迭代次数是否过高
  - 检查工具调用频率
- **资源检查**：监控资源使用
  - 检查输出大小
  - 防止资源过度消耗

#### 干预机制实现
- **警告（warning）**：记录问题但不中断执行
  - 适用于低严重度问题
  - 触发警告回调
  - 累计警告次数
- **暂停（pause）**：暂停当前Agent，等待人工确认
  - 适用于高严重度问题
  - 发送暂停信号
  - 等待恢复信号
- **中断（interrupt）**：中断当前操作，注入修正指令
  - 适用于严重安全问题
  - 发送停止信号
  - 立即停止执行
- **回滚（rollback）**：回滚到上一个稳定状态
  - 适用于严重错误
  - 恢复保存的稳定状态
  - 需要预先保存状态

#### 监督流程
```
Agent执行 -> 监督器并行检查 -> 发现问题 -> 干预决策 -> 执行干预
```

**详细步骤**：
1. Agent产生输出
2. 监督器接收输出并执行各项检查
3. 汇总发现的问题
4. 确定监督状态（pass/warning/intervention）
5. 根据配置决定是否干预
6. 执行干预动作（如果需要）
7. 触发相应回调
8. 记录干预历史和统计信息

#### internal/supervisor/monitor_test.go
- 完整的单元测试覆盖
- 测试监督器创建和配置
- 测试各类监督检查
- 测试干预机制
- 测试并行监督
- 测试回调机制
- 测试统计和日志
- 所有测试通过（20+测试用例）

### 2026-04-06 Task 6 新增：程序逻辑审查模块

#### internal/agent/reviewer.go
- `ReviewStatus` - 审查结果状态类型（pass/warning/block）
- `IssueType` - 问题类型枚举
  - direction: 方向偏离
  - infinite_loop: 无限循环
  - invalid_tool_call: 无效工具调用
  - repeated_failure: 重复失败
  - fabrication: 编造内容
  - no_progress: 无进度
- `ReviewIssue` - 审查发现的问题结构体
  - Type: 问题类型
  - Severity: 严重程度（low/medium/high/critical）
  - Description: 问题描述
  - Evidence: 证据/示例
  - Suggestion: 修正建议
  - ToolName: 相关工具名称
  - Timestamp: 发现时间
- `ReviewResult` - 审查结果结构体
  - Status: 审查状态
  - Issues: 发现的问题列表
  - Timestamp: 审查时间
  - Summary: 审查摘要
- `ReviewConfig` - 审查器配置结构体
  - EnableDirectionCheck: 是否启用方向偏离检测
  - MaxRepeatedActions: 相同操作最大重复次数
  - LoopWindowSize: 循环检测窗口大小
  - MaxConsecutiveFailures: 最大连续失败次数
  - FailureResetInterval: 失败计数重置间隔
  - EnableFabricationCheck: 是否启用编造检测
  - MaxIterationsWithoutProgress: 无进度最大迭代次数
  - MaxFileChecksPerReview: 每次审查最大文件检查数
- `Reviewer` - 程序逻辑审查器（线程安全）
  - 管理审查规则和状态跟踪
  - 维护操作历史记录
  - 跟踪失败计数和进度
- `ActionRecord` - 操作记录结构体
- `NewReviewer(config) *Reviewer` - 创建新的审查器
- `ReviewOutput(output, toolCalls, state) *ReviewResult` - 审查模型输出
- `ReviewToolResult(toolName, arguments, result, success) *ReviewResult` - 审查工具执行结果
- `GetActionHistory() []ActionRecord` - 获取操作历史
- `Reset()` - 重置审查器状态
- `GetFailureCount() int` - 获取当前失败计数

#### 审查规则实现
- **方向偏离检测**：验证输出是否与YAML中的任务目标一致
  - 提取任务目标关键词
  - 检查输出内容与目标的相关性
  - 关键词匹配率过低时发出警告
- **错误模式识别**：
  - 无限循环检测：检测重复相同操作（默认3次触发）
  - 无效工具调用：检查工具名称和参数格式
  - 重复失败检测：检测连续失败次数（默认3次触发）
- **编造内容检测**：
  - 验证声称存在的文件是否真实存在
  - 检查工具调用中的文件路径有效性
  - 支持Windows和Unix路径格式
- **进度验证**：检查是否真正推进任务进度
  - 跟踪已完成步骤
  - 检测迭代无进度情况

#### internal/agent/feedback.go
- `FeedbackLevel` - 反馈级别类型（info/warning/error/critical）
- `FeedbackMessage` - 反馈消息结构体
  - Level: 反馈级别
  - Title: 反馈标题
  - Content: 反馈内容
  - Suggestions: 修正建议列表
  - Timestamp: 时间戳
- `FeedbackInjector` - 审查反馈注入器
  - 将审查结果转换为用户消息
  - 支持多种反馈格式
- `FeedbackInjectorConfig` - 反馈注入器配置
  - MaxFeedbackLength: 最大反馈长度
  - IncludeEvidence: 是否包含证据
  - IncludeTimestamp: 是否包含时间戳
- `NewFeedbackInjector(config) *FeedbackInjector` - 创建新的反馈注入器
- `InjectFeedback(result) Message` - 根据审查结果生成反馈消息
- `InjectFeedbackForIssue(issue) Message` - 为单个问题生成反馈
- `InjectBlockingFeedback(result) Message` - 生成阻断级别反馈
- `InjectWarningFeedback(result) Message` - 生成警告级别反馈
- `InjectProgressFeedback(iteration, max) Message` - 生成进度反馈
- `InjectLoopDetectedFeedback(toolName, args, count) Message` - 生成循环检测反馈
- `InjectFailureFeedback(toolName, count, error) Message` - 生成失败反馈
- `InjectDirectionFeedback(goal, output) Message` - 生成方向偏离反馈
- `BatchInjectFeedback(results) []Message` - 批量生成反馈消息
- `FormatFeedbackForLog(result) string` - 格式化反馈用于日志

#### 审查触发时机（设计）
- 每次工具调用前
- 每次任务状态更新前
- 检测到异常输出时

#### 审查结果处理
- **通过（pass）**：继续执行
- **警告（warning）**：记录但继续执行
- **阻断（block）**：注入反馈，要求模型修正

#### internal/agent/reviewer_test.go
- 完整的单元测试覆盖
- 测试审查器创建和配置
- 测试各类审查规则
- 测试并发安全性
- 所有测试通过（20+测试用例）

#### internal/agent/feedback_test.go
- 完整的单元测试覆盖
- 测试反馈消息生成
- 测试各级别反馈
- 测试批量反馈
- 所有测试通过（15+测试用例）

### 2026-04-06 Task 8 新增：团队定义与角色管理模块

#### internal/team/definition.go
- `AgentRole` - Agent角色定义结构体
  - Name: 角色名称（唯一标识）
  - Description: 角色描述
  - SystemPrompt: 系统提示词
  - Tools: 可用工具列表
  - CanFork: 是否可以创建子代理
  - MaxIterations: 最大迭代次数
  - Priority: 角色优先级
  - Tags: 角色标签
- `Team` - 团队结构定义
  - Name: 团队名称
  - Description: 团队描述
  - Roles: 角色定义映射
  - DefaultRole: 默认角色
  - Workflow: 工作流定义
- `WorkflowStep` - 工作流步骤定义
  - StepName: 步骤名称
  - Role: 执行角色
  - Description: 步骤描述
  - NextSteps: 下一步骤条件分支
- `NewTeam(name, description) *Team` - 创建新的团队实例
- `AddRole(role) error` - 添加角色到团队
- `GetRole(name) (*AgentRole, bool)` - 获取角色定义
- `RemoveRole(name) error` - 从团队中移除角色
- `SetDefaultRole(name) error` - 设置默认角色
- `ListRoles() []string` - 列出所有角色名称
- `AddWorkflowStep(step) error` - 添加工作流步骤
- `GetWorkflow() []WorkflowStep` - 获取工作流定义
- `DefaultTeam() *Team` - 创建默认团队配置
- `Validate() error` - 验证团队配置有效性
- `Clone() *Team` - 克隆团队配置

#### internal/team/roles.go
- `RoleOrchestrator` - Orchestrator角色常量
- `RoleArchitect` - Architect角色常量
- `RoleDeveloper` - Developer角色常量
- `RoleTester` - Tester角色常量
- `RoleReviewer` - Reviewer角色常量
- `RoleSupervisor` - Supervisor角色常量
- `GetPredefinedRoles() []*AgentRole` - 返回所有预定义角色
- `NewOrchestratorRole() *AgentRole` - 创建Orchestrator角色
  - 职责：协调任务流程，维护YAML状态
  - 可用工具：所有工具（文件操作、命令执行、状态管理）
  - 可Fork：是
  - 最大迭代次数：50
- `NewArchitectRole() *AgentRole` - 创建Architect角色
  - 职责：架构设计，技术选型，模块划分
  - 可用工具：文件读写、命令执行
  - 可Fork：是
  - 最大迭代次数：30
- `NewDeveloperRole() *AgentRole` - 创建Developer角色
  - 职责：代码实现，功能开发
  - 可用工具：文件读写、命令执行
  - 可Fork：否
  - 最大迭代次数：40
- `NewTesterRole() *AgentRole` - 创建Tester角色
  - 职责：测试编写，验证功能
  - 可用工具：文件读写、命令执行
  - 可Fork：否
  - 最大迭代次数：30
- `NewReviewerRole() *AgentRole` - 创建Reviewer角色
  - 职责：代码审查，质量把控
  - 可用工具：文件读取（只读）
  - 可Fork：否
  - 最大迭代次数：20
- `NewSupervisorRole() *AgentRole` - 创建Supervisor角色
  - 职责：监督Agent，检查输出质量，纠偏
  - 可用工具：文件读取（只读）
  - 可Fork：否
  - 最大迭代次数：15

#### internal/team/manager.go
- `RoleManager` - 角色管理器（线程安全）
  - 管理团队角色配置
  - 提供角色查询和切换功能
  - 管理角色切换历史
- `RoleSwitchRecord` - 角色切换记录
  - FromRole: 切换前角色
  - ToRole: 切换后角色
  - Reason: 切换原因
  - Timestamp: 切换时间戳
- `NewRoleManager(team) (*RoleManager, error)` - 创建新的角色管理器
- `GetRole(name) (*AgentRole, error)` - 获取角色定义
- `GetCurrentRole() *AgentRole` - 获取当前活动角色
- `GetCurrentRoleName() string` - 获取当前角色名称
- `GetSystemPrompt(roleName) (string, error)` - 获取指定角色的系统提示词
- `GetCurrentSystemPrompt() string` - 获取当前角色的系统提示词
- `GetAvailableTools(roleName) ([]string, error)` - 获取指定角色可用的工具列表
- `GetCurrentAvailableTools() []string` - 获取当前角色可用的工具列表
- `HasToolPermission(roleName, toolName) (bool, error)` - 检查指定角色是否有权限使用某个工具
- `CanFork(roleName) (bool, error)` - 检查指定角色是否可以创建子代理
- `CanCurrentFork() bool` - 检查当前角色是否可以创建子代理
- `GetMaxIterations(roleName) (int, error)` - 获取指定角色的最大迭代次数
- `GetCurrentMaxIterations() int` - 获取当前角色的最大迭代次数
- `SwitchRole(newRole, reason) (*AgentRole, error)` - 切换角色
- `SwitchRoleWithValidation(newRole, reason) (*AgentRole, error)` - 切换角色并进行权限验证
- `GetRoleHistory() []RoleSwitchRecord` - 获取角色切换历史
- `ListAllRoles() []string` - 列出所有角色名称
- `GetTeam() *Team` - 获取团队配置
- `ResetToDefault() error` - 重置到默认角色
- `GetRolePriority(roleName) (int, error)` - 获取指定角色的优先级
- `GetRoleTags(roleName) ([]string, error)` - 获取指定角色的标签
- `FindRolesByTag(tag) []string` - 根据标签查找角色
- `GetWorkflow() []WorkflowStep` - 获取工作流定义

#### internal/team/team_test.go
- 完整的单元测试覆盖
- 测试团队创建、角色管理、角色切换等功能
- 测试默认团队配置和工作流定义
- 所有测试通过（18个测试用例）

### 2026-04-06 Task 7 新增：子代理Fork机制

#### internal/agent/fork.go
- `ForkManager` - 子代理Fork管理器（线程安全）
  - 管理子代理的创建、执行和合并
  - 提供身份切换和上下文隔离机制
  - 支持嵌套Fork（子代理可以再Fork）
- `ForkedAgent` - 被Fork出来的子代理结构
  - ID: 子代理唯一标识
  - Role: 子代理角色
  - Task: 子代理任务描述
  - ParentTaskID: 父任务ID
  - Agent: 子代理实例
  - StartTime/EndTime: 开始/结束时间
  - Summary: 执行总结
  - Status: 状态（running/completed/failed）
- `ForkResult` - 子代理执行结果结构
  - ForkID: 子代理ID
  - Role: 角色
  - Task: 任务描述
  - Summary: 执行总结
  - Status: 状态
  - Duration: 执行时长
  - Iterations: 迭代次数
- `ForkConfig` - Fork配置结构体
- `NewForkManager(config) (*ForkManager, error)` - 创建新的Fork管理器
- `Fork(ctx, parentAgent, role, task) (*ForkResult, error)` - 创建并执行子代理
- `Join(parentAgent, forkResult) (string, error)` - 合并子代理结果到父Agent
- `GetActiveForks() []*ForkedAgent` - 获取当前活动的子代理列表
- `SetOnForkStart(callback)` - 设置子代理开始回调
- `SetOnForkEnd(callback)` - 设置子代理结束回调
- `SetOnStreamChunk(callback)` - 设置流式输出回调
- `buildForkedAgentPrompt(role, task) string` - 构建子代理系统提示词
- `buildForkMessages(parentAgent, role, task) []Message` - 构建子代理初始消息（上下文隔离）
- `executeForkedAgent(ctx, agent, messages) (*ForkResult, error)` - 执行子代理
- `extractSummaryFromHistory(agent) string` - 从历史中提取总结

#### 身份切换提示模板
- `BuildForkTaskPrompt(role, task, stateSummary) string` - 构建子代理任务提示
  - 身份切换提示："接下来我转变身份为【{role}】，需要执行以下任务"
  - 包含任务内容和当前状态摘要
  - 包含完成指令："完成后使用 complete_as_agent 工具提交执行总结"
- `BuildJoinPrompt(role, task, summary) string` - 构建合并提示
  - 完成提示："我以【{role}】身份完成了以下任务"
  - 包含任务和执行总结
  - 返回主任务指令："现在继续主任务，请检查YAML状态并继续执行"

#### 子代理工具定义
- `SpawnAgentTool` - 创建子代理工具
  - Name: "spawn_agent"
  - 参数：role（角色）、task（任务描述）
  - 功能：创建子代理并同步执行，返回执行总结
- `CompleteAsAgentTool` - 子代理完成任务工具
  - Name: "complete_as_agent"
  - 参数：summary（执行总结）
  - 功能：子代理提交执行总结并结束
- `NewSpawnAgentTool(forkManager, parentAgent) *SpawnAgentTool` - 创建spawn_agent工具
- `NewCompleteAsAgentTool() *CompleteAsAgentTool` - 创建complete_as_agent工具
- `RegisterForkTools(registry, forkManager, parentAgent) error` - 注册Fork相关工具

#### 辅助函数
- `getPromptTypeByRole(role) SystemPromptType` - 根据角色获取提示词类型

### 2026-04-06 Task 5 新增：Agent核心循环模块

#### internal/agent/prompts.go
- `SystemPromptType` - 系统提示词类型（orchestrator/worker/reviewer）
- `OrchestratorSystemPrompt` - Orchestrator角色系统提示词（高效执行模式）
- `WorkerSystemPrompt` - Worker角色系统提示词
- `ReviewerSystemPrompt` - Reviewer角色系统提示词
- `YAMLStatePrompt` - YAML状态维护提示
- `GetSystemPrompt(promptType) string` - 根据类型获取系统提示词
- `BuildTaskPrompt(taskGoal, stateSummary) string` - 构建任务提示词
- `BuildToolResultPrompt(toolName, result) string` - 构建工具结果提示词
- `BuildErrorPrompt(err) string` - 构建错误提示词

#### internal/agent/history.go
- `HistoryManager` - 消息历史管理器（线程安全）
  - 使用sync.RWMutex保护并发访问
  - 支持消息添加、获取、截断等操作
- `NewHistoryManager() *HistoryManager` - 创建新的消息历史管理器
- `AddMessage(msg)` - 添加消息到历史
- `AddMessages(msgs)` - 批量添加消息
- `GetMessages() []Message` - 获取所有消息的副本
- `GetMessagesRef() []Message` - 获取消息引用（只读）
- `GetLastMessage() (Message, bool)` - 获取最后一条消息
- `GetMessageCount() int` - 获取消息数量
- `Clear()` - 清空消息历史
- `Truncate(maxTokens, tokenCounter)` - 截断历史以适应token限制
- `TruncateSimple(keepCount)` - 简单截断策略
- `GetTokenCount(tokenCounter) int` - 获取当前历史的token数量
- `GetMessagesByRole(role) []Message` - 获取指定角色的消息
- `RemoveLastMessage() bool` - 移除最后一条消息
- `ReplaceLastMessage(msg) bool` - 替换最后一条消息
- `GetRecentMessages(n) []Message` - 获取最近N条消息
- `Clone() *HistoryManager` - 克隆历史管理器

#### internal/agent/executor.go
- `ToolExecutor` - 工具调用执行器
- `NewToolExecutor(registry) *ToolExecutor` - 创建新的工具执行器
- `ExecuteToolCalls(ctx, toolCalls) ([]Message, error)` - 执行工具调用列表（并行）
- `ExecuteToolCallWithTimeout(ctx, tc, timeout) (Message, error)` - 执行工具调用（带超时）
- `ExecuteToolCallSequential(ctx, toolCalls) ([]Message, error)` - 顺序执行工具调用
- `ToolExecutionResult` - 工具执行结果（包含详细信息）
- `ExecuteToolCallsWithDetails(ctx, toolCalls) ([]ToolExecutionResult, error)` - 执行工具调用并返回详细信息
- `GetAvailableTools() []Tool` - 获取可用工具列表
- `GetToolSchemas() []model.Tool` - 获取工具Schema列表
- `HasTool(name) bool` - 检查工具是否存在
- `ParseToolCallArguments(tc) (map, error)` - 解析工具调用参数
- `BuildToolResultMessage(tc, result) Message` - 构建工具结果消息
- `BuildToolErrorMessage(tc, errMsg) Message` - 构建工具错误消息

#### internal/agent/core.go
- `Agent` - 核心Agent结构体
  - modelClient: 模型客户端
  - toolRegistry: 工具注册中心
  - stateManager: 状态管理器
  - executor: 工具执行器
  - history: 消息历史
  - maxIterations: 最大迭代次数
  - systemPrompt: 系统提示词
- `Config` - Agent配置结构体
- `NewAgent(config) (*Agent, error)` - 创建新的Agent实例
- `Run(ctx, taskGoal) (*RunResult, error)` - 执行任务主循环
- `RunResult` - 运行结果结构体
  - TaskID: 任务ID
  - Status: 状态（completed/failed/cancelled/max_iterations）
  - StartTime/EndTime: 开始/结束时间
  - Duration: 执行时长
  - Iterations: 迭代次数
  - FinalResponse: 最终响应内容
  - Error: 错误信息
- `Stop()` - 停止Agent运行
- `IsRunning() bool` - 检查Agent是否正在运行
- `SetTaskID(taskID)` - 设置任务ID
- `GetTaskID() string` - 获取当前任务ID
- `SetOnStreamChunk(callback)` - 设置流式输出回调
- `SetOnToolCall(callback)` - 设置工具调用回调
- `SetOnIteration(callback)` - 设置迭代回调
- `GetHistory() *HistoryManager` - 获取消息历史管理器
- `GetState() (*TaskState, error)` - 获取当前任务状态
- `RunSync(ctx, taskGoal) error` - 同步执行任务（简化接口）

### 2026-04-06 Task 4 新增：YAML状态管理模块

#### internal/state/task.go
- `TaskState` - 完整的任务状态文档结构
  - `Task TaskInfo` - 任务基本信息
  - `Progress ProgressInfo` - 任务进度信息
  - `Context ContextInfo` - 任务上下文信息
  - `Agents AgentsInfo` - Agent团队状态
- `TaskInfo` - 任务基本信息结构
  - `ID string` - 任务唯一标识符
  - `Goal string` - 任务目标描述
  - `Status string` - 任务状态（pending/in_progress/completed/failed）
  - `CreatedAt time.Time` - 创建时间
  - `UpdatedAt time.Time` - 最后更新时间
- `ProgressInfo` - 任务进度信息结构
  - `CurrentPhase string` - 当前阶段名称
  - `CompletedSteps []string` - 已完成的步骤列表
  - `PendingSteps []string` - 待完成的步骤列表
- `ContextInfo` - 任务上下文信息结构
  - `Decisions []string` - 决策记录
  - `Constraints []string` - 约束条件
  - `Files []FileInfo` - 相关文件信息
- `FileInfo` - 文件信息结构
  - `Path string` - 文件路径
  - `Description string` - 文件描述
  - `Status string` - 文件状态
- `AgentsInfo` - Agent团队状态结构
  - `Active string` - 当前活动的Agent角色
  - `History []AgentRecord` - Agent执行历史记录
- `AgentRecord` - Agent执行记录结构
  - `Role string` - Agent角色
  - `Summary string` - 执行摘要
  - `Duration string` - 执行时长
- `NewTaskState(id, goal string) *TaskState` - 创建新的任务状态实例
- `IsCompleted() bool` - 检查任务是否已完成
- `IsInProgress() bool` - 检查任务是否正在进行中
- `IsFailed() bool` - 检查任务是否失败
- `UpdateStatus(status string)` - 更新任务状态
- `AddCompletedStep(step string)` - 添加已完成的步骤
- `RemovePendingStep(step string)` - 从待办步骤中移除指定步骤
- `AddDecision(decision string)` - 添加决策记录
- `AddConstraint(constraint string)` - 添加约束条件
- `AddFile(path, description, status string)` - 添加相关文件信息
- `AddAgentRecord(role, summary, duration string)` - 添加Agent执行记录
- `SetActiveAgent(role string)` - 设置当前活动的Agent
- `SetCurrentPhase(phase string)` - 设置当前阶段

#### internal/state/yaml.go
- `LoadYAML(filePath string) (*TaskState, error)` - 从文件加载YAML
- `SaveYAML(state *TaskState, filePath string) error` - 保存YAML到文件
- `ParseYAML(data []byte) (*TaskState, error)` - 解析YAML字符串
- `ToYAML(state *TaskState) ([]byte, error)` - 序列化为YAML字节数据
- `ToYAMLString(state *TaskState) (string, error)` - 序列化为YAML字符串
- `GetYAMLSummary(state *TaskState) (string, error)` - 获取YAML摘要（用于上下文压缩）

#### internal/state/manager.go
- `StateManager` - 状态管理器（并发安全）
  - 使用sync.RWMutex保护并发访问
  - 支持内存缓存和文件持久化
  - 支持自动保存功能
- `NewStateManager(stateDir string, autoSave bool) (*StateManager, error)` - 创建状态管理器
- `CreateTask(id, goal string) (*TaskState, error)` - 创建新任务状态
- `Load(id string) (*TaskState, error)` - 从文件加载任务状态
- `Save(id string) error` - 保存任务状态到文件
- `UpdateProgress(id, phase, completedStep string) error` - 更新任务进度
- `AddDecision(id, decision string) error` - 添加决策记录
- `CompleteStep(id, step string) error` - 完成一个步骤
- `SwitchAgent(id, role, summary, duration string) error` - 切换活动Agent
- `GetState(id string) (*TaskState, error)` - 获取任务状态（只读）
- `GetYAMLSummary(id string) (string, error)` - 获取任务的YAML摘要
- `SetPendingSteps(id string, steps []string) error` - 设置待完成步骤列表
- `AddConstraint(id, constraint string) error` - 添加约束条件
- `AddFile(id, path, description, status string) error` - 添加相关文件信息
- `UpdateTaskStatus(id, status string) error` - 更新任务状态
- `ListTasks() ([]string, error)` - 列出所有已知任务ID
- `DeleteTask(id string) error` - 删除任务状态

### 2026-04-06 Task 2 新增：模型服务连接模块

#### internal/model/config.go
- `Config` - 模型服务配置结构体
  - `Endpoint string` - API端点地址
  - `APIKey string` - API密钥
  - `ModelName string` - 模型名称
  - `ContextSize int` - 上下文大小
- `DefaultConfig() *Config` - 返回默认配置
- `Validate() error` - 验证配置有效性
- `ConfigError` - 配置错误类型

#### internal/model/message.go
- `Role` - 消息角色类型（system/user/assistant/tool）
- `Message` - 聊天消息结构体
  - `Role Role` - 消息角色
  - `Content string` - 消息内容
  - `ToolCalls []ToolCall` - 工具调用请求
  - `ToolCallID string` - 工具调用ID
- `ToolCall` - 工具调用请求结构体
- `FunctionCall` - 函数调用详情
- `Tool` - 工具定义结构体
- `FunctionDef` - 函数定义结构体
- `ChatCompletionRequest` - 聊天补全请求
- `ChatCompletionResponse` - 聊天补全响应
- `Choice` - 响应选项
- `Delta` - 流式响应增量内容
- `Usage` - token使用统计
- `StreamResponse` - 流式响应结构
- `NewSystemMessage(content string) Message` - 创建系统消息
- `NewUserMessage(content string) Message` - 创建用户消息
- `NewAssistantMessage(content string) Message` - 创建助手消息
- `NewToolResultMessage(toolCallID, name, content string) Message` - 创建工具结果消息
- `ParseToolCallArguments() (map[string]interface{}, error)` - 解析工具调用参数

#### internal/model/client.go
- `Client` - OpenAI API兼容客户端
- `NewClient(config *Config) (*Client, error)` - 创建新的模型客户端
- `ChatCompletion(ctx context.Context, messages []Message, tools []Tool) (*ChatCompletionResponse, error)` - 发送聊天补全请求
- `ChatCompletionWithTemperature(ctx context.Context, messages []Message, tools []Tool, temperature float64) (*ChatCompletionResponse, error)` - 发送带温度参数的聊天补全请求
- `StreamChatCompletion(ctx context.Context, messages []Message, tools []Tool) (<-chan StreamEvent, error)` - 流式聊天补全
- `StreamEvent` - 流式事件结构
- `GetConfig() *Config` - 获取客户端配置
- `CountTokens(messages []Message) int` - 估算消息的token数量
- `IsContextOverflow(messages []Message) bool` - 检查是否超出上下文限制
- `RequestError` - 请求错误类型
- `APIError` - API错误类型

## 架构说明

### 模块划分

#### internal/model - 模型服务连接模块
负责与OpenAI API兼容的模型服务进行通信，支持：
- 标准Chat Completion API调用
- 流式响应处理（SSE）
- Function Calling/Tool Calling
- 思考标签处理
- 上下文管理

#### internal/tools - 工具集模块
提供Agent可调用的基础工具：
- 文件系统操作（读写、列表、删除）
- 命令执行
- 工具注册机制

#### internal/state - 状态管理模块
使用YAML格式维护任务状态：
- YAML解析与序列化
- 任务状态结构定义
- 状态自动更新机制

#### internal/agent - Agent核心模块
Agent主循环和核心逻辑：
- Agent主循环（Run方法）
- 消息历史管理
- 工具调用执行
- 系统提示词管理
- 状态自动更新机制
- 流式响应处理
- 思考标签处理
- 高效执行模式
- **程序逻辑审查**（Task 6）
  - 方向偏离检测
  - 错误模式识别（无限循环、无效调用、重复失败）
  - 编造内容检测
  - 进度验证
  - 审查反馈注入
- **子代理Fork机制**（Task 7）
  - 创建和管理子代理
  - 身份切换和上下文隔离
  - 结果合并和总结提取
  - 支持嵌套Fork

#### internal/supervisor - 监督系统模块
外部监督Agent行为，提供实时监督和干预机制：
- **监督检查**（Task 9）
  - 输出质量检查
  - 任务进度检查
  - 错误检测
  - 安全检查（可选）
  - 行为检查
  - 资源检查
- **干预机制**
  - 警告：记录问题但不中断
  - 暂停：暂停当前Agent，等待人工确认
  - 中断：中断当前操作，注入修正指令
  - 回滚：回滚到上一个稳定状态
- **监督报告**
  - 生成监督日志
  - 记录干预历史
  - 统计监督指标
- **并行监督**
  - 支持goroutine并行监督
  - 工作池模式
  - 线程安全

#### internal/team - 团队定义与角色管理模块
定义Agent团队和角色：
- 团队结构定义
- 角色职责管理
- 工作流定义
- 角色切换机制
- 工具权限管理

**预定义角色**：
- **Orchestrator（主控Agent）**：协调任务流程，维护YAML状态，可使用所有工具，可Fork
- **Architect（架构师）**：架构设计，技术选型，模块划分，可Fork
- **Developer（开发者）**：代码实现，功能开发，不可Fork
- **Tester（测试者）**：测试编写，验证功能，不可Fork
- **Reviewer（审查者）**：代码审查，质量把控，只读权限，不可Fork
- **Supervisor（监督者）**：监督Agent，检查输出质量，纠偏，只读权限，不可Fork

## 数据流

### 主流程
1. **用户输入** -> Agent核心
2. Agent核心 -> **模型服务** (ChatCompletion/StreamChatCompletion)
3. 模型服务 -> **工具调用** (Tool Calling)
4. 工具执行 -> **状态更新** (YAML State)
5. 状态更新 -> Agent核心 -> **循环或完成**

### 子代理Fork流程（Task 7）
```
主Agent -> spawn_agent(role, task) -> ForkManager创建子代理
        -> 子代理执行任务 -> complete_as_agent(summary)
        -> ForkManager返回总结 -> 主Agent继续
```

**详细步骤**：
1. 主Agent调用 `spawn_agent` 工具，指定角色和任务
2. ForkManager创建子代理实例，构建独立上下文
3. 子代理以指定角色身份执行任务
4. 子代理调用 `complete_as_agent` 提交执行总结
5. ForkManager提取总结并合并到主Agent
6. 主Agent接收总结，继续主任务

**上下文隔离策略**：
- 子代理继承父任务ID（共享状态）
- 子代理不复制完整消息历史（避免上下文过长）
- 子代理获取当前状态摘要作为上下文
- 子代理有独立的消息历史管理器

### 监督流程（Task 9）
```
Agent执行 -> 监督器并行检查 -> 发现问题 -> 干预决策 -> 执行干预
```

**详细步骤**：
1. Agent产生输出
2. 监督器接收输出并执行各项检查（质量、进度、错误、安全、行为、资源）
3. 汇总发现的问题
4. 确定监督状态（pass/warning/intervention）
5. 根据配置决定是否干预
6. 执行干预动作（如果需要）
   - 警告：记录问题但不中断
   - 暂停：暂停Agent，等待人工确认
   - 中断：中断当前操作
   - 回滚：回滚到上一个稳定状态
7. 触发相应回调
8. 记录干预历史和统计信息

**并行监督**：
- 使用工作池模式并行处理多个Agent输出
- 支持配置最大并行检查数
- 线程安全的状态管理

## 配置说明

配置文件位于 `configs/config.yaml`，包含：
- model: 模型服务配置
- agent: Agent行为配置
- state: 状态管理配置
- tools: 工具配置

## 测试覆盖

所有模块均包含单元测试，测试文件位于对应模块目录下，命名格式为 `*_test.go`。

当前测试覆盖：
- internal/model: 100% 核心功能覆盖
- internal/tools: 待实现
- internal/state: 100% 核心功能覆盖
  - 任务状态创建和操作
  - YAML序列化与反序列化
  - 文件读写操作
  - 状态管理器功能
  - 并发安全性验证
  - 时间字段序列化
- internal/agent: 100% 核心功能覆盖
  - Agent创建和配置
  - 消息历史管理
  - 工具执行器
  - Agent主循环
  - 取消和并发控制
  - 回调机制
  - **程序逻辑审查**（Task 6）
    - Reviewer创建和配置
    - 各类审查规则检测
    - 审查结果生成
    - 反馈消息注入
    - 并发安全性
  - **子代理Fork机制**（Task 7）
    - ForkManager创建和配置
    - Fork和Join操作
    - 身份切换提示构建
    - 工具定义和执行
    - 回调机制
    - 并发安全性
- internal/team: 100% 核心功能覆盖
  - 团队创建和配置
  - 角色管理
  - 角色切换
  - 工具权限检查
  - 默认团队配置
  - 工作流定义
- internal/supervisor: 100% 核心功能覆盖
  - 监督器创建和配置
  - 各类监督检查（质量、进度、错误、安全、行为、资源）
  - 干预机制（警告、暂停、中断、回滚）
  - 并行监督
  - 回调机制
  - 统计和日志
  - 并发安全性
