import React, { useState } from "react";

interface ThinkingBlockProps {
  content: string;
  isStreaming: boolean;
}

export default function ThinkingBlock({ content, isStreaming }: ThinkingBlockProps): React.ReactElement {
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="my-2">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-2 text-gray-500 hover:text-gray-300 text-sm"
      >
        <span>{expanded ? "▼" : "▶"}</span>
        <span>💭 Thinking</span>
        {isStreaming && <span className="animate-pulse">▌</span>}
      </button>
      {expanded && (
        <div className="mt-1 ml-4 p-3 bg-gray-800/50 rounded border border-gray-700 text-gray-400 text-sm italic whitespace-pre-wrap">
          {content}
          {isStreaming && <span className="animate-pulse">▌</span>}
        </div>
      )}
    </div>
  );
}
