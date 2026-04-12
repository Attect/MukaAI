import React, { useState } from "react";
import type { ToolCall } from "../types";

interface ToolResultBlockProps {
  toolCall: ToolCall;
}

export default function ToolResultBlock({ toolCall }: ToolResultBlockProps): React.ReactElement {
  const [expanded, setExpanded] = useState(false);
  const result = toolCall.resultError || toolCall.result;
  const isError = !!toolCall.resultError;
  const lines = result.split("\n");
  const isLong = lines.length > 5;

  return (
    <div className="my-1">
      {isLong && !expanded ? (
        <div>
          <pre className={`text-xs p-2 rounded ${isError ? "text-red-400 bg-red-900/20" : "text-green-400 bg-green-900/20"}`}>
            {lines.slice(0, 5).join("\n")}
          </pre>
          <button
            onClick={() => setExpanded(true)}
            className="text-gray-500 hover:text-gray-300 text-xs mt-1"
          >
            ▼ 展开完整结果 ({lines.length - 5} more lines)
          </button>
        </div>
      ) : (
        <pre className={`text-xs p-2 rounded max-h-64 overflow-y-auto ${isError ? "text-red-400 bg-red-900/20" : "text-green-400 bg-green-900/20"}`}>
          {result}
        </pre>
      )}
      {expanded && isLong && (
        <button
          onClick={() => setExpanded(false)}
          className="text-gray-500 hover:text-gray-300 text-xs mt-1"
        >
          ▲ 折叠
        </button>
      )}
    </div>
  );
}
