import React, { useState, useRef, useCallback } from "react";

interface InputAreaProps {
  isStreaming: boolean;
  onSend: (content: string) => void;
  onCommand: (command: string) => void;
}

const COMMANDS = [
  { cmd: "/cd", desc: "切换工作目录" },
  { cmd: "/clear", desc: "清空当前对话" },
  { cmd: "/save", desc: "保存对话历史" },
  { cmd: "/help", desc: "显示帮助信息" },
  { cmd: "/exit", desc: "退出应用" },
];

/** 大文件阈值 1MB */
const LARGE_FILE_THRESHOLD = 1 * 1024 * 1024;

export default function InputArea({ isStreaming, onSend, onCommand }: InputAreaProps): React.ReactElement {
  const [value, setValue] = useState("");
  const [showCommands, setShowCommands] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const dragCounterRef = useRef(0);

  const handleSubmit = useCallback(() => {
    const trimmed = value.trim();
    if (!trimmed) return;

    if (trimmed.startsWith("/")) {
      onCommand(trimmed);
    } else {
      onSend(trimmed);
    }
    setValue("");
    setShowCommands(false);
  }, [value, onSend, onCommand]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
    if (e.key === "/" && value === "") {
      setShowCommands(true);
    }
  }, [handleSubmit, value]);

  const handleCommandSelect = useCallback((cmd: string) => {
    setValue(cmd + " ");
    setShowCommands(false);
    textareaRef.current?.focus();
  }, []);

  /** 处理拖拽进入 */
  const handleDragEnter = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounterRef.current += 1;
    if (dragCounterRef.current === 1) {
      setIsDragging(true);
    }
  }, []);

  /** 处理拖拽经过 */
  const handleDragOver = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
  }, []);

  /** 处理拖拽离开 */
  const handleDragLeave = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounterRef.current -= 1;
    if (dragCounterRef.current === 0) {
      setIsDragging(false);
    }
  }, []);

  /** 处理文件放下 */
  const handleDrop = useCallback((e: React.DragEvent) => {
    e.preventDefault();
    e.stopPropagation();
    dragCounterRef.current = 0;
    setIsDragging(false);

    const files = e.dataTransfer.files;
    if (!files || files.length === 0) return;

    const paths: string[] = [];
    const largeFiles: string[] = [];

    for (let i = 0; i < files.length; i++) {
      const file = files[i];
      const path = (file as File & { path?: string }).path || file.name;
      paths.push(path);

      if (file.size > LARGE_FILE_THRESHOLD) {
        largeFiles.push(`${file.name} (${(file.size / 1024 / 1024).toFixed(1)}MB)`);
      }
    }

    // 大文件提示
    if (largeFiles.length > 0) {
      const proceed = window.confirm(
        `以下文件较大:\n${largeFiles.join("\n")}\n\n是否继续附加文件路径?`
      );
      if (!proceed) return;
    }

    // 将文件路径附加到输入框
    const pathText = paths.map((p) => `@${p}`).join(" ");
    setValue((prev) => {
      const separator = prev.trim() ? " " : "";
      return prev.trim() + separator + pathText;
    });
    textareaRef.current?.focus();
  }, []);

  return (
    <div
      style={{
        borderTop: "1px solid var(--border-color)",
        padding: "0.75rem",
        position: "relative",
      }}
      onDragEnter={handleDragEnter}
      onDragOver={handleDragOver}
      onDragLeave={handleDragLeave}
      onDrop={handleDrop}
    >
      {/* 拖拽高亮覆盖层 */}
      {isDragging && (
        <div
          style={{
            position: "absolute",
            inset: 0,
            background: "var(--drop-zone-bg, rgba(59,130,246,0.08))",
            border: "2px dashed var(--drop-zone-border, rgba(59,130,246,0.5))",
            borderRadius: "0.5rem",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            zIndex: 20,
            color: "var(--drop-zone-text, #3b82f6)",
            fontSize: "0.875rem",
            fontWeight: 600,
            pointerEvents: "none",
          }}
        >
          📎 拖放文件到此处附加文件路径
        </div>
      )}
      <div style={{ position: "relative" }}>
        <textarea
          ref={textareaRef}
          value={value}
          onChange={(e) => {
            setValue(e.target.value);
            setShowCommands(e.target.value.startsWith("/") && !e.target.value.includes(" "));
          }}
          onKeyDown={handleKeyDown}
          disabled={isStreaming}
          placeholder={isStreaming ? "推理中..." : "请输入你的问题... (Enter 发送, Shift+Enter 换行, 支持文件拖拽)"}
          style={{
            width: "100%",
            background: "var(--bg-input)",
            color: "var(--text-primary)",
            borderRadius: "0.5rem",
            padding: "0.75rem 1rem",
            resize: "none",
            outline: "none",
            border: isDragging ? "2px dashed var(--drop-zone-border, rgba(59,130,246,0.5))" : "none",
            fontFamily: "-apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', sans-serif",
            fontSize: "0.875rem",
            opacity: isStreaming ? 0.5 : 1,
            cursor: isStreaming ? "not-allowed" : "text",
            boxSizing: "border-box",
            transition: "border 0.2s",
          }}
          rows={3}
        />
        {showCommands && (
          <div style={{
            position: "absolute",
            bottom: "100%",
            left: 0,
            marginBottom: "0.25rem",
            background: "var(--bg-modal)",
            border: "1px solid var(--border-color)",
            borderRadius: "0.5rem",
            boxShadow: "0 4px 6px -1px rgba(0,0,0,0.1)",
            overflow: "hidden",
            zIndex: 10,
          }}>
            {COMMANDS.map((c) => (
              <button
                key={c.cmd}
                onClick={() => handleCommandSelect(c.cmd)}
                style={{
                  display: "block",
                  width: "100%",
                  textAlign: "left",
                  padding: "0.5rem 1rem",
                  fontSize: "0.875rem",
                  background: "transparent",
                  border: "none",
                  color: "var(--text-secondary)",
                  cursor: "pointer",
                }}
                onMouseEnter={(e) => { e.currentTarget.style.background = "var(--bg-hover)"; }}
                onMouseLeave={(e) => { e.currentTarget.style.background = "transparent"; }}
              >
                <span style={{ color: "var(--text-code)" }}>{c.cmd}</span> - {c.desc}
              </button>
            ))}
          </div>
        )}
      </div>
      <div style={{ display: "flex", justifyContent: "space-between", marginTop: "0.25rem", fontSize: "0.75rem", color: "var(--text-dim)" }}>
        <span>Enter 发送 │ Shift+Enter 换行 │ / 命令 │ 拖拽文件附加路径</span>
        <span>{value.length} 字符</span>
      </div>
    </div>
  );
}
