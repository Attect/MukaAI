import React, { useState, useRef, useCallback } from "react";

interface InputAreaProps {
  isStreaming: boolean;
  onSend: (content: string) => void;
  onCommand: (command: string) => void;
}

const COMMANDS = [
  { cmd: "/cd", desc: "切换工作目录" },
  { cmd: "/clear", desc: "清空当前对话" },
  { cmd: "/save", desc: "保存对话历史" },
  { cmd: "/help", desc: "显示帮助信息" },
  { cmd: "/exit", desc: "退出应用" },
];

export default function InputArea({ isStreaming, onSend, onCommand }: InputAreaProps): React.ReactElement {
  const [value, setValue] = useState("");
  const [showCommands, setShowCommands] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);

  const handleSubmit = useCallback(() => {
    const trimmed = value.trim();
    if (!trimmed) return;

    if (trimmed.startsWith("/")) {
      onCommand(trimmed);
    } else {
      onSend(trimmed);
    }
    setValue("");
    setShowCommands(false);
  }, [value, onSend, onCommand]);

  const handleKeyDown = useCallback((e: React.KeyboardEvent) => {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSubmit();
    }
    if (e.key === "/" && value === "") {
      setShowCommands(true);
    }
  }, [handleSubmit, value]);

  const handleCommandSelect = useCallback((cmd: string) => {
    setValue(cmd + " ");
    setShowCommands(false);
    textareaRef.current?.focus();
  }, []);

  return (
    <div className="border-t border-gray-700 p-3">
      <div className="relative">
        <textarea
          ref={textareaRef}
          value={value}
          onChange={(e) => {
            setValue(e.target.value);
            setShowCommands(e.target.value.startsWith("/") && !e.target.value.includes(" "));
          }}
          onKeyDown={handleKeyDown}
          disabled={isStreaming}
          placeholder={isStreaming ? "推理中..." : "请输入你的问题... (Enter 发送, Shift+Enter 换行)"}
          className="w-full bg-gray-800 text-gray-200 rounded-lg px-4 py-3 resize-none focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
          rows={3}
        />
        {showCommands && (
          <div className="absolute bottom-full left-0 mb-1 bg-gray-800 border border-gray-600 rounded-lg shadow-lg overflow-hidden z-10">
            {COMMANDS.map((c) => (
              <button
                key={c.cmd}
                onClick={() => handleCommandSelect(c.cmd)}
                className="block w-full text-left px-4 py-2 text-sm text-gray-300 hover:bg-gray-700"
              >
                <span className="text-yellow-400">{c.cmd}</span> - {c.desc}
              </button>
            ))}
          </div>
        )}
      </div>
      <div className="flex justify-between mt-1 text-xs text-gray-500">
        <span>Enter 发送 │ Shift+Enter 换行 │ / 命令</span>
        <span>{value.length} 字符</span>
      </div>
    </div>
  );
}
