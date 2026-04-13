import React, { useRef, useEffect, useCallback } from "react";
import { useVirtualizer } from "@tanstack/react-virtual";
import type { Message } from "../types";
import MessageItem from "./MessageItem";

interface MessageListProps {
  messages: Message[];
}

/** 估算每条消息的行高度（用于虚拟滚动初始估算） */
function estimateSize(index: number, messages: Message[]): number {
  const msg = messages[index];
  if (!msg) return 80;

  // 基础高度
  let height = 48; // 头部 + padding

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

export default function MessageList({ messages }: MessageListProps): React.ReactElement {
  const parentRef = useRef<HTMLDivElement>(null);

  const virtualizer = useVirtualizer({
    count: messages.length,
    getScrollElement: () => parentRef.current,
    estimateSize: (index: number) => estimateSize(index, messages),
    overscan: 8,
    // 动态测量实际高度
    measureElement: (el) => el?.getBoundingClientRect().height ?? 80,
  });

  // 追踪用户是否手动滚动（用于判断是否自动滚到底部）
  const isAutoScrollRef = useRef(true);

  // 新消息到来时自动滚动到底部
  useEffect(() => {
    if (messages.length > 0 && isAutoScrollRef.current) {
      // 使用requestAnimationFrame确保DOM已更新
      requestAnimationFrame(() => {
        virtualizer.scrollToIndex(messages.length - 1, { align: "end", behavior: "smooth" });
      });
    }
  }, [messages.length, virtualizer]);

  // 检测用户手动滚动
  const handleScroll = useCallback(() => {
    const el = parentRef.current;
    if (!el) return;
    const isAtBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 60;
    isAutoScrollRef.current = isAtBottom;
  }, []);

  // 空消息占位
  if (messages.length === 0) {
    return (
      <div style={{ flex: 1, display: "flex", alignItems: "center", justifyContent: "center", color: "var(--text-dim)" }}>
        <p>开始新的对话...</p>
      </div>
    );
  }

  const virtualItems = virtualizer.getVirtualItems();

  return (
    <div
      ref={parentRef}
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
          const msg = messages[virtualRow.index];
          return (
            <div
              key={virtualRow.index}
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
              <MessageItem message={msg} />
            </div>
          );
        })}
      </div>
    </div>
  );
}
