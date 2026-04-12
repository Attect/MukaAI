package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agentplus/internal/state"
)

// 测试辅助函数

func createTestFile(t *testing.T, content string) string {
	t.Helper()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "test_file.go")
	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		t.Fatalf("创建测试文件失败: %v", err)
	}
	return filePath
}

func createEmptyTestFile(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "empty_file.go")
	if err := os.WriteFile(filePath, []byte(""), 0644); err != nil {
		t.Fatalf("创建空文件失败: %v", err)
	}
	return filePath
}

// TestNewVerifier 测试创建校验器
func TestNewVerifier(t *testing.T) {
	t.Run("使用默认配置", func(t *testing.T) {
		verifier := NewVerifier(nil)
		if verifier == nil {
			t.Fatal("校验器不应为nil")
		}
		if verifier.config == nil {
			t.Fatal("配置不应为nil")
		}
		if !verifier.config.CheckFileExists {
			t.Error("默认配置应启用文件存在检查")
		}
		if !verifier.config.CheckFileNonEmpty {
			t.Error("默认配置应启用文件非空检查")
		}
	})

	t.Run("使用自定义配置", func(t *testing.T) {
		config := &VerifyConfig{
			CheckFileExists:   false,
			CheckFileNonEmpty: true,
		}
		verifier := NewVerifier(config)
		if verifier.config.CheckFileExists {
			t.Error("自定义配置应禁用文件存在检查")
		}
		if !verifier.config.CheckFileNonEmpty {
			t.Error("自定义配置应启用文件非空检查")
		}
	})
}

// TestDefaultVerifierConfig 测试默认配置
func TestDefaultVerifierConfig(t *testing.T) {
	config := DefaultVerifierConfig()
	if config == nil {
		t.Fatal("默认配置不应为nil")
	}

	if !config.CheckFileExists {
		t.Error("默认应启用文件存在检查")
	}
	if !config.CheckFileNonEmpty {
		t.Error("默认应启用文件非空检查")
	}
	if config.MaxFileSizeToCheck <= 0 {
		t.Error("最大文件大小应大于0")
	}
	if !config.EnableCustomRules {
		t.Error("默认应启用自定义规则")
	}
	if config.MaxIssuesToReport <= 0 {
		t.Error("最大报告问题数应大于0")
	}
}

// TestVerifyResult 测试校验结果方法
func TestVerifyResult(t *testing.T) {
	t.Run("IsFailed", func(t *testing.T) {
		result := &VerifyResult{Status: VerifyStatusPass}
		if result.IsFailed() {
			t.Error("通过状态不应返回失败")
		}

		result.Status = VerifyStatusFail
		if !result.IsFailed() {
			t.Error("失败状态应返回失败")
		}
	})

	t.Run("HasWarnings", func(t *testing.T) {
		result := &VerifyResult{Status: VerifyStatusPass}
		if result.HasWarnings() {
			t.Error("通过状态不应有警告")
		}

		result.Status = VerifyStatusWarning
		if !result.HasWarnings() {
			t.Error("警告状态应有警告")
		}
	})

	t.Run("GetCriticalIssues", func(t *testing.T) {
		result := &VerifyResult{
			Issues: []VerifyIssue{
				{Severity: "low"},
				{Severity: "high"},
				{Severity: "critical"},
				{Severity: "medium"},
			},
		}

		critical := result.GetCriticalIssues()
		if len(critical) != 2 {
			t.Errorf("应返回2个严重问题，实际返回%d个", len(critical))
		}
	})
}

// TestVerify 测试成果校验
func TestVerify(t *testing.T) {
	t.Run("校验存在的文件", func(t *testing.T) {
		filePath := createTestFile(t, "package main\n\nfunc main() {}")
		verifier := NewVerifier(nil)

		result := verifier.Verify([]string{filePath}, nil)
		if result.IsFailed() {
			t.Errorf("文件存在校验应通过，问题: %v", result.Issues)
		}
	})

	t.Run("校验不存在的文件", func(t *testing.T) {
		verifier := NewVerifier(nil)

		result := verifier.Verify([]string{"/nonexistent/file.go"}, nil)
		if !result.IsFailed() {
			t.Error("不存在的文件应导致校验失败")
		}
		if len(result.Issues) == 0 {
			t.Error("应报告问题")
		}
		if result.Issues[0].Type != VerifyIssueTypeFileNotFound {
			t.Errorf("问题类型应为file_not_found，实际为%s", result.Issues[0].Type)
		}
	})

	t.Run("校验空文件", func(t *testing.T) {
		filePath := createEmptyTestFile(t)
		verifier := NewVerifier(nil)

		result := verifier.Verify([]string{filePath}, nil)
		if result.Status != VerifyStatusWarning && result.Status != VerifyStatusFail {
			t.Error("空文件应导致警告或失败")
		}

		// 查找空文件问题
		found := false
		for _, issue := range result.Issues {
			if issue.Type == VerifyIssueTypeFileEmpty {
				found = true
				break
			}
		}
		if !found {
			t.Error("应报告文件为空问题")
		}
	})

	t.Run("校验只包含空白的文件", func(t *testing.T) {
		filePath := createTestFile(t, "   \n\t\n   ")
		verifier := NewVerifier(nil)

		result := verifier.Verify([]string{filePath}, nil)
		if result.Status != VerifyStatusWarning && result.Status != VerifyStatusFail {
			t.Error("只包含空白的文件应导致警告或失败")
		}
	})

	t.Run("校验多个文件", func(t *testing.T) {
		file1 := createTestFile(t, "content1")
		file2 := createTestFile(t, "content2")

		verifier := NewVerifier(nil)
		result := verifier.Verify([]string{file1, file2}, nil)

		if result.IsFailed() {
			t.Errorf("两个存在的文件应通过校验，问题: %v", result.Issues)
		}
		if result.Passed < 2 {
			t.Errorf("应至少通过2项检查，实际通过%d项", result.Passed)
		}
	})

	t.Run("遇到第一个失败停止", func(t *testing.T) {
		config := &VerifyConfig{
			CheckFileExists:    true,
			CheckFileNonEmpty:  false, // 禁用非空检查，避免多个问题
			StopOnFirstFailure: true,
			MaxIssuesToReport:  50,
		}
		verifier := NewVerifier(config)

		result := verifier.Verify([]string{"/nonexistent1", "/nonexistent2"}, nil)
		// StopOnFirstFailure应该在第一个文件失败后停止，只报告一个文件的问题
		if len(result.Issues) > 1 {
			t.Errorf("StopOnFirstFailure应只报告一个文件的问题，实际报告%d个", len(result.Issues))
		}
	})
}

// TestVerifyWithKeywords 测试关键词校验
func TestVerifyWithKeywords(t *testing.T) {
	t.Run("关键词匹配 - all模式", func(t *testing.T) {
		content := "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}"
		filePath := createTestFile(t, content)

		config := &VerifyConfig{
			CheckFileExists:   true,
			CheckFileNonEmpty: true,
			CheckKeywords:     true,
			RequiredKeywords:  []string{"package", "func", "main"},
			KeywordMatchMode:  "all",
		}
		verifier := NewVerifier(config)

		result := verifier.Verify([]string{filePath}, nil)
		if result.IsFailed() {
			t.Errorf("包含所有关键词应通过校验，问题: %v", result.Issues)
		}
	})

	t.Run("关键词不匹配 - all模式", func(t *testing.T) {
		content := "package main\n\nfunc main() {}"
		filePath := createTestFile(t, content)

		config := &VerifyConfig{
			CheckFileExists:   true,
			CheckFileNonEmpty: true,
			CheckKeywords:     true,
			RequiredKeywords:  []string{"package", "missing_keyword"},
			KeywordMatchMode:  "all",
		}
		verifier := NewVerifier(config)

		result := verifier.Verify([]string{filePath}, nil)
		if !result.IsFailed() {
			t.Error("缺少关键词应导致校验失败")
		}

		found := false
		for _, issue := range result.Issues {
			if issue.Type == VerifyIssueTypeKeywordNotFound {
				found = true
				break
			}
		}
		if !found {
			t.Error("应报告关键词未找到问题")
		}
	})

	t.Run("关键词匹配 - any模式", func(t *testing.T) {
		content := "package main\n\nfunc main() {}"
		filePath := createTestFile(t, content)

		config := &VerifyConfig{
			CheckFileExists:   true,
			CheckFileNonEmpty: true,
			CheckKeywords:     true,
			RequiredKeywords:  []string{"package", "missing_keyword"},
			KeywordMatchMode:  "any",
		}
		verifier := NewVerifier(config)

		result := verifier.Verify([]string{filePath}, nil)
		if result.IsFailed() {
			t.Errorf("包含任一关键词应通过校验，问题: %v", result.Issues)
		}
	})

	t.Run("关键词不匹配 - any模式", func(t *testing.T) {
		content := "package main\n\nfunc main() {}"
		filePath := createTestFile(t, content)

		config := &VerifyConfig{
			CheckFileExists:   true,
			CheckFileNonEmpty: true,
			CheckKeywords:     true,
			RequiredKeywords:  []string{"missing1", "missing2"},
			KeywordMatchMode:  "any",
		}
		verifier := NewVerifier(config)

		result := verifier.Verify([]string{filePath}, nil)
		// any模式下，没有任何关键词匹配应该失败
		if result.Status == VerifyStatusPass {
			t.Error("没有任何关键词匹配应导致警告或失败")
		}
	})
}

// TestVerifyTaskCompletion 测试任务完成校验
func TestVerifyTaskCompletion(t *testing.T) {
	t.Run("无文件时的空任务", func(t *testing.T) {
		taskState := state.NewTaskState("test-id", "test goal")
		taskState.UpdateStatus("in_progress")

		verifier := NewVerifier(nil)
		result := verifier.VerifyTaskCompletion([]string{}, taskState)

		// 无文件时不应该导致失败，因为有些任务不需要创建文件
		// VerifyTaskCompletion 只检查文件，不检查任务状态
		if result.IsFailed() {
			t.Error("空文件列表不应导致校验失败")
		}
	})

	t.Run("任务已完成且文件存在", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		taskState := state.NewTaskState("test-id", "test goal")
		taskState.UpdateStatus("completed")

		verifier := NewVerifier(nil)
		result := verifier.VerifyTaskCompletion([]string{filePath}, taskState)

		if result.IsFailed() {
			t.Errorf("任务完成且文件存在应通过校验，问题: %v", result.Issues)
		}
	})

	t.Run("任务完成但文件不存在", func(t *testing.T) {
		taskState := state.NewTaskState("test-id", "test goal")
		taskState.UpdateStatus("completed")

		verifier := NewVerifier(nil)
		result := verifier.VerifyTaskCompletion([]string{"/nonexistent"}, taskState)

		if !result.IsFailed() {
			t.Error("文件不存在应导致校验失败")
		}

		// 检查严重度是否提高
		for _, issue := range result.Issues {
			if issue.Type == VerifyIssueTypeFileNotFound && issue.Severity != "critical" {
				t.Error("任务完成校验时，文件不存在应为critical严重度")
			}
		}
	})

	t.Run("任务完成但文件为空", func(t *testing.T) {
		filePath := createEmptyTestFile(t)
		taskState := state.NewTaskState("test-id", "test goal")
		taskState.UpdateStatus("completed")

		verifier := NewVerifier(nil)
		result := verifier.VerifyTaskCompletion([]string{filePath}, taskState)

		if !result.IsFailed() {
			t.Error("文件为空应导致校验失败")
		}
	})
}

// TestVerifyContent 测试内容校验
func TestVerifyContent(t *testing.T) {
	t.Run("内容关键词匹配", func(t *testing.T) {
		config := &VerifyConfig{
			CheckKeywords:    true,
			RequiredKeywords: []string{"package", "func"},
			KeywordMatchMode: "all",
		}
		verifier := NewVerifier(config)

		content := "package main\n\nfunc main() {}"
		result := verifier.VerifyContent(content, nil)

		if result.IsFailed() {
			t.Errorf("内容包含关键词应通过校验，问题: %v", result.Issues)
		}
	})

	t.Run("内容关键词不匹配", func(t *testing.T) {
		config := &VerifyConfig{
			CheckKeywords:    true,
			RequiredKeywords: []string{"missing"},
			KeywordMatchMode: "all",
		}
		verifier := NewVerifier(config)

		content := "package main\n\nfunc main() {}"
		result := verifier.VerifyContent(content, nil)

		if !result.IsFailed() {
			t.Error("内容缺少关键词应导致校验失败")
		}
	})
}

// TestCustomRules 测试自定义规则
type mockRule struct {
	name        string
	description string
	shouldFail  bool
}

func (r *mockRule) Name() string {
	return r.name
}

func (r *mockRule) Description() string {
	return r.description
}

func (r *mockRule) Execute(ctx *VerifyContext) error {
	if r.shouldFail {
		return fmt.Errorf("mock rule failed")
	}
	return nil
}

func TestCustomRules(t *testing.T) {
	t.Run("添加自定义规则", func(t *testing.T) {
		verifier := NewVerifier(nil)
		rule := &mockRule{name: "test_rule", description: "test description"}

		verifier.AddRule(rule)
		rules := verifier.GetCustomRules()

		if len(rules) != 1 {
			t.Errorf("应有1个规则，实际有%d个", len(rules))
		}
		if rules[0].Name() != "test_rule" {
			t.Errorf("规则名称应为test_rule，实际为%s", rules[0].Name())
		}
	})

	t.Run("移除自定义规则", func(t *testing.T) {
		verifier := NewVerifier(nil)
		rule := &mockRule{name: "test_rule", description: "test"}

		verifier.AddRule(rule)
		removed := verifier.RemoveRule("test_rule")

		if !removed {
			t.Error("应成功移除规则")
		}

		rules := verifier.GetCustomRules()
		if len(rules) != 0 {
			t.Errorf("移除后应有0个规则，实际有%d个", len(rules))
		}
	})

	t.Run("清除所有规则", func(t *testing.T) {
		verifier := NewVerifier(nil)
		verifier.AddRule(&mockRule{name: "rule1", description: "test"})
		verifier.AddRule(&mockRule{name: "rule2", description: "test"})

		verifier.ClearRules()
		rules := verifier.GetCustomRules()

		if len(rules) != 0 {
			t.Errorf("清除后应有0个规则，实际有%d个", len(rules))
		}
	})

	t.Run("自定义规则通过", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		config := &VerifyConfig{
			CheckFileExists:   true,
			EnableCustomRules: true,
		}
		verifier := NewVerifier(config)
		verifier.AddRule(&mockRule{name: "pass_rule", description: "should pass", shouldFail: false})

		result := verifier.Verify([]string{filePath}, nil)
		if result.IsFailed() {
			t.Errorf("规则通过应不影响校验结果，问题: %v", result.Issues)
		}
	})

	t.Run("自定义规则失败", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		config := &VerifyConfig{
			CheckFileExists:   true,
			EnableCustomRules: true,
		}
		verifier := NewVerifier(config)
		verifier.AddRule(&mockRule{name: "fail_rule", description: "should fail", shouldFail: true})

		result := verifier.Verify([]string{filePath}, nil)
		if !result.IsFailed() {
			t.Error("规则失败应导致校验失败")
		}

		found := false
		for _, issue := range result.Issues {
			if issue.Type == VerifyIssueTypeCustomRuleFailed && issue.RuleName == "fail_rule" {
				found = true
				break
			}
		}
		if !found {
			t.Error("应报告自定义规则失败")
		}
	})

	t.Run("禁用自定义规则", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		config := &VerifyConfig{
			CheckFileExists:   true,
			EnableCustomRules: false,
		}
		verifier := NewVerifier(config)
		verifier.AddRule(&mockRule{name: "fail_rule", description: "should fail", shouldFail: true})

		result := verifier.Verify([]string{filePath}, nil)
		if result.IsFailed() {
			t.Error("禁用自定义规则时，规则失败不应影响校验结果")
		}
	})
}

// TestVerifyHistory 测试校验历史
func TestVerifyHistory(t *testing.T) {
	t.Run("记录校验历史", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		verifier := NewVerifier(nil)

		verifier.Verify([]string{filePath}, nil)
		verifier.Verify([]string{"/nonexistent"}, nil)

		history := verifier.GetVerifyHistory()
		if len(history) != 2 {
			t.Errorf("应有2条历史记录，实际有%d条", len(history))
		}
	})

	t.Run("获取最后一次校验结果", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		verifier := NewVerifier(nil)

		verifier.Verify([]string{filePath}, nil)
		verifier.Verify([]string{"/nonexistent"}, nil)

		lastResult := verifier.GetLastResult()
		if lastResult == nil {
			t.Fatal("最后一次校验结果不应为nil")
		}
		if lastResult.Status != VerifyStatusFail {
			t.Errorf("最后一次校验应失败，实际状态: %s", lastResult.Status)
		}
	})

	t.Run("无历史记录时获取最后一次结果", func(t *testing.T) {
		verifier := NewVerifier(nil)
		lastResult := verifier.GetLastResult()

		if lastResult != nil {
			t.Error("无历史记录时应返回nil")
		}
	})

	t.Run("重置校验器", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		verifier := NewVerifier(nil)

		verifier.Verify([]string{filePath}, nil)
		verifier.Reset()

		history := verifier.GetVerifyHistory()
		if len(history) != 0 {
			t.Errorf("重置后应有0条历史记录，实际有%d条", len(history))
		}
	})
}

// TestVerifyConfig 测试配置管理
func TestVerifyConfig(t *testing.T) {
	t.Run("获取配置", func(t *testing.T) {
		config := &VerifyConfig{CheckFileExists: false}
		verifier := NewVerifier(config)

		retrievedConfig := verifier.GetConfig()
		if retrievedConfig.CheckFileExists {
			t.Error("获取的配置应与设置的一致")
		}
	})

	t.Run("更新配置", func(t *testing.T) {
		verifier := NewVerifier(nil)
		newConfig := &VerifyConfig{CheckFileExists: false}

		err := verifier.UpdateConfig(newConfig)
		if err != nil {
			t.Errorf("更新配置失败: %v", err)
		}

		if verifier.GetConfig().CheckFileExists {
			t.Error("配置应已更新")
		}
	})

	t.Run("更新配置为nil", func(t *testing.T) {
		verifier := NewVerifier(nil)
		err := verifier.UpdateConfig(nil)

		if err == nil {
			t.Error("更新配置为nil应返回错误")
		}
	})

	t.Run("设置必需关键词", func(t *testing.T) {
		verifier := NewVerifier(nil)
		keywords := []string{"package", "func"}

		verifier.SetRequiredKeywords(keywords)
		config := verifier.GetConfig()

		if len(config.RequiredKeywords) != 2 {
			t.Errorf("应有2个关键词，实际有%d个", len(config.RequiredKeywords))
		}
	})
}

// TestVerifyFiles 测试批量文件校验
func TestVerifyFiles(t *testing.T) {
	t.Run("批量校验文件", func(t *testing.T) {
		file1 := createTestFile(t, "content1")
		file2 := createTestFile(t, "content2")

		verifier := NewVerifier(nil)
		result := verifier.VerifyFiles([]string{file1, file2})

		if result.IsFailed() {
			t.Errorf("批量校验存在的文件应通过，问题: %v", result.Issues)
		}
	})
}

// TestCheckFileExists 测试文件存在检查
func TestCheckFileExists(t *testing.T) {
	t.Run("文件存在", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		verifier := NewVerifier(nil)

		issue := verifier.checkFileExists(filePath)
		if issue != nil {
			t.Errorf("存在的文件不应返回问题，问题: %v", issue)
		}
	})

	t.Run("文件不存在", func(t *testing.T) {
		verifier := NewVerifier(nil)

		issue := verifier.checkFileExists("/nonexistent/file.go")
		if issue == nil {
			t.Fatal("不存在的文件应返回问题")
		}
		if issue.Type != VerifyIssueTypeFileNotFound {
			t.Errorf("问题类型应为file_not_found，实际为%s", issue.Type)
		}
	})

	t.Run("路径是目录", func(t *testing.T) {
		tmpDir := t.TempDir()
		verifier := NewVerifier(nil)

		issue := verifier.checkFileExists(tmpDir)
		if issue == nil {
			t.Fatal("目录路径应返回问题")
		}
		if issue.Type != VerifyIssueTypeInvalidPath {
			t.Errorf("问题类型应为invalid_path，实际为%s", issue.Type)
		}
	})
}

// TestCheckFileNonEmpty 测试文件非空检查
func TestCheckFileNonEmpty(t *testing.T) {
	t.Run("文件非空", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		verifier := NewVerifier(nil)

		issue := verifier.checkFileNonEmpty(filePath)
		if issue != nil {
			t.Errorf("非空文件不应返回问题，问题: %v", issue)
		}
	})

	t.Run("文件为空", func(t *testing.T) {
		filePath := createEmptyTestFile(t)
		verifier := NewVerifier(nil)

		issue := verifier.checkFileNonEmpty(filePath)
		if issue == nil {
			t.Fatal("空文件应返回问题")
		}
		if issue.Type != VerifyIssueTypeFileEmpty {
			t.Errorf("问题类型应为file_empty，实际为%s", issue.Type)
		}
	})

	t.Run("文件只包含空白", func(t *testing.T) {
		filePath := createTestFile(t, "   \n\t  ")
		verifier := NewVerifier(nil)

		issue := verifier.checkFileNonEmpty(filePath)
		if issue == nil {
			t.Fatal("只包含空白的文件应返回问题")
		}
		if issue.Type != VerifyIssueTypeFileEmpty {
			t.Errorf("问题类型应为file_empty，实际为%s", issue.Type)
		}
	})
}

// TestConcurrency 测试并发安全性
func TestConcurrency(t *testing.T) {
	t.Run("并发添加规则", func(t *testing.T) {
		verifier := NewVerifier(nil)
		done := make(chan bool)

		for i := 0; i < 10; i++ {
			go func(id int) {
				verifier.AddRule(&mockRule{
					name:        fmt.Sprintf("rule_%d", id),
					description: fmt.Sprintf("rule %d", id),
				})
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			<-done
		}

		rules := verifier.GetCustomRules()
		if len(rules) != 10 {
			t.Errorf("应有10个规则，实际有%d个", len(rules))
		}
	})

	t.Run("并发校验", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		verifier := NewVerifier(nil)
		done := make(chan *VerifyResult)

		for i := 0; i < 5; i++ {
			go func() {
				result := verifier.Verify([]string{filePath}, nil)
				done <- result
			}()
		}

		for i := 0; i < 5; i++ {
			result := <-done
			if result.IsFailed() {
				t.Errorf("并发校验应通过，问题: %v", result.Issues)
			}
		}

		history := verifier.GetVerifyHistory()
		if len(history) != 5 {
			t.Errorf("应有5条历史记录，实际有%d条", len(history))
		}
	})

	t.Run("并发读写配置", func(t *testing.T) {
		verifier := NewVerifier(nil)
		done := make(chan bool)

		// 并发读
		for i := 0; i < 5; i++ {
			go func() {
				_ = verifier.GetConfig()
				done <- true
			}()
		}

		// 并发写
		for i := 0; i < 5; i++ {
			go func() {
				verifier.SetRequiredKeywords([]string{"test"})
				done <- true
			}()
		}

		for i := 0; i < 10; i++ {
			<-done
		}
	})
}

// TestTimestamp 测试时间戳
func TestTimestamp(t *testing.T) {
	filePath := createTestFile(t, "content")
	verifier := NewVerifier(nil)

	before := time.Now()
	result := verifier.Verify([]string{filePath}, nil)
	after := time.Now()

	if result.Timestamp.Before(before) || result.Timestamp.After(after) {
		t.Error("时间戳应在校验时间范围内")
	}

	for _, issue := range result.Issues {
		if issue.Timestamp.Before(before) || issue.Timestamp.After(after) {
			t.Error("问题时间戳应在校验时间范围内")
		}
	}
}

// TestSummary 测试摘要生成
func TestSummary(t *testing.T) {
	t.Run("无问题摘要", func(t *testing.T) {
		filePath := createTestFile(t, "content")
		verifier := NewVerifier(nil)

		result := verifier.Verify([]string{filePath}, nil)
		if result.Summary == "" {
			t.Error("摘要不应为空")
		}
		if !strings.Contains(result.Summary, "通过") {
			t.Errorf("摘要应包含'通过'，实际: %s", result.Summary)
		}
	})

	t.Run("有问题摘要", func(t *testing.T) {
		verifier := NewVerifier(nil)

		result := verifier.Verify([]string{"/nonexistent"}, nil)
		if result.Summary == "" {
			t.Error("摘要不应为空")
		}
		if !strings.Contains(result.Summary, "问题") {
			t.Errorf("摘要应包含'问题'，实际: %s", result.Summary)
		}
	})
}
