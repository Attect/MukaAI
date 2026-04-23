import React, { useRef, useEffect, useCallback } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import type { MessageListItem, Message, CompressionEvent } from "../types";
import MessageItem from "./MessageItem";

interface MessageListProps {
  items: MessageListItem[];
  isStreaming?: boolean;
}

/** 估算每条消息的行高度（用于虚拟滚动初始估算） */
function estimateSize(index: number, items: MessageListItem[]): number {
  const item = items[index];
  if (!item) return 80;

  if (item.type === "compression") {
    return 60; // 压缩事件高度较小
  }

  const msg = item.data as Message;
  // 基础高度
  let height = 48; // 头部 + padding

  // 空消息（占位符）至少60px，确保虚拟滚动可见
  if (msg.role === "assistant" && !msg.content && !msg.thinking && (!msg.toolCalls || msg.toolCalls.length === 0)) {
    return Math.max(height, 60);
  }

  // 思考内容
  if (msg.thinking) {
    height += Math.min(msg.thinking.length / 2, 200);
  }

  // 工具调用
  const toolCalls = msg.toolCalls || [];
  height += toolCalls.length * 60;

  // 正文内容
  if (msg.content) {
    // 按换行符估算行数
    const lines = msg.content.split("\n").length;
    // 代码块等长内容需要更多空间
    const hasCodeBlock = msg.content.includes("```");
    height += Math.max(lines * 20, hasCodeBlock ? 200 : 0);
    height += Math.min(msg.content.length / 4, 600);
  }

  // 上下限
  return Math.max(60, Math.min(height, 1200));
}

/** 压缩事件组件 */
function CompressionItem({ data }: { data: CompressionEvent }): React.ReactElement {
  return (
    <div style={{
      margin: "0.5rem 0",
      padding: "0.5rem 1rem",
      background: "var(--bg-code)",
      borderLeft: "3px solid var(--text-accent)",
      borderRadius: "0.25rem",
      fontSize: "0.875rem",
      color: "var(--text-muted)",
    }}>
      <div style={{ fontWeight: 600, marginBottom: "0.25rem", color: "var(--text-accent)" }}>
        📦 上下文压缩
      </div>
      <div>
        消息: {data.originalCount} → {data.compressedCount} | 
        Tokens: {data.originalTokens} → {data.compressedTokens} 
        ({data.compressionRatio.toFixed(1)}%)
      </div>
      {data.summary && (
        <div style={{ marginTop: "0.25rem", fontSize: "0.75rem", opacity: 0.8 }}>
          摘要: {data.summary.length > 100 ? data.summary.slice(0, 100) + "..." : data.summary}
        </div>
      )}
    </div>
  );
}

export default function MessageList({ items }: MessageListProps): React.ReactElement {
  const parentRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => parentRef.current,
    estimateSize: (index: number) => estimateSize(index, items),
    overscan: 12,
    // 动态测量实际高度
    measureElement: (el) => el?.getBoundingClientRect().height ?? 80,
  });

  // 追踪用户是否手动滚动（用于判断是否自动滚到底部）
  const isAutoScrollRef = useRef(true);

  // 新消息到来时自动滚动到底部
  useEffect(() => {
    if (items.length > 0 && isAutoScrollRef.current) {
      // 使用requestAnimationFrame确保DOM已更新
      requestAnimationFrame(() => {
        virtualizer.scrollToIndex(items.length - 1, { align: "end", behavior: "smooth" });
      });
    }
  }, [items.length, virtualizer]);

  // 检测用户手动滚动
  const handleScroll = useCallback(() => {
    const el = parentRef.current;
    if (!el) return;
    const isAtBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 60;
    isAutoScrollRef.current = isAtBottom;
  }, []);

  // 空消息占位
  if (items.length === 0) {
    return (
      <div role="log" aria-label="对话消息" aria-live="polite" style={{ flex: 1, display: "flex", alignItems: "center", justifyContent: "center", color: "var(--text-dim)" }}>
        <p>开始新的对话...</p>
      </div>
    );
  }

  const virtualItems = virtualizer.getVirtualItems();

  return (
    <div
      ref={parentRef}
      role="log"
      aria-label="对话消息"
      aria-live="polite"
      onScroll={handleScroll}
      style={{
        flex: 1,
        overflowY: "auto",
        padding: "0.5rem 1rem",
        contain: "strict",
      }}
    >
      <div
        style={{
          height: virtualizer.getTotalSize(),
          width: "100%",
          position: "relative",
        }}
      >
        {virtualItems.map((virtualRow) => {
          const item = items[virtualRow.index];
          return (
            <div
              key={item.key}
              data-index={virtualRow.index}
              ref={virtualizer.measureElement}
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "100%",
                transform: `translateY(${virtualRow.start}px)`,
              }}
            >
              {item.type === "message" ? (
                <MessageItem message={item.data as Message} />
              ) : (
                <CompressionItem data={item.data as CompressionEvent} />
              )}
            </div>
          );
        })}
      </div>
    </div>
  );
}
