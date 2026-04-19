import React, { useState, useCallback, useRef, useEffect } from "react";
import type { Message } from "../types";
import ThinkingBlock from "./ThinkingBlock";
import ToolCallBlock from "./ToolCallBlock";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";

interface MessageItemProps {
  message: Message;
}

// CodeBlock 组件：带复制按钮的代码块渲染器
function CodeBlock({ className, children, ...props }: React.HTMLAttributes<HTMLElement> & { children?: React.ReactNode }) {
  const [copied, setCopied] = useState(false);
  const codeRef = useRef<HTMLElement>(null);

  const handleCopy = useCallback(() => {
    const codeEl = codeRef.current;
    if (!codeEl) return;
    const text = codeEl.textContent || "";
    navigator.clipboard.writeText(text).then(() => {
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }).catch(() => {
      // fallback: do nothing
    });
  }, []);

  // 判断是否为代码块（有语言类名或包含换行）vs 行内代码
  const match = /language-(\w+)/.exec(className || "");
  const codeString = String(children).replace(/\n$/, "");

  // 如果没有语言类名且不包含换行，视为行内代码
  if (!match && !codeString.includes("\n")) {
    return (
      <code className={className} {...props}>
        {children}
      </code>
    );
  }

  return (
    <div className="code-block-wrapper">
      <button
        className={`code-copy-btn ${copied ? "copied" : ""}`}
        onClick={handleCopy}
        type="button"
      >
        {copied ? "✓ 已复制" : "复制"}
      </button>
      <pre>
        <code ref={codeRef} className={className} {...props}>
          {children}
        </code>
      </pre>
    </div>
  );
}

export default function MessageItem({ message }: MessageItemProps): React.ReactElement {
  const toolCalls = message.toolCalls || [];
  const thinking = message.thinking || "";
  const content = message.content || "";
  const isStreaming = message.isStreaming || false;
  const streamingType = message.streamingType || "";

  // 流式输出时自动滚动
  const containerRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (isStreaming && containerRef.current) {
      containerRef.current.scrollIntoView({ behavior: "smooth", block: "end" });
    }
  }, [content, isStreaming]);

  if (message.role === "user") {
    return (
      <div style={{ margin: "0.75rem 0" }} ref={containerRef}>
        <div style={{ color: "var(--text-user)", fontSize: "0.875rem", fontWeight: 700, marginBottom: "0.25rem" }}>👤 User</div>
        <div style={{ marginLeft: "1rem", color: "var(--text-primary)", whiteSpace: "pre-wrap" }}>{content}</div>
      </div>
    );
  }

  return (
    <div style={{ margin: "0.75rem 0" }} ref={containerRef}>
      <div style={{ color: "var(--text-assistant)", fontSize: "0.875rem", fontWeight: 700, marginBottom: "0.25rem" }}>🤖 Assistant</div>
      <div style={{ marginLeft: "1rem" }}>
        {/* 1. 思考内容 */}
        {thinking && (
          <ThinkingBlock content={thinking} isStreaming={isStreaming && streamingType === "thinking"} />
        )}
        {/* 2. 正文内容 */}
        {content && (
          <div className="markdown-body" style={{ color: "var(--text-primary)" }}>
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              rehypePlugins={[rehypeHighlight]}
              components={{
                code: CodeBlock as any,
              }}
            >
              {content}
            </ReactMarkdown>
            {isStreaming && streamingType === "content" && <span className="animate-blink" style={{ color: "var(--text-user)" }}>▌</span>}
          </div>
        )}
        {/* 3. 工具调用和结果 */}
        {toolCalls.length > 0 && toolCalls.map((tc) => (
          <ToolCallBlock key={tc.id || Math.random().toString()} toolCall={tc} isStreaming={isStreaming && streamingType === "tool"} />
        ))}
        {(message.tokenUsage || 0) > 0 && (
          <div style={{ color: "var(--text-dim)", fontSize: "0.75rem", marginTop: "0.25rem", fontStyle: "italic" }}>
            Tokens: {message.tokenUsage}
          </div>
        )}
      </div>
    </div>
  );
}
