// Package agent 模型输出重复检测器
// 用于实时检测模型流式输出中的重复模式，防止无限重复消耗资源
package agent

import (
	"fmt"
	"strings"
)

// RepetitionConfig 重复检测配置
// 定义重复检测器的各项参数阈值
type RepetitionConfig struct {
	// WindowSize 滑动窗口大小（检测最近N个chunk）
	WindowSize int
	// MinPatternLength 最小重复模式长度（字符数），短于此长度的chunk不参与模式检测
	MinPatternLength int
	// RepetitionThreshold 重复次数阈值（超过此值判定为重复）
	RepetitionThreshold int
	// MaxContentLength 最大内容长度（超过此值启动强制后置检测）
	MaxContentLength int
	// MaxRetries 最大重试次数（超过后放弃重试，跳过当前迭代）
	MaxRetries int
}

// DefaultRepetitionConfig 返回默认的重复检测配置
func DefaultRepetitionConfig() RepetitionConfig {
	return RepetitionConfig{
		WindowSize:          20,
		MinPatternLength:    10,
		RepetitionThreshold: 8,
		MaxContentLength:    50000,
		MaxRetries:          3,
	}
}

// RepetitionDetector 重复检测器
// 用于实时检测模型流式输出中的重复模式。
// 采用滑动窗口机制，在流式接收过程中逐chunk检测重复，
// 同时支持对完整内容进行后置检测。
type RepetitionDetector struct {
	config      RepetitionConfig // 检测配置
	contentBuf  strings.Builder  // 累积的内容缓冲区
	chunkWindow []string         // 滑动窗口，存储最近的chunk内容
	chunkCount  int              // 已处理的chunk总数
	patternHits map[string]int   // 模式命中计数（在当前窗口内的重复次数）
	detected    bool             // 是否已检测到重复
	pattern     string           // 检测到的重复模式
}

// NewRepetitionDetector 创建新的重复检测器
// config: 检测配置，如果传入零值则使用默认配置
func NewRepetitionDetector(config RepetitionConfig) *RepetitionDetector {
	// 对零值字段使用默认值
	cfg := config
	if cfg.WindowSize <= 0 {
		cfg.WindowSize = 20
	}
	if cfg.MinPatternLength <= 0 {
		cfg.MinPatternLength = 10
	}
	if cfg.RepetitionThreshold <= 0 {
		cfg.RepetitionThreshold = 8
	}
	if cfg.MaxContentLength <= 0 {
		cfg.MaxContentLength = 50000
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = 3
	}

	return &RepetitionDetector{
		config:      cfg,
		chunkWindow: make([]string, 0, cfg.WindowSize),
		patternHits: make(map[string]int),
	}
}

// Feed 输入一个流式chunk进行实时检测
// 每次新chunk到来时，检查其是否与窗口内的chunk构成重复模式。
// 当最近WindowSize个chunk中有超过RepetitionThreshold个相同内容时，判定为重复。
// 返回值:
//   - detected: 是否检测到重复
//   - pattern: 检测到的重复模式内容
func (d *RepetitionDetector) Feed(chunk string) (detected bool, pattern string) {
	if d.detected {
		return true, d.pattern
	}

	d.chunkCount++
	d.contentBuf.WriteString(chunk)

	// 短chunk不参与模式检测（避免误判常见短文本如换行符）
	if len(chunk) < d.config.MinPatternLength {
		// 仍然加入窗口以维持窗口大小，但不计入模式命中
		d.pushWindow(chunk, false)
		return false, ""
	}

	// 当前chunk计入模式命中
	d.patternHits[chunk]++
	d.pushWindow(chunk, true)

	// 检查是否超过重复阈值
	if d.patternHits[chunk] >= d.config.RepetitionThreshold {
		d.detected = true
		d.pattern = chunk
		return true, chunk
	}

	return false, ""
}

// pushWindow 将chunk推入滑动窗口
// tracked: 该chunk是否参与模式命中计数
func (d *RepetitionDetector) pushWindow(chunk string, tracked bool) {
	// 如果窗口已满，移除最旧的chunk
	if len(d.chunkWindow) >= d.config.WindowSize {
		oldest := d.chunkWindow[0]
		d.chunkWindow = d.chunkWindow[1:]

		// 减少被移除chunk的模式命中计数
		if len(oldest) >= d.config.MinPatternLength {
			if count, ok := d.patternHits[oldest]; ok {
				count--
				if count <= 0 {
					delete(d.patternHits, oldest)
				} else {
					d.patternHits[oldest] = count
				}
			}
		}
	}

	d.chunkWindow = append(d.chunkWindow, chunk)
}

// Reset 重置检测器状态，准备新一轮检测
// 每次新的模型调用开始前应调用此方法
func (d *RepetitionDetector) Reset() {
	d.contentBuf.Reset()
	d.chunkWindow = d.chunkWindow[:0]
	d.chunkCount = 0
	d.patternHits = make(map[string]int)
	d.detected = false
	d.pattern = ""
}

// CheckFullContent 对完整内容进行后置重复检测
// 适用于流式检测未触发但内容异常长的情况。
// 通过检测连续重复的子串来判断是否存在重复。
// 返回值:
//   - detected: 是否检测到重复
//   - pattern: 检测到的重复模式
//   - repetitionCount: 重复次数
func (d *RepetitionDetector) CheckFullContent(content string) (detected bool, pattern string, repetitionCount int) {
	contentLen := len(content)

	// 内容不够长，无需检测
	if contentLen < d.config.MaxContentLength {
		return false, "", 0
	}

	// 使用递增步长检测重复模式
	// 从较大的模式长度开始检测，提高效率
	minPattern := d.config.MinPatternLength
	if minPattern < 10 {
		minPattern = 10
	}

	// 最大检测模式长度：内容的1/4，避免不必要的长模式检测
	maxPatternLen := contentLen / 4
	if maxPatternLen > 1000 {
		maxPatternLen = 1000
	}

	// 按不同步长采样检测，平衡精度和性能
	for patternLen := minPattern; patternLen <= maxPatternLen; patternLen += d.calcStep(patternLen) {
		// 从内容末尾向前寻找重复模式（重复通常出现在输出尾部）
		tail := content[contentLen-patternLen:]
		if len(tail) < patternLen {
			continue
		}

		// 计算该模式在内容中出现的次数
		count := strings.Count(content, tail)
		if count >= d.config.RepetitionThreshold {
			// 截取模式的前100个字符作为摘要，避免日志中出现过长的模式
			patternPreview := tail
			if len(patternPreview) > 100 {
				patternPreview = patternPreview[:100] + "..."
			}
			return true, patternPreview, count
		}
	}

	return false, "", 0
}

// calcStep 根据模式长度计算检测步长
// 较短的模式使用小步长以提高精度，较长的模式使用大步长以提高效率
func (d *RepetitionDetector) calcStep(patternLen int) int {
	switch {
	case patternLen < 50:
		return 5
	case patternLen < 200:
		return 20
	default:
		return 50
	}
}

// BuildRetryPrompt 生成注入到对话历史中的防重复提示
// 告知模型其输出被检测为重复，要求改变输出策略
// pattern: 检测到的重复模式
// retryCount: 当前重试次数（第几次重试）
func (d *RepetitionDetector) BuildRetryPrompt(pattern string, retryCount int) string {
	// 截取模式预览，避免注入过长内容
	patternPreview := pattern
	if len(patternPreview) > 80 {
		patternPreview = patternPreview[:80] + "..."
	}

	return fmt.Sprintf(
		"[系统警告] 你的上一次输出被检测到严重的重复模式，已被丢弃。"+
			"重复内容类似: %q。"+
			"这是第%d次重试。请务必改变你的输出方式，不要重复相同的内容块。"+
			"请直接给出不同的、简洁的回答，避免任何形式的循环重复输出。"+
			"如果之前正在执行工具调用，请继续执行不同的操作。",
		patternPreview,
		retryCount,
	)
}

// GetContent 获取当前累积的内容
// 当检测到重复时，可用于获取重复前的有效内容
func (d *RepetitionDetector) GetContent() string {
	return d.contentBuf.String()
}

// IsDetected 返回是否已检测到重复
func (d *RepetitionDetector) IsDetected() bool {
	return d.detected
}

// GetConfig 返回当前配置的副本
func (d *RepetitionDetector) GetConfig() RepetitionConfig {
	return d.config
}
