package mcp

import (
	"testing"
	"time"
)

// TestConfigValidation 测试配置验证
func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *MCPConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "disabled config should pass",
			config: &MCPConfig{
				Enabled: false,
			},
			wantErr: false,
		},
		{
			name: "valid stdio config",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{
						ID:        "test",
						Enabled:   true,
						Transport: "stdio",
						Command:   "echo",
					},
				},
				Security: MCPSecurityConfig{
					DefaultPolicy: "allow",
				},
			},
			wantErr: false,
		},
		{
			name: "valid http config",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{
						ID:        "remote",
						Enabled:   true,
						Transport: "http",
						URL:       "http://localhost:3001/mcp",
					},
				},
				Security: MCPSecurityConfig{
					DefaultPolicy: "allow",
				},
			},
			wantErr: false,
		},
		{
			name: "valid sse config",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{
						ID:        "remote_sse",
						Enabled:   true,
						Transport: "sse",
						URL:       "http://localhost:3001/sse",
					},
				},
				Security: MCPSecurityConfig{
					DefaultPolicy: "allow",
				},
			},
			wantErr: false,
		},
		{
			name: "duplicate server ID",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{ID: "test", Enabled: true, Transport: "stdio", Command: "echo"},
					{ID: "test", Enabled: true, Transport: "stdio", Command: "echo"},
				},
			},
			wantErr:     true,
			errContains: "重复",
		},
		{
			name: "stdio without command",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{ID: "test", Enabled: true, Transport: "stdio"},
				},
			},
			wantErr:     true,
			errContains: "command",
		},
		{
			name: "http without url",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{ID: "test", Enabled: true, Transport: "http"},
				},
			},
			wantErr:     true,
			errContains: "url",
		},
		{
			name: "sse without url",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{ID: "test", Enabled: true, Transport: "sse"},
				},
			},
			wantErr:     true,
			errContains: "url",
		},
		{
			name: "invalid transport",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{ID: "test", Enabled: true, Transport: "grpc"},
				},
			},
			wantErr:     true,
			errContains: "transport",
		},
		{
			name: "invalid server ID",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{ID: "test-server!", Enabled: true, Transport: "stdio", Command: "echo"},
				},
			},
			wantErr:     true,
			errContains: "不合法",
		},
		{
			name: "timeout too low",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{ID: "test", Enabled: true, Transport: "stdio", Command: "echo", Timeout: 1},
				},
			},
			wantErr:     true,
			errContains: "timeout",
		},
		{
			name: "timeout too high",
			config: &MCPConfig{
				Enabled: true,
				Servers: []ServerConfig{
					{ID: "test", Enabled: true, Transport: "stdio", Command: "echo", Timeout: 500},
				},
			},
			wantErr:     true,
			errContains: "timeout",
		},
		{
			name: "disabled server should be skipped",
			config: &MCPConfig{
				Enabled: true,
				Security: MCPSecurityConfig{
					DefaultPolicy: "allow",
				},
				Servers: []ServerConfig{
					{ID: "test", Enabled: false, Transport: "invalid"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid default policy",
			config: &MCPConfig{
				Enabled: true,
				Security: MCPSecurityConfig{
					DefaultPolicy: "invalid",
				},
				Servers: []ServerConfig{
					{ID: "test", Enabled: true, Transport: "stdio", Command: "echo"},
				},
			},
			wantErr:     true,
			errContains: "default_policy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && tt.errContains != "" {
				if !contains(err.Error(), tt.errContains) {
					t.Errorf("Validate() error = %v, want containing %v", err, tt.errContains)
				}
			}
		})
	}
}

// TestGetTimeout 测试超时配置
func TestGetTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		want    time.Duration
	}{
		{"zero returns default", 0, 30 * time.Second},
		{"negative returns default", -1, 30 * time.Second},
		{"normal value", 60, 60 * time.Second},
		{"minimum allowed", 5, 5 * time.Second},
		{"maximum allowed", 300, 300 * time.Second},
		{"over maximum capped", 500, 300 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := ServerConfig{Timeout: tt.timeout}
			got := cfg.GetTimeout()
			if got != tt.want {
				t.Errorf("GetTimeout() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGetEnabledServers 测试获取启用的Server列表
func TestGetEnabledServers(t *testing.T) {
	config := &MCPConfig{
		Servers: []ServerConfig{
			{ID: "a", Enabled: true},
			{ID: "b", Enabled: false},
			{ID: "c", Enabled: true},
		},
	}

	enabled := config.GetEnabledServers()
	if len(enabled) != 2 {
		t.Fatalf("GetEnabledServers() returned %d servers, want 2", len(enabled))
	}
	if enabled[0].ID != "a" || enabled[1].ID != "c" {
		t.Errorf("GetEnabledServers() returned wrong IDs: %v", enabled)
	}
}

// TestSecurityChecker 测试安全检查器
func TestSecurityChecker(t *testing.T) {
	config := &MCPSecurityConfig{
		DefaultPolicy: "confirm",
		DenyTools:     []string{"mcp_*_exec_*"},
		ConfirmTools:  []string{"mcp_*_delete_*"},
		AllowTools:    []string{"mcp_filesystem_*"},
	}

	checker := NewMCPSecurityChecker(config)

	tests := []struct {
		toolName string
		want     SecurityPolicy
	}{
		{"mcp_test_exec_cmd", SecurityPolicyDeny},       // 匹配deny模式
		{"mcp_filesystem_read", SecurityPolicyAllow},    // 匹配allow模式
		{"mcp_db_delete_row", SecurityPolicyConfirm},    // 匹配confirm模式
		{"mcp_other_tool", SecurityPolicyConfirm},       // 默认策略confirm
		{"mcp_test_exec_something", SecurityPolicyDeny}, // 匹配deny模式
		{"mcp_filesystem_write", SecurityPolicyAllow},   // 匹配allow模式
	}

	for _, tt := range tests {
		t.Run(tt.toolName, func(t *testing.T) {
			got := checker.CheckTool(tt.toolName)
			if got != tt.want {
				t.Errorf("CheckTool(%s) = %v, want %v", tt.toolName, got, tt.want)
			}
		})
	}
}

// TestSecurityCheckerDenyPriority 测试deny策略的优先级
func TestSecurityCheckerDenyPriority(t *testing.T) {
	// 工具同时在deny和allow中，deny应优先
	config := &MCPSecurityConfig{
		DefaultPolicy: "allow",
		DenyTools:     []string{"mcp_test_*"},
		AllowTools:    []string{"mcp_test_read"},
	}

	checker := NewMCPSecurityChecker(config)

	// mcp_test_read同时在deny和allow中，deny优先
	got := checker.CheckTool("mcp_test_read")
	if got != SecurityPolicyDeny {
		t.Errorf("CheckTool(mcp_test_read) = %v, want deny (deny should have priority)", got)
	}
}

// TestSecurityCheckerNil 测试nil安全配置
func TestSecurityCheckerNil(t *testing.T) {
	checker := NewMCPSecurityChecker(nil)
	got := checker.CheckTool("any_tool")
	if got != SecurityPolicyAllow {
		t.Errorf("CheckTool with nil config should return allow, got %v", got)
	}
}

// TestWildcardMatch 测试通配符匹配
func TestWildcardMatch(t *testing.T) {
	tests := []struct {
		pattern string
		text    string
		want    bool
	}{
		{"mcp_*_delete_*", "mcp_fs_delete_file", true},
		{"mcp_*_delete_*", "mcp_delete_file", false},
		{"mcp_filesystem_*", "mcp_filesystem_read", true},
		{"mcp_filesystem_*", "mcp_filesystem_", true},
		{"mcp_filesystem_*", "mcp_other_read", false},
		{"*", "anything", true},
		{"exact", "exact", true},
		{"exact", "other", false},
		{"mcp_*", "mcp_filesystem_read", true},
		{"*_exec_*", "mcp_test_exec_cmd", true},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.text, func(t *testing.T) {
			got := matchToolPattern(tt.pattern, tt.text)
			if got != tt.want {
				t.Errorf("matchToolPattern(%s, %s) = %v, want %v", tt.pattern, tt.text, got, tt.want)
			}
		})
	}
}

// TestToolNameHelpers 测试工具名辅助函数
func TestToolNameHelpers(t *testing.T) {
	// Test IsMCPToolName
	tests := []struct {
		name string
		want bool
	}{
		{"mcp_filesystem_read", true},
		{"mcp_test_exec", true},
		{"read_file", false},
		{"mcp_", false},
		{"filesystem_read", false},
	}
	for _, tt := range tests {
		got := IsMCPToolName(tt.name)
		if got != tt.want {
			t.Errorf("IsMCPToolName(%s) = %v, want %v", tt.name, got, tt.want)
		}
	}

	// Test ExtractServerID
	serverIDs := map[string]string{
		"mcp_filesystem_read": "filesystem",
		"mcp_db_query":        "db",
		"read_file":           "",
		"mcp_":                "",
	}
	for toolName, want := range serverIDs {
		got := ExtractServerID(toolName)
		if got != want {
			t.Errorf("ExtractServerID(%s) = %v, want %v", toolName, got, want)
		}
	}

	// Test ExtractOriginalToolName
	originalNames := map[string]string{
		"mcp_filesystem_read": "read",
		"mcp_db_query":        "query",
		"read_file":           "read_file",
	}
	for toolName, want := range originalNames {
		got := ExtractOriginalToolName(toolName)
		if got != want {
			t.Errorf("ExtractOriginalToolName(%s) = %v, want %v", toolName, got, want)
		}
	}
}

// TestConvertMCPResult 测试MCP结果转换
func TestConvertMCPResult(t *testing.T) {
	// 测试nil结果
	result := convertMCPResult(nil, "test")
	if result.Success {
		t.Error("convertMCPResult with nil should return error result")
	}
	if result.Error == "" {
		t.Error("convertMCPResult with nil should have error message")
	}
}

// helper
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
