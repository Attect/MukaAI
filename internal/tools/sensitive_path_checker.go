package tools

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// PathCheckLevel 路径检查结果级别
type PathCheckLevel string

const (
	// PathCheckPass 路径安全，无需额外操作
	PathCheckPass PathCheckLevel = "pass"
	// PathCheckWarn 路径敏感但允许操作，附加警告
	PathCheckWarn PathCheckLevel = "warn"
	// PathCheckDeny 路径高度敏感，阻止操作
	PathCheckDeny PathCheckLevel = "deny"
)

// SensitivePathCheckResult 敏感路径检查结果
type SensitivePathCheckResult struct {
	// Level 检查级别
	Level PathCheckLevel `json:"level"`
	// Reason 原因说明
	Reason string `json:"reason,omitempty"`
	// Path 匹配到的敏感路径模式
	MatchedPattern string `json:"matched_pattern,omitempty"`
}

// unixDenyPaths Unix系统上拒绝访问的路径前缀（精确匹配或前缀匹配）
var unixDenyPaths = []string{
	"/etc/passwd",
	"/etc/shadow",
	"/etc/sudoers",
	"/etc/ssh/",
	"/boot/",
	"/proc/",
	"/sys/",
}

// windowsDenyPaths Windows系统上拒绝访问的路径前缀
var windowsDenyPaths = []string{
	`c:\windows\system32\`,
	`c:\windows\syswow64\`,
}

// crossPlatformWarnPaths 跨平台警告路径（匹配路径末尾部分）
var crossPlatformWarnPaths = []string{
	".ssh/authorized_keys",
	".ssh/authorized_keys2",
	".gnupg/",
}

// CheckSensitivePath 检查目标路径是否为系统敏感路径
// 返回检查结果，包含级别和原因
// 策略：
//   - deny: 阻止操作（系统关键文件）
//   - warn: 附加警告但允许（用户密钥等）
//   - pass: 安全路径
func CheckSensitivePath(requestedPath string) *SensitivePathCheckResult {
	// 清理路径以便统一比较
	cleaned := filepath.Clean(requestedPath)
	// 统一使用正斜杠用于模式匹配
	normalized := filepath.ToSlash(cleaned)
	normalizedLower := strings.ToLower(normalized)

	if runtime.GOOS == "windows" {
		return checkWindowsSensitivePath(cleaned, normalizedLower)
	}
	return checkUnixSensitivePath(cleaned, normalized, normalizedLower)
}

// checkUnixSensitivePath Unix系统敏感路径检查
func checkUnixSensitivePath(cleaned, normalized, normalizedLower string) *SensitivePathCheckResult {
	// 检查拒绝路径
	for _, denyPath := range unixDenyPaths {
		denyNorm := filepath.ToSlash(filepath.Clean(denyPath))
		denyLower := strings.ToLower(denyNorm)

		if isPathOrPrefixMatch(normalizedLower, denyLower) {
			return &SensitivePathCheckResult{
				Level:          PathCheckDeny,
				Reason:         fmt.Sprintf("目标路径 '%s' 是系统关键路径，禁止操作", cleaned),
				MatchedPattern: denyPath,
			}
		}
	}

	// 检查跨平台警告路径
	if warnResult := checkCrossPlatformWarnPaths(normalizedLower); warnResult != nil {
		warnResult.Reason = fmt.Sprintf("目标路径 '%s' 涉及安全凭证，请谨慎操作", cleaned)
		return warnResult
	}

	return &SensitivePathCheckResult{Level: PathCheckPass}
}

// checkWindowsSensitivePath Windows系统敏感路径检查
func checkWindowsSensitivePath(cleaned, normalizedLower string) *SensitivePathCheckResult {
	// 检查Windows拒绝路径
	for _, denyPath := range windowsDenyPaths {
		denyNorm := strings.ToLower(filepath.ToSlash(filepath.Clean(denyPath)))
		// 对于Windows路径，使用反斜杠和正斜杠双重匹配
		denyBackslash := strings.ReplaceAll(denyPath, `/`, `\`)
		pathLower := strings.ToLower(cleaned)

		if strings.HasPrefix(pathLower, denyBackslash) ||
			isPathOrPrefixMatch(normalizedLower, denyNorm) {
			return &SensitivePathCheckResult{
				Level:          PathCheckDeny,
				Reason:         fmt.Sprintf("目标路径 '%s' 是系统关键路径，禁止操作", cleaned),
				MatchedPattern: denyPath,
			}
		}
	}

	// 检查跨平台警告路径
	if warnResult := checkCrossPlatformWarnPaths(normalizedLower); warnResult != nil {
		warnResult.Reason = fmt.Sprintf("目标路径 '%s' 涉及安全凭证，请谨慎操作", cleaned)
		return warnResult
	}

	return &SensitivePathCheckResult{Level: PathCheckPass}
}

// checkCrossPlatformWarnPaths 检查跨平台警告路径
func checkCrossPlatformWarnPaths(normalizedLower string) *SensitivePathCheckResult {
	for _, warnPath := range crossPlatformWarnPaths {
		warnLower := strings.ToLower(warnPath)
		// 匹配路径末尾部分（可能出现在任意目录层级下）
		if strings.Contains(normalizedLower, warnLower) ||
			strings.HasSuffix(normalizedLower, strings.TrimSuffix(warnLower, "/")) {
			return &SensitivePathCheckResult{
				Level:          PathCheckWarn,
				MatchedPattern: warnPath,
			}
		}
	}
	return nil
}

// isPathOrPrefixMatch 检查路径是否精确匹配或前缀匹配
func isPathOrPrefixMatch(pathLower, patternLower string) bool {
	// 精确匹配
	if pathLower == patternLower {
		return true
	}
	// 前缀匹配（需要确保是路径边界，避免 /etc/passwd 匹配 /etc/passwd_backup）
	// 如果pattern以/结尾，直接前缀匹配即可
	if strings.HasSuffix(patternLower, "/") {
		return strings.HasPrefix(pathLower, patternLower)
	}
	// 否则需要确保前缀匹配后有路径分隔符或精确到边界
	if strings.HasPrefix(pathLower, patternLower) {
		remaining := pathLower[len(patternLower):]
		if remaining == "" || remaining[0] == '/' {
			return true
		}
	}
	return false
}
