// Package agent 实现Agent核心循环和业务逻辑
package agent

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// TestNewSelfCorrector 测试创建自我修正器
func TestNewSelfCorrector(t *testing.T) {
	t.Run("使用默认配置创建", func(t *testing.T) {
		corrector := NewSelfCorrector(nil)
		if corrector == nil {
			t.Fatal("修正器不应为nil")
		}

		config := corrector.GetConfig()
		if config == nil {
			t.Fatal("配置不应为nil")
		}

		// 验证默认配置
		if config.MaxRetries != 3 {
			t.Errorf("默认最大重试次数应为3，实际为%d", config.MaxRetries)
		}
		if config.RetryDelayMs != 1000 {
			t.Errorf("默认重试延迟应为1000ms，实际为%d", config.RetryDelayMs)
		}
		if !config.ExponentialBackoff {
			t.Error("默认应启用指数退避")
		}
	})

	t.Run("使用自定义配置创建", func(t *testing.T) {
		customConfig := &SelfCorrectorConfig{
			MaxRetries:          5,
			RetryDelayMs:       500,
			ExponentialBackoff: false,
			MaxFailureHistory:  100,
		}

		corrector := NewSelfCorrector(customConfig)
		if corrector == nil {
			t.Fatal("修正器不应为nil")
		}

		config := corrector.GetConfig()
		if config.MaxRetries != 5 {
			t.Errorf("最大重试次数应为5，实际为%d", config.MaxRetries)
		}
		if config.RetryDelayMs != 500 {
			t.Errorf("重试延迟应为500ms，实际为%d", config.RetryDelayMs)
		}
		if config.ExponentialBackoff {
			t.Error("不应启用指数退避")
		}
	})
}

// TestAnalyzeFailure 测试分析失败
func TestAnalyzeFailure(t *testing.T) {
	corrector := NewSelfCorrector(nil)

	t.Run("分析校验失败", func(t *testing.T) {
		verifyResult := &VerifyResult{
			Status: VerifyStatusFail,
			Issues: []VerifyIssue{
				{
					Type:        VerifyIssueTypeFileNotFound,
					Severity:    "high",
					Description: "文件不存在: test.txt",
					Evidence:    "路径: /path/to/test.txt",
					Suggestion:  "请确保文件已创建",
				},
			},
			Timestamp: time.Now(),
		}

		result := corrector.AnalyzeFailure(verifyResult, nil)

		if result == nil {
			t.Fatal("修正结果不应为nil")
		}
		if !result.NeedsCorrection {
			t.Error("应需要修正")
		}
		if result.Status != CorrectionStatusNeeded {
			t.Errorf("状态应为needed，实际为%s", result.Status)
		}
		if result.Priority != "high" {
			t.Errorf("优先级应为high，实际为%s", result.Priority)
		}
		if len(result.Suggestions) == 0 {
			t.Error("应包含修正建议")
		}
	})

	t.Run("分析审查失败", func(t *testing.T) {
		reviewResult := &ReviewResult{
			Status: ReviewStatusBlock,
			Issues: []ReviewIssue{
				{
					Type:        IssueTypeInfiniteLoop,
					Severity:    "critical",
					Description: "检测到无限循环",
					Evidence:    "相同操作重复3次",
					Suggestion:  "请尝试不同的方法",
				},
			},
			Timestamp: time.Now(),
		}

		result := corrector.AnalyzeFailure(nil, reviewResult)

		if result == nil {
			t.Fatal("修正结果不应为nil")
		}
		if !result.NeedsCorrection {
			t.Error("应需要修正")
		}
		if result.Priority != "critical" {
			t.Errorf("优先级应为critical，实际为%s", result.Priority)
		}
	})

	t.Run("分析混合失败", func(t *testing.T) {
		verifyResult := &VerifyResult{
			Status: VerifyStatusFail,
			Issues: []VerifyIssue{
				{
					Type:        VerifyIssueTypeFileEmpty,
					Severity:    "medium",
					Description: "文件为空",
					Suggestion:  "请添加内容",
				},
			},
		}

		reviewResult := &ReviewResult{
			Status: ReviewStatusBlock,
			Issues: []ReviewIssue{
				{
					Type:        IssueTypeDirection,
					Severity:    "high",
					Description: "方向偏离",
					Suggestion:  "请重新聚焦任务目标",
				},
			},
		}

		result := corrector.AnalyzeFailure(verifyResult, reviewResult)

		if result == nil {
			t.Fatal("修正结果不应为nil")
		}
		if len(result.Suggestions) < 2 {
			t.Errorf("应包含至少2个建议，实际为%d", len(result.Suggestions))
		}
		// 优先级应取最高的
		if result.Priority != "high" {
			t.Errorf("优先级应为high，实际为%s", result.Priority)
		}
	})

	t.Run("无失败情况", func(t *testing.T) {
		verifyResult := &VerifyResult{
			Status:    VerifyStatusPass,
			Issues:    []VerifyIssue{},
			Timestamp: time.Now(),
		}

		result := corrector.AnalyzeFailure(verifyResult, nil)

		// 即使没有失败，也会记录一条记录
		if result == nil {
			t.Fatal("修正结果不应为nil")
		}
	})
}

// TestGenerateCorrectionInstruction 测试生成修正指令
func TestGenerateCorrectionInstruction(t *testing.T) {
	corrector := NewSelfCorrector(nil)

	t.Run("生成标准修正指令", func(t *testing.T) {
		correctionResult := &CorrectionResult{
			Status:           CorrectionStatusNeeded,
			NeedsCorrection:  true,
			FailureSummary:   "文件不存在",
			Priority:         "high",
			RemainingRetries: 2,
			Suggestions: []string{
				"请确保文件已创建",
				"检查文件路径是否正确",
			},
			Timestamp: time.Now(),
		}

		instruction := corrector.GenerateCorrectionInstruction(correctionResult)

		if instruction == "" {
			t.Fatal("修正指令不应为空")
		}
		if !strings.Contains(instruction, "文件不存在") {
			t.Error("修正指令应包含失败原因")
		}
		if !strings.Contains(instruction, "high") {
			t.Error("修正指令应包含优先级")
		}
		if !strings.Contains(instruction, "2") {
			t.Error("修正指令应包含剩余重试次数")
		}
		if !strings.Contains(instruction, "请确保文件已创建") {
			t.Error("修正指令应包含建议")
		}
	})

	t.Run("不需要修正时返回空", func(t *testing.T) {
		correctionResult := &CorrectionResult{
			NeedsCorrection: false,
		}

		instruction := corrector.GenerateCorrectionInstruction(correctionResult)

		if instruction != "" {
			t.Error("不需要修正时应返回空字符串")
		}
	})

	t.Run("nil结果返回空", func(t *testing.T) {
		instruction := corrector.GenerateCorrectionInstruction(nil)

		if instruction != "" {
			t.Error("nil结果应返回空字符串")
		}
	})

	t.Run("指令长度限制", func(t *testing.T) {
		// 创建一个会产生超长指令的配置
		config := &SelfCorrectorConfig{
			MaxRetries:           3,
			MaxInstructionLength: 100, // 设置较小的长度限制
			IncludeSuggestions:   true,
		}
		corrector := NewSelfCorrector(config)

		correctionResult := &CorrectionResult{
			NeedsCorrection: true,
			FailureSummary:  "这是一个很长的失败原因描述，用于测试指令长度限制功能是否正常工作",
			Priority:        "high",
			RemainingRetries: 2,
			Suggestions: []string{
				"建议1：这是一个很长的建议内容",
				"建议2：这是另一个很长的建议内容",
				"建议3：这是第三个很长的建议内容",
			},
			Timestamp: time.Now(),
		}

		instruction := corrector.GenerateCorrectionInstruction(correctionResult)

		if len(instruction) > 150 { // 允许一些余量
			t.Errorf("指令长度应被限制，实际长度为%d", len(instruction))
		}
		if !strings.Contains(instruction, "截断") {
			t.Error("被截断的指令应包含截断标记")
		}
	})
}

// TestShouldRetry 测试重试逻辑
func TestShouldRetry(t *testing.T) {
	t.Run("正常重试", func(t *testing.T) {
		corrector := NewSelfCorrector(&SelfCorrectorConfig{
			MaxRetries: 3,
		})

		// 第一次重试
		if !corrector.ShouldRetry() {
			t.Error("第一次应该允许重试")
		}
		if corrector.GetRetryCount() != 1 {
			t.Errorf("重试计数应为1，实际为%d", corrector.GetRetryCount())
		}

		// 第二次重试
		if !corrector.ShouldRetry() {
			t.Error("第二次应该允许重试")
		}
		if corrector.GetRetryCount() != 2 {
			t.Errorf("重试计数应为2，实际为%d", corrector.GetRetryCount())
		}

		// 第三次重试
		if !corrector.ShouldRetry() {
			t.Error("第三次应该允许重试")
		}
		if corrector.GetRetryCount() != 3 {
			t.Errorf("重试计数应为3，实际为%d", corrector.GetRetryCount())
		}

		// 第四次不应允许
		if corrector.ShouldRetry() {
			t.Error("第四次不应允许重试")
		}
	})

	t.Run("重试次数耗尽", func(t *testing.T) {
		corrector := NewSelfCorrector(&SelfCorrectorConfig{
			MaxRetries: 1,
		})

		// 第一次重试
		if !corrector.ShouldRetry() {
			t.Error("第一次应该允许重试")
		}

		// 第二次不应允许
		if corrector.ShouldRetry() {
			t.Error("重试次数耗尽后不应允许重试")
		}

		if corrector.GetRemainingRetries() != 0 {
			t.Errorf("剩余重试次数应为0，实际为%d", corrector.GetRemainingRetries())
		}
	})
}

// TestRecordFailure 测试记录失败
func TestRecordFailure(t *testing.T) {
	corrector := NewSelfCorrector(&SelfCorrectorConfig{
		MaxFailureHistory: 3,
	})

	t.Run("记录单次失败", func(t *testing.T) {
		corrector.RecordFailure(
			FailureTypeVerify,
			"文件不存在",
			"详细错误信息",
			[]string{"问题1", "问题2"},
		)

		history := corrector.GetFailureHistory()
		if len(history) != 1 {
			t.Fatalf("历史记录长度应为1，实际为%d", len(history))
		}

		if history[0].Type != FailureTypeVerify {
			t.Errorf("失败类型应为verify，实际为%s", history[0].Type)
		}
		if history[0].Summary != "文件不存在" {
			t.Errorf("摘要不匹配，实际为%s", history[0].Summary)
		}
		if len(history[0].Issues) != 2 {
			t.Errorf("问题数量应为2，实际为%d", len(history[0].Issues))
		}
	})

	t.Run("历史记录限制", func(t *testing.T) {
		// 记录多次失败
		for i := 0; i < 5; i++ {
			corrector.RecordFailure(
				FailureTypeTool,
				"工具执行失败",
				"详细信息",
				[]string{"问题"},
			)
		}

		history := corrector.GetFailureHistory()
		if len(history) > 3 {
			t.Errorf("历史记录应被限制在3条，实际为%d", len(history))
		}
	})
}

// TestReset 测试重置
func TestReset(t *testing.T) {
	corrector := NewSelfCorrector(nil)

	// 进行一些操作
	corrector.ShouldRetry()
	corrector.ShouldRetry()
	corrector.RecordFailure(FailureTypeVerify, "失败", "详情", []string{"问题"})

	// 重置
	corrector.Reset()

	// 验证状态
	if corrector.GetRetryCount() != 0 {
		t.Errorf("重试计数应为0，实际为%d", corrector.GetRetryCount())
	}

	history := corrector.GetFailureHistory()
	if len(history) != 0 {
		t.Errorf("历史记录应为空，实际长度为%d", len(history))
	}
}

// TestDetectFailurePattern 测试检测失败模式
func TestDetectFailurePattern(t *testing.T) {
	t.Run("检测重复模式", func(t *testing.T) {
		config := &SelfCorrectorConfig{
			MaxRetries:          3,
			MaxFailureHistory:   50,
			FailurePatternWindow: 5,
		}
		corrector := NewSelfCorrector(config)

		// 记录多次相似的失败
		for i := 0; i < 5; i++ {
			corrector.RecordFailure(
				FailureTypeVerify,
				"文件不存在",
				"详情",
				[]string{"文件路径错误", "权限不足"},
			)
		}

		pattern := corrector.DetectFailurePattern()

		if pattern == nil {
			t.Fatal("应检测到失败模式")
		}
		if len(pattern.RecurringIssues) == 0 {
			t.Error("应检测到重复出现的问题")
		}
		if pattern.Severity != "high" {
			t.Errorf("严重程度应为high，实际为%s", pattern.Severity)
		}
	})

	t.Run("历史记录不足时不检测", func(t *testing.T) {
		config := &SelfCorrectorConfig{
			MaxRetries:          3,
			MaxFailureHistory:   50,
			FailurePatternWindow: 5,
		}
		corrector := NewSelfCorrector(config)

		// 只记录2次失败
		for i := 0; i < 2; i++ {
			corrector.RecordFailure(
				FailureTypeVerify,
				"失败",
				"详情",
				[]string{"问题"},
			)
		}

		pattern := corrector.DetectFailurePattern()

		if pattern != nil {
			t.Error("历史记录不足时不应检测模式")
		}
	})
}

// TestGetRetryDelay 测试获取重试延迟
func TestGetRetryDelay(t *testing.T) {
	t.Run("指数退避", func(t *testing.T) {
		config := &SelfCorrectorConfig{
			MaxRetries:         3,
			RetryDelayMs:       1000,
			ExponentialBackoff: true,
		}
		corrector := NewSelfCorrector(config)

		// 初始状态（第0次重试）
		delay0 := corrector.GetRetryDelay()
		if delay0 != 1000*time.Millisecond {
			t.Errorf("初始延迟应为1000ms，实际为%v", delay0)
		}

		// 第一次重试后
		corrector.ShouldRetry()
		delay1 := corrector.GetRetryDelay()
		if delay1 != 2000*time.Millisecond {
			t.Errorf("第一次重试后延迟应为2000ms，实际为%v", delay1)
		}

		// 第二次重试后
		corrector.ShouldRetry()
		delay2 := corrector.GetRetryDelay()
		if delay2 != 4000*time.Millisecond {
			t.Errorf("第二次重试后延迟应为4000ms，实际为%v", delay2)
		}
	})

	t.Run("固定延迟", func(t *testing.T) {
		config := &SelfCorrectorConfig{
			MaxRetries:         3,
			RetryDelayMs:       500,
			ExponentialBackoff: false,
		}
		corrector := NewSelfCorrector(config)

		corrector.ShouldRetry()
		delay1 := corrector.GetRetryDelay()
		corrector.ShouldRetry()
		delay2 := corrector.GetRetryDelay()

		if delay1 != delay2 {
			t.Errorf("固定延迟应保持不变，delay1=%v, delay2=%v", delay1, delay2)
		}
	})

	t.Run("最大延迟限制", func(t *testing.T) {
		config := &SelfCorrectorConfig{
			RetryDelayMs:       10000, // 10秒
			ExponentialBackoff: true,
		}
		corrector := NewSelfCorrector(config)

		// 多次重试使延迟超过30秒
		for i := 0; i < 5; i++ {
			corrector.ShouldRetry()
		}

		delay := corrector.GetRetryDelay()
		if delay > 30*time.Second {
			t.Errorf("延迟应被限制在30秒以内，实际为%v", delay)
		}
	})
}

// TestMarkSuccess 测试标记成功
func TestMarkSuccess(t *testing.T) {
	corrector := NewSelfCorrector(nil)

	// 进行一些重试
	corrector.ShouldRetry()
	corrector.ShouldRetry()
	corrector.RecordFailure(FailureTypeVerify, "失败", "详情", []string{"问题"})

	// 标记成功
	corrector.MarkSuccess()

	// 验证重试计数被重置
	if corrector.GetRetryCount() != 0 {
		t.Errorf("重试计数应被重置为0，实际为%d", corrector.GetRetryCount())
	}

	// 验证失败记录被标记为已修正
	history := corrector.GetFailureHistory()
	if len(history) > 0 && !history[len(history)-1].Corrected {
		t.Error("最近的失败记录应被标记为已修正")
	}
}

// TestGetStatistics 测试获取统计信息
func TestGetStatistics(t *testing.T) {
	corrector := NewSelfCorrector(&SelfCorrectorConfig{
		MaxRetries:         3,
		MaxFailureHistory:  50,
	})

	// 进行一些操作
	corrector.ShouldRetry()
	corrector.RecordFailure(FailureTypeVerify, "失败1", "详情", []string{"问题"})
	corrector.RecordFailure(FailureTypeTool, "失败2", "详情", []string{"问题"})
	corrector.MarkSuccess()

	stats := corrector.GetStatistics()

	if stats == nil {
		t.Fatal("统计信息不应为nil")
	}
	if stats.CurrentRetryCount != 0 {
		t.Errorf("当前重试次数应为0，实际为%d", stats.CurrentRetryCount)
	}
	if stats.MaxRetries != 3 {
		t.Errorf("最大重试次数应为3，实际为%d", stats.MaxRetries)
	}
	if stats.TotalFailures != 2 {
		t.Errorf("总失败次数应为2，实际为%d", stats.TotalFailures)
	}
	if stats.CorrectedFailures != 1 {
		t.Errorf("已修正失败次数应为1，实际为%d", stats.CorrectedFailures)
	}
	if stats.CorrectionSuccessRate != 0.5 {
		t.Errorf("修正成功率应为0.5，实际为%f", stats.CorrectionSuccessRate)
	}
}

// TestUpdateConfig 测试更新配置
func TestUpdateConfig(t *testing.T) {
	corrector := NewSelfCorrector(nil)

	t.Run("更新有效配置", func(t *testing.T) {
		newConfig := &SelfCorrectorConfig{
			MaxRetries: 5,
		}

		err := corrector.UpdateConfig(newConfig)
		if err != nil {
			t.Errorf("更新配置不应出错: %v", err)
		}

		config := corrector.GetConfig()
		if config.MaxRetries != 5 {
			t.Errorf("最大重试次数应为5，实际为%d", config.MaxRetries)
		}
	})

	t.Run("更新nil配置应出错", func(t *testing.T) {
		err := corrector.UpdateConfig(nil)
		if err == nil {
			t.Error("更新nil配置应返回错误")
		}
	})
}

// TestRecordCorrection 测试记录修正结果
func TestRecordCorrection(t *testing.T) {
	corrector := NewSelfCorrector(nil)

	// 记录失败
	corrector.RecordFailure(FailureTypeVerify, "失败", "详情", []string{"问题"})

	// 记录修正成功
	correctionResult := &CorrectionResult{
		Status: CorrectionStatusSuccess,
	}
	corrector.RecordCorrection(correctionResult)

	// 验证重试计数被重置
	if corrector.GetRetryCount() != 0 {
		t.Errorf("重试计数应被重置为0，实际为%d", corrector.GetRetryCount())
	}

	// 验证修正历史
	correctionHistory := corrector.GetCorrectionHistory()
	if len(correctionHistory) != 1 {
		t.Errorf("修正历史长度应为1，实际为%d", len(correctionHistory))
	}
}

// TestConcurrentAccess 测试并发访问
func TestConcurrentAccess(t *testing.T) {
	corrector := NewSelfCorrector(nil)

	var wg sync.WaitGroup
	numGoroutines := 10

	// 并发重试
	t.Run("并发重试", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				corrector.ShouldRetry()
			}()
		}
		wg.Wait()

		// 重试计数应在合理范围内
		count := corrector.GetRetryCount()
		if count < 0 || count > corrector.GetConfig().MaxRetries {
			t.Errorf("重试计数异常: %d", count)
		}
	})

	// 并发记录失败
	t.Run("并发记录失败", func(t *testing.T) {
		corrector.Reset()
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				corrector.RecordFailure(
					FailureTypeVerify,
					"失败",
					"详情",
					[]string{"问题"},
				)
			}(i)
		}
		wg.Wait()

		history := corrector.GetFailureHistory()
		if len(history) != numGoroutines {
			t.Errorf("历史记录长度应为%d，实际为%d", numGoroutines, len(history))
		}
	})

	// 并发读取
	t.Run("并发读取", func(t *testing.T) {
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = corrector.GetRetryCount()
				_ = corrector.GetRemainingRetries()
				_ = corrector.GetFailureHistory()
				_ = corrector.GetStatistics()
			}()
		}
		wg.Wait()
	})
}
