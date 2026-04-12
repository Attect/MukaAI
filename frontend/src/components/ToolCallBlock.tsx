import React, { useState } from "react";
import type { ToolCall } from "../types";

interface ToolCallBlockProps {
  toolCall: ToolCall;
  isStreaming: boolean;
}

export default function ToolCallBlock({ toolCall, isStreaming }: ToolCallBlockProps): React.ReactElement {
  const [expanded, setExpanded] = useState(false);

  const formatArguments = (args: string): string => {
    try {
      return JSON.stringify(JSON.parse(args), null, 2);
    } catch {
      return args;
    }
  };

  return (
    <div className="my-2 border border-yellow-700/50 rounded bg-yellow-900/20">
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-2 w-full px-3 py-2 text-left text-yellow-400 text-sm hover:bg-yellow-900/30"
      >
        <span>{expanded ? "▼" : "▶"}</span>
        <span>🔧 {toolCall.name}</span>
        {isStreaming && !toolCall.isComplete && <span className="animate-pulse text-yellow-300">▌</span>}
        {toolCall.isComplete && <span className="text-green-400 text-xs">✓</span>}
      </button>
      {expanded && (
        <div className="px-3 pb-2">
          <div className="text-gray-400 text-xs mb-1">Parameters:</div>
          <pre className="text-gray-300 text-xs bg-gray-900/50 p-2 rounded overflow-x-auto">
            {formatArguments(toolCall.arguments)}
          </pre>
          {(toolCall.result || toolCall.resultError) && (
            <>
              <div className="text-gray-400 text-xs mt-2 mb-1">Result:</div>
              <pre className={`text-xs p-2 rounded overflow-x-auto max-h-48 overflow-y-auto ${toolCall.resultError ? "text-red-400 bg-red-900/20" : "text-green-400 bg-green-900/20"}`}>
                {toolCall.resultError || toolCall.result}
              </pre>
            </>
          )}
        </div>
      )}
    </div>
  );
}
