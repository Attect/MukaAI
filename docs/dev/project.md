# 项目架构文档

## 更新日志

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
