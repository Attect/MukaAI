import React from "react";
import type { TokenStats } from "../types";

interface ToolbarProps {
  workDir: string;
  tokenStats: TokenStats;
  isStreaming: boolean;
  onInterrupt: () => void;
  onClear: () => void;
  onToggleSidebar: () => void;
  onToggleSettings: () => void;
  onToggleTheme: () => void;
  onToggleTerminal: () => void;
  isTerminalVisible: boolean;
  isDarkTheme: boolean;
}

export default function Toolbar({
  workDir,
  tokenStats,
  isStreaming,
  onInterrupt,
  onClear,
  onToggleSidebar,
  onToggleSettings,
  onToggleTheme,
  onToggleTerminal,
  isTerminalVisible,
  isDarkTheme,
}: ToolbarProps): React.ReactElement {
  return (
    <div
      style={{
        display: "flex",
        alignItems: "center",
        justifyContent: "space-between",
        background: "var(--bg-toolbar)",
        padding: "0.5rem 1rem",
        borderBottom: "1px solid var(--border-color)",
      }}
    >
      <div style={{ display: "flex", alignItems: "center", gap: "1rem" }}>
        <button
          onClick={onToggleSidebar}
          style={{ background: "none", border: "none", color: "var(--text-muted)", cursor: "pointer", fontSize: "1.25rem" }}
          title="对话列表"
        >
          ☰
        </button>
        <span style={{ color: "var(--text-muted)", fontSize: "0.875rem" }}>📁 {workDir}</span>
      </div>
      <div style={{ display: "flex", alignItems: "center", gap: "0.75rem" }}>
        <span style={{ color: "var(--text-muted)", fontSize: "0.875rem" }}>Tokens: {tokenStats.totalTokens}</span>
        <span style={{ color: "var(--text-muted)", fontSize: "0.875rem" }}>推理: {tokenStats.inferenceCount}</span>
        {isStreaming && (
          <button
            onClick={onInterrupt}
            style={{
              background: "var(--bg-danger)",
              color: "#fff",
              border: "none",
              fontSize: "0.875rem",
              padding: "0.25rem 0.75rem",
              borderRadius: "0.375rem",
              cursor: "pointer",
            }}
            title="打断推理"
          >
            ⏹ 打断
          </button>
        )}
        <button
          onClick={onClear}
          style={{ background: "none", border: "none", color: "var(--text-muted)", cursor: "pointer", fontSize: "0.875rem" }}
          title="清空对话"
        >
          🗑 清空
        </button>
        <button
          onClick={onToggleTerminal}
          style={{
            background: isTerminalVisible ? "var(--bg-active)" : "none",
            border: isTerminalVisible ? "1px solid var(--border-color)" : "none",
            color: "var(--text-muted)",
            cursor: "pointer",
            fontSize: "1rem",
            borderRadius: "0.25rem",
            padding: "0.1rem 0.3rem",
          }}
          title={isTerminalVisible ? "关闭终端" : "打开终端"}
        >
          ⌨
        </button>
        <button
          onClick={onToggleTheme}
          style={{ background: "none", border: "none", color: "var(--text-muted)", cursor: "pointer", fontSize: "1rem" }}
          title={isDarkTheme ? "切换到亮色主题" : "切换到暗色主题"}
        >
          {isDarkTheme ? "☀️" : "🌙"}
        </button>
        <button
          onClick={onToggleSettings}
          style={{ background: "none", border: "none", color: "var(--text-muted)", cursor: "pointer", fontSize: "1rem" }}
          title="设置"
        >
          ⚙️
        </button>
      </div>
    </div>
  );
}
