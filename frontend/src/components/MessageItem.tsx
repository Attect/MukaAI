import React from "react";
import type { Message } from "../types";
import ThinkingBlock from "./ThinkingBlock";
import ToolCallBlock from "./ToolCallBlock";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import rehypeHighlight from "rehype-highlight";

interface MessageItemProps {
  message: Message;
}

export default function MessageItem({ message }: MessageItemProps): React.ReactElement {
  const toolCalls = message.toolCalls || [];
  const thinking = message.thinking || "";
  const content = message.content || "";
  const isStreaming = message.isStreaming || false;
  const streamingType = message.streamingType || "";

  if (message.role === "user") {
    return (
      <div className="my-3">
        <div className="text-blue-400 text-sm font-bold mb-1">👤 User</div>
        <div className="ml-4 text-gray-200 whitespace-pre-wrap">{content}</div>
      </div>
    );
  }

  return (
    <div className="my-3">
      <div className="text-green-400 text-sm font-bold mb-1">🤖 Assistant</div>
      <div className="ml-4">
        {thinking && (
          <ThinkingBlock content={thinking} isStreaming={isStreaming && streamingType === "thinking"} />
        )}
        {toolCalls.length > 0 && toolCalls.map((tc) => (
          <ToolCallBlock key={tc.id || Math.random().toString()} toolCall={tc} isStreaming={isStreaming && streamingType === "tool"} />
        ))}
        {content && (
          <div className="markdown-body text-gray-200">
            <ReactMarkdown remarkPlugins={[remarkGfm]} rehypePlugins={[rehypeHighlight]}>
              {content}
            </ReactMarkdown>
            {isStreaming && streamingType === "content" && <span className="animate-blink text-blue-400">▌</span>}
          </div>
        )}
        {(message.tokenUsage || 0) > 0 && (
          <div className="text-gray-500 text-xs mt-1 italic">Tokens: {message.tokenUsage}</div>
        )}
      </div>
    </div>
  );
}
