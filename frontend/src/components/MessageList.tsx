import React, { useRef, useEffect } from "react";
import type { Message } from "../types";
import MessageItem from "./MessageItem";

interface MessageListProps {
  messages: Message[];
}

export default function MessageList({ messages }: MessageListProps): React.ReactElement {
  const bottomRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages]);

  return (
    <div className="flex-1 overflow-y-auto px-4 py-2">
      {messages.length === 0 && (
        <div className="flex items-center justify-center h-full text-gray-500">
          <p>开始新的对话...</p>
        </div>
      )}
      {messages.map((msg, i) => (
        <MessageItem key={i} message={msg} />
      ))}
      <div ref={bottomRef} />
    </div>
  );
}
