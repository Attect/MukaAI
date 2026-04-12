import React from "react";
import type { Conversation } from "../types";

interface SidebarProps {
  visible: boolean;
  conversations: Conversation[];
  activeId: string;
  onSelect: (id: string) => void;
  onClose: () => void;
  onNew: () => void;
}

export default function Sidebar({ visible, conversations, activeId, onSelect, onClose, onNew }: SidebarProps): React.ReactElement {
  if (!visible) return <></>;

  return (
    <div className="w-64 bg-gray-800 border-r border-gray-700 flex flex-col">
      <div className="flex items-center justify-between px-4 py-3 border-b border-gray-700">
        <h2 className="text-gray-200 font-bold text-sm">对话列表</h2>
        <button onClick={onClose} className="text-gray-400 hover:text-white">✕</button>
      </div>
      <button onClick={onNew} className="mx-3 my-2 bg-blue-600 hover:bg-blue-700 text-white text-sm py-2 rounded">
        + 新对话
      </button>
      <div className="flex-1 overflow-y-auto">
        {conversations.map((conv) => (
          <button
            key={conv.id}
            onClick={() => onSelect(conv.id)}
            className={`block w-full text-left px-4 py-2 text-sm border-b border-gray-700/50 ${
              conv.id === activeId ? "bg-gray-700 text-white" : "text-gray-400 hover:bg-gray-700/50"
            }`}
          >
            <div className="truncate">{conv.title}</div>
            <div className="text-xs text-gray-500 mt-1">
              {conv.status === "active" ? "🔄" : "✓"} {conv.messageCount} 条消息
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}
