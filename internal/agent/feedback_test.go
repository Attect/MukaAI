package agent

import (
	"strings"
	"testing"
	"time"

	"agentplus/internal/model"
)

// TestNewFeedbackInjector 测试创建反馈注入器
func TestNewFeedbackInjector(t *testing.T) {
	// 使用默认配置
	injector := NewFeedbackInjector(nil)
	if injector == nil {
		t.Fatal("NewFeedbackInjector returned nil")
	}

	if injector.maxFeedbackLength != 500 {
		t.Errorf("expected maxFeedbackLength=500, got %d", injector.maxFeedbackLength)
	}

	// 使用自定义配置
	config := &FeedbackInjectorConfig{
		MaxFeedbackLength: 1000,
		IncludeEvidence:   false,
		IncludeTimestamp:  true,
	}
	injector = NewFeedbackInjector(config)
	if injector.maxFeedbackLength != 1000 {
		t.Errorf("expected maxFeedbackLength=1000, got %d", injector.maxFeedbackLength)
	}
	if injector.includeEvidence {
		t.Error("includeEvidence should be false")
	}
	if !injector.includeTimestamp {
		t.Error("includeTimestamp should be true")
	}
}

// TestFeedbackMessageToUserMessage 测试反馈消息转换
func TestFeedbackMessageToUserMessage(t *testing.T) {
	feedback := &FeedbackMessage{
		Level:       FeedbackLevelWarning,
		Title:       "测试反馈",
		Content:     "这是一个测试反馈内容",
		Suggestions: []string{"建议1", "建议2"},
		Timestamp:   time.Now(),
	}

	msg := feedback.ToUserMessage()

	if msg.Role != model.RoleUser {
		t.Errorf("expected role user, got %s", msg.Role)
	}

	if !strings.Contains(msg.Content, "[WARNING]") {
		t.Error("message should contain warning level")
	}
	if !strings.Contains(msg.Content, "测试反馈") {
		t.Error("message should contain title")
	}
	if !strings.Contains(msg.Content, "建议1") {
		t.Error("message should contain suggestions")
	}
}

// TestInjectFeedback 测试反馈注入
func TestInjectFeedback(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	// 测试空结果
	msg := injector.InjectFeedback(nil)
	if msg.Content != "" {
		t.Error("empty result should produce empty message")
	}

	// 测试无问题的结果
	result := &ReviewResult{
		Status:    ReviewStatusPass,
		Issues:    []ReviewIssue{},
		Timestamp: time.Now(),
	}
	msg = injector.InjectFeedback(result)
	if msg.Content != "" {
		t.Error("pass result with no issues should produce empty message")
	}

	// 测试有问题的结果
	result = &ReviewResult{
		Status: ReviewStatusBlock,
		Issues: []ReviewIssue{
			{
				Type:        IssueTypeInfiniteLoop,
				Severity:    "critical",
				Description: "检测到无限循环",
				Evidence:    "重复调用3次",
				Suggestion:  "请修改参数",
			},
		},
		Timestamp: time.Now(),
	}
	msg = injector.InjectFeedback(result)

	if msg.Role != model.RoleUser {
		t.Errorf("expected role user, got %s", msg.Role)
	}
	if !strings.Contains(msg.Content, "CRITICAL") {
		t.Error("blocking result should produce critical level message")
	}
}

// TestInjectFeedbackForIssue 测试单个问题反馈
func TestInjectFeedbackForIssue(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	issue := ReviewIssue{
		Type:        IssueTypeDirection,
		Severity:    "medium",
		Description: "输出与任务目标偏离",
		Evidence:    "关键词匹配率低",
		Suggestion:  "请确保输出与目标相关",
		Timestamp:   time.Now(),
	}

	msg := injector.InjectFeedbackForIssue(issue)

	if msg.Role != model.RoleUser {
		t.Errorf("expected role user, got %s", msg.Role)
	}
	if !strings.Contains(msg.Content, "方向偏离") {
		t.Error("message should contain issue title")
	}
	if !strings.Contains(msg.Content, "请确保输出与目标相关") {
		t.Error("message should contain suggestion")
	}
}

// TestInjectBlockingFeedback 测试阻断反馈
func TestInjectBlockingFeedback(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	// 测试无阻断问题
	result := &ReviewResult{
		Status: ReviewStatusPass,
		Issues: []ReviewIssue{
			{Type: IssueTypeDirection, Severity: "low"},
		},
	}
	msg := injector.InjectBlockingFeedback(result)
	if msg.Content != "" {
		t.Error("non-blocking result should produce empty message")
	}

	// 测试有阻断问题
	result = &ReviewResult{
		Status: ReviewStatusBlock,
		Issues: []ReviewIssue{
			{Type: IssueTypeInfiniteLoop, Severity: "critical", Description: "无限循环"},
			{Type: IssueTypeInvalidToolCall, Severity: "high", Description: "无效调用"},
		},
	}
	msg = injector.InjectBlockingFeedback(result)

	if !strings.Contains(msg.Content, "CRITICAL") {
		t.Error("should contain critical level")
	}
	if !strings.Contains(msg.Content, "执行被阻断") {
		t.Error("should contain blocking title")
	}
}

// TestInjectWarningFeedback 测试警告反馈
func TestInjectWarningFeedback(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	// 测试非警告状态
	result := &ReviewResult{Status: ReviewStatusPass}
	msg := injector.InjectWarningFeedback(result)
	if msg.Content != "" {
		t.Error("non-warning result should produce empty message")
	}

	// 测试警告状态
	result = &ReviewResult{
		Status: ReviewStatusWarning,
		Issues: []ReviewIssue{
			{Type: IssueTypeDirection, Severity: "medium", Description: "方向偏离", Suggestion: "调整方向"},
		},
	}
	msg = injector.InjectWarningFeedback(result)

	if !strings.Contains(msg.Content, "WARNING") {
		t.Error("should contain warning level")
	}
	if !strings.Contains(msg.Content, "执行警告") {
		t.Error("should contain warning title")
	}
}

// TestInjectProgressFeedback 测试进度反馈
func TestInjectProgressFeedback(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	// 测试低迭代次数
	msg := injector.InjectProgressFeedback(5, 50)
	if !strings.Contains(msg.Content, "已迭代 5 次") {
		t.Error("should contain iteration count")
	}
	if strings.Contains(msg.Content, "超过最大迭代次数的一半") {
		t.Error("should not contain warning for low iteration")
	}

	// 测试高迭代次数
	msg = injector.InjectProgressFeedback(30, 50)
	if !strings.Contains(msg.Content, "超过最大迭代次数的一半") {
		t.Error("should contain warning for high iteration")
	}
}

// TestInjectLoopDetectedFeedback 测试循环检测反馈
func TestInjectLoopDetectedFeedback(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	msg := injector.InjectLoopDetectedFeedback("read_file", `{"file_path": "test.txt"}`, 3)

	if !strings.Contains(msg.Content, "CRITICAL") {
		t.Error("should contain critical level")
	}
	if !strings.Contains(msg.Content, "无限循环") {
		t.Error("should contain loop detection title")
	}
	if !strings.Contains(msg.Content, "read_file") {
		t.Error("should contain tool name")
	}
	if !strings.Contains(msg.Content, "3 次") {
		t.Error("should contain repeat count")
	}
}

// TestInjectFailureFeedback 测试失败反馈
func TestInjectFailureFeedback(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	msg := injector.InjectFailureFeedback("write_file", 3, "permission denied")

	if !strings.Contains(msg.Content, "ERROR") {
		t.Error("should contain error level")
	}
	if !strings.Contains(msg.Content, "连续失败") {
		t.Error("should contain failure title")
	}
	if !strings.Contains(msg.Content, "write_file") {
		t.Error("should contain tool name")
	}
	if !strings.Contains(msg.Content, "permission denied") {
		t.Error("should contain error message")
	}
}

// TestInjectDirectionFeedback 测试方向偏离反馈
func TestInjectDirectionFeedback(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	msg := injector.InjectDirectionFeedback("实现登录功能", "今天天气很好")

	if !strings.Contains(msg.Content, "WARNING") {
		t.Error("should contain warning level")
	}
	if !strings.Contains(msg.Content, "偏离任务目标") {
		t.Error("should contain direction deviation title")
	}
	if !strings.Contains(msg.Content, "实现登录功能") {
		t.Error("should contain task goal")
	}
}

// TestDetermineFeedbackLevel 测试反馈级别确定
func TestDetermineFeedbackLevel(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	tests := []struct {
		status   ReviewStatus
		expected FeedbackLevel
	}{
		{ReviewStatusBlock, FeedbackLevelCritical},
		{ReviewStatusWarning, FeedbackLevelWarning},
		{ReviewStatusPass, FeedbackLevelInfo},
	}

	for _, test := range tests {
		result := &ReviewResult{Status: test.status}
		level := injector.determineFeedbackLevel(result)
		if level != test.expected {
			t.Errorf("status %s: expected level %s, got %s", test.status, test.expected, level)
		}
	}
}

// TestGetIssueTitle 测试获取问题标题
func TestGetIssueTitle(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	tests := []struct {
		issueType IssueType
		expected  string
	}{
		{IssueTypeDirection, "方向偏离"},
		{IssueTypeInfiniteLoop, "无限循环"},
		{IssueTypeInvalidToolCall, "无效工具调用"},
		{IssueTypeRepeatedFailure, "重复失败"},
		{IssueTypeFabrication, "编造内容"},
		{IssueTypeNoProgress, "无进度"},
		{IssueType("unknown"), "未知问题"},
	}

	for _, test := range tests {
		title := injector.getIssueTitle(test.issueType)
		if title != test.expected {
			t.Errorf("issue type %s: expected title %s, got %s", test.issueType, test.expected, title)
		}
	}
}

// TestIssueSeverityToFeedbackLevel 测试严重度转换
func TestIssueSeverityToFeedbackLevel(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	tests := []struct {
		severity string
		expected FeedbackLevel
	}{
		{"critical", FeedbackLevelCritical},
		{"high", FeedbackLevelError},
		{"medium", FeedbackLevelWarning},
		{"low", FeedbackLevelInfo},
		{"unknown", FeedbackLevelInfo},
	}

	for _, test := range tests {
		level := injector.issueSeverityToFeedbackLevel(test.severity)
		if level != test.expected {
			t.Errorf("severity %s: expected level %s, got %s", test.severity, test.expected, level)
		}
	}
}

// TestBatchInjectFeedback 测试批量反馈注入
func TestBatchInjectFeedback(t *testing.T) {
	injector := NewFeedbackInjector(nil)

	results := []*ReviewResult{
		{
			Status: ReviewStatusWarning,
			Issues: []ReviewIssue{{Type: IssueTypeDirection, Severity: "medium"}},
		},
		{
			Status: ReviewStatusBlock,
			Issues: []ReviewIssue{{Type: IssueTypeInfiniteLoop, Severity: "critical"}},
		},
		nil, // 应该被忽略
		{
			Status: ReviewStatusPass,
			Issues: []ReviewIssue{}, // 应该被忽略
		},
	}

	messages := injector.BatchInjectFeedback(results)

	if len(messages) != 2 {
		t.Errorf("expected 2 messages, got %d", len(messages))
	}
}

// TestFormatFeedbackForLog 测试日志格式化
func TestFormatFeedbackForLog(t *testing.T) {
	// 测试空结果
	output := FormatFeedbackForLog(nil)
	if output != "无审查结果" {
		t.Errorf("expected '无审查结果', got '%s'", output)
	}

	// 测试有结果
	result := &ReviewResult{
		Status:  ReviewStatusWarning,
		Summary: "发现1个问题: direction(1)",
		Issues: []ReviewIssue{
			{Type: IssueTypeDirection, Severity: "medium", Description: "方向偏离"},
		},
	}
	output = FormatFeedbackForLog(result)

	if !strings.Contains(output, "[warning]") {
		t.Error("should contain status")
	}
	if !strings.Contains(output, "direction") {
		t.Error("should contain issue type")
	}
}

// TestBuildContent 测试内容构建
func TestBuildContent(t *testing.T) {
	config := &FeedbackInjectorConfig{
		IncludeEvidence: true,
	}
	injector := NewFeedbackInjector(config)

	result := &ReviewResult{
		Issues: []ReviewIssue{
			{
				Type:        IssueTypeDirection,
				Description: "方向偏离",
				Evidence:    "关键词不匹配",
			},
		},
	}

	content := injector.buildContent(result)

	if !strings.Contains(content, "方向偏离") {
		t.Error("should contain description")
	}
	if !strings.Contains(content, "关键词不匹配") {
		t.Error("should contain evidence when enabled")
	}

	// 测试不包含证据
	config.IncludeEvidence = false
	injector = NewFeedbackInjector(config)
	content = injector.buildContent(result)

	if strings.Contains(content, "关键词不匹配") {
		t.Error("should not contain evidence when disabled")
	}
}

// TestBuildContentMaxLength 测试内容长度限制
func TestBuildContentMaxLength(t *testing.T) {
	config := &FeedbackInjectorConfig{
		MaxFeedbackLength: 50,
		IncludeEvidence:   true,
	}
	injector := NewFeedbackInjector(config)

	// 创建长内容
	longDesc := ""
	for i := 0; i < 100; i++ {
		longDesc += "这是一段很长的描述内容"
	}

	result := &ReviewResult{
		Issues: []ReviewIssue{
			{
				Type:        IssueTypeDirection,
				Description: longDesc,
				Evidence:    "证据",
			},
		},
	}

	content := injector.buildContent(result)

	if len(content) > 53 { // 50 + "..."
		t.Errorf("content should be truncated, got length %d", len(content))
	}
	if !strings.Contains(content, "...") {
		t.Error("truncated content should end with ...")
	}
}

// TestFeedbackLevels 测试所有反馈级别
func TestFeedbackLevels(t *testing.T) {
	levels := []FeedbackLevel{
		FeedbackLevelInfo,
		FeedbackLevelWarning,
		FeedbackLevelError,
		FeedbackLevelCritical,
	}

	for _, level := range levels {
		feedback := &FeedbackMessage{
			Level:   level,
			Title:   "测试",
			Content: "内容",
		}

		msg := feedback.ToUserMessage()
		// 消息中包含大写的级别字符串，如 [INFO]
		expectedLevel := strings.ToUpper(string(level))
		if !strings.Contains(msg.Content, expectedLevel) {
			t.Errorf("message should contain level %s, got: %s", expectedLevel, msg.Content)
		}
	}
}
