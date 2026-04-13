// Package lsp 提供LSP（Language Server Protocol）客户端实现
// 通过stdin/stdout（JSON-RPC 2.0）与语言服务器进程通信
// 第一版聚焦于核心功能：initialize、didOpen、diagnostics
package lsp

// ==================== JSON-RPC 2.0 消息定义 ====================

// JSONRPCRequest JSON-RPC 2.0 请求消息
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse JSON-RPC 2.0 响应消息
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// JSONRPCNotification JSON-RPC 2.0 通知消息（无ID，无需响应）
type JSONRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// RPCError JSON-RPC错误
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *RPCError) Error() string {
	return e.Message
}

// ==================== LSP 协议消息定义 ====================

// InitializeParams LSP initialize请求参数
type InitializeParams struct {
	ProcessID    int        `json:"processId"`
	RootURI      string     `json:"rootUri"`
	Capabilities ClientCaps `json:"capabilities"`
}

// ClientCaps 客户端能力声明
type ClientCaps struct {
	TextDocument TextDocumentClientCaps `json:"textDocument"`
}

// TextDocumentClientCaps 文本文档相关能力
type TextDocumentClientCaps struct {
	PublishDiagnostics PublishDiagnosticsCaps `json:"publishDiagnostics"`
}

// PublishDiagnosticsCaps 诊断能力
type PublishDiagnosticsCaps struct {
	RelatedInformation bool `json:"relatedInformation"`
}

// InitializeResult LSP initialize响应结果
type InitializeResult struct {
	Capabilities interface{} `json:"capabilities"`
}

// ==================== 文档通知 ====================

// DidOpenParams textDocument/didOpen 参数
type DidOpenParams struct {
	TextDocument TextDocumentItem `json:"textDocument"`
}

// TextDocumentItem 文本文档项
type TextDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

// DidChangeParams textDocument/didChange 参数
type DidChangeParams struct {
	TextDocument   VersionedTextDocumentIdentifier  `json:"textDocument"`
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// VersionedTextDocumentIdentifier 带版本的文本文档标识
type VersionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

// TextDocumentContentChangeEvent 内容变更事件
type TextDocumentContentChangeEvent struct {
	Text string `json:"text"`
}

// ==================== 诊断 ====================

// PublishDiagnosticsParams textDocument/publishDiagnostics 通知参数
type PublishDiagnosticsParams struct {
	URI         string       `json:"uri"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

// Diagnostic LSP诊断信息
type Diagnostic struct {
	Range    Range       `json:"range"`
	Severity int         `json:"severity"` // 1=Error, 2=Warning, 3=Info, 4=Hint
	Message  string      `json:"message"`
	Source   string      `json:"source"`
	Code     interface{} `json:"code,omitempty"`
}

// SeverityString 返回严重级别的字符串表示
func (d *Diagnostic) SeverityString() string {
	switch d.Severity {
	case 1:
		return "Error"
	case 2:
		return "Warning"
	case 3:
		return "Information"
	case 4:
		return "Hint"
	default:
		return "Unknown"
	}
}

// Range 范围（0-based行号和字符偏移）
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position 位置（0-based）
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}
