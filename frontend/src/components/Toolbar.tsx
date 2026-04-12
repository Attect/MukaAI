import React from "react";
import type { TokenStats } from "../types";

interface ToolbarProps {
  workDir: string;
  tokenStats: TokenStats;
  isStreaming: boolean;
  onInterrupt: () => void;
  onClear: () => void;
  onToggleSidebar: () => void;
}

export default function Toolbar({ workDir, tokenStats, isStreaming, onInterrupt, onClear, onToggleSidebar }: ToolbarProps): React.ReactElement {
  return (
    <div className="flex items-center justify-between bg-gray-800 px-4 py-2 border-b border-gray-700">
      <div className="flex items-center gap-4">
        <button onClick={onToggleSidebar} className="text-gray-400 hover:text-white" title="对话列表">
          ☰
        </button>
        <span className="text-gray-400 text-sm">📁 {workDir}</span>
      </div>
      <div className="flex items-center gap-4">
        <span className="text-gray-400 text-sm">Tokens: {tokenStats.totalTokens}</span>
        <span className="text-gray-400 text-sm">推理: {tokenStats.inferenceCount}</span>
        {isStreaming && (
          <button onClick={onInterrupt} className="bg-red-600 hover:bg-red-700 text-white text-sm px-3 py-1 rounded" title="打断推理">
            ⏹ 打断
          </button>
        )}
        <button onClick={onClear} className="text-gray-400 hover:text-white text-sm" title="清空对话">
          🗑 清空
        </button>
      </div>
    </div>
  );
}
