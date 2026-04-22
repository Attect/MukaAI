import React, { useState, useEffect, useRef } from "react";
import { useUI } from "../contexts/UIContext";

interface ThinkingBlockProps {
  content: string;
  isStreaming: boolean;
}

export default function ThinkingBlock({ content, isStreaming }: ThinkingBlockProps): React.ReactElement {
  const { settings } = useUI();
  const [expanded, setExpanded] = useState(settings.defaultExpandThinking);
  const userToggledRef = useRef(false);

  // 当设置变化且用户未手动交互时，更新默认展开状态
  useEffect(() => {
    if (!userToggledRef.current) {
      setExpanded(settings.defaultExpandThinking);
    }
  }, [settings.defaultExpandThinking]);

  const handleToggle = () => {
    userToggledRef.current = true;
    setExpanded((prev) => !prev);
  };

  return (
    <div className="my-2">
      <button
        onClick={handleToggle}
        className="flex items-center gap-2 text-blue-300 hover:text-blue-100 text-sm font-medium"
      >
        <span>{expanded ? "▼" : "▶"}</span>
        <span>💭 Thinking</span>
        {isStreaming && <span className="animate-pulse">▌</span>}
      </button>
      {expanded && (
        <div className="mt-1 ml-4 p-3 bg-blue-950/60 rounded border border-blue-800/70 text-blue-200 text-sm italic whitespace-pre-wrap">
          {content}
          {isStreaming && <span className="animate-pulse">▌</span>}
        </div>
      )}
    </div>
  );
}
