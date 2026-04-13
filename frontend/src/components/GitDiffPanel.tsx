import React, { useState, useMemo } from "react";

/** 单个Diff文件块的数据（从git_diff工具返回） */
interface DiffFileData {
  path: string;
  status: string;
  additions: number;
  deletions: number;
  diff: string;
}

/** Diff行类型 */
type DiffLineType = "header" | "hunk" | "add" | "delete" | "context" | "no-newline";

/** 解析后的Diff行 */
interface ParsedDiffLine {
  type: DiffLineType;
  content: string;
  oldLineNo?: number;
  newLineNo?: number;
}

/** 解析后的文件Diff */
interface ParsedFileDiff {
  path: string;
  status: string;
  additions: number;
  deletions: number;
  lines: ParsedDiffLine[];
}

/** 解析unified diff文本为结构化行 */
function parseDiffLines(diffText: string): ParsedDiffLine[] {
  const lines = diffText.split("\n");
  const result: ParsedDiffLine[] = [];
  let oldLineNo = 0;
  let newLineNo = 0;

  for (const line of lines) {
    if (line.startsWith("diff --git")) {
      result.push({ type: "header", content: line });
    } else if (line.startsWith("index ") || line.startsWith("--- ") || line.startsWith("+++ ")) {
      result.push({ type: "header", content: line });
    } else if (line.startsWith("@@")) {
      // 解析 @@ -oldStart,oldCount +newStart,newCount @@ 格式
      const match = line.match(/@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/);
      if (match) {
        oldLineNo = parseInt(match[1], 10);
        newLineNo = parseInt(match[2], 10);
      }
      result.push({ type: "hunk", content: line });
    } else if (line.startsWith("+")) {
      result.push({ type: "add", content: line.substring(1), newLineNo: newLineNo++ });
    } else if (line.startsWith("-")) {
      result.push({ type: "delete", content: line.substring(1), oldLineNo: oldLineNo++ });
    } else if (line.startsWith(" ")) {
      result.push({ type: "context", content: line.substring(1), oldLineNo: oldLineNo++, newLineNo: newLineNo++ });
    } else if (line === "\\ No newline at end of file") {
      result.push({ type: "no-newline", content: line });
    } else if (line.trim() !== "") {
      // 其他非空行（可能是旧的diff行格式）
      result.push({ type: "context", content: line, oldLineNo: oldLineNo++, newLineNo: newLineNo++ });
    }
  }

  return result;
}

/** 按文件拆分diff文本 */
function splitDiffByFile(diffText: string): string[] {
  if (!diffText.trim()) return [];
  const parts = diffText.split(/(?=^diff --git )/m);
  return parts.filter((p) => p.trim().length > 0);
}

/** 从diff头部提取文件路径 */
function extractFilePath(diffPart: string): string {
  const match = diffPart.match(/^diff --git a\/(.+?) b\/(.+?)$/m);
  if (match) return match[2];
  // 备选：从 --- a/path 行提取
  const pathMatch = diffPart.match(/^\+\+\+ b\/(.+)$/m);
  if (pathMatch) return pathMatch[1];
  return "unknown";
}

/** 从diff内容检测文件状态 */
function detectStatus(diffPart: string): string {
  if (diffPart.includes("new file mode")) return "added";
  if (diffPart.includes("deleted file mode")) return "deleted";
  if (diffPart.includes("rename from")) return "renamed";
  return "modified";
}

/** 状态标签的颜色映射 */
function statusColor(status: string): string {
  switch (status) {
    case "added":
      return "text-green-400";
    case "deleted":
      return "text-red-400";
    case "renamed":
      return "text-blue-400";
    default:
      return "text-yellow-400";
  }
}

/** 状态标签文字 */
function statusLabel(status: string): string {
  switch (status) {
    case "added":
      return "A";
    case "deleted":
      return "D";
    case "renamed":
      return "R";
    default:
      return "M";
  }
}

/** 单文件Diff渲染 */
function FileDiffView({ fileDiff, defaultExpanded }: { fileDiff: ParsedFileDiff; defaultExpanded: boolean }) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  return (
    <div className="border border-gray-700 rounded mb-2">
      {/* 文件头部 */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-2 w-full px-3 py-1.5 text-left text-sm hover:bg-gray-800/50"
      >
        <span className="text-gray-500 text-xs">{expanded ? "▼" : "▶"}</span>
        <span className={`font-mono text-xs font-bold px-1.5 py-0.5 rounded ${statusColor(fileDiff.status)}`}>
          {statusLabel(fileDiff.status)}
        </span>
        <span className="font-mono text-gray-300 text-xs flex-1">{fileDiff.path}</span>
        <span className="text-green-400 text-xs">+{fileDiff.additions}</span>
        <span className="text-red-400 text-xs">-{fileDiff.deletions}</span>
      </button>

      {/* Diff内容 */}
      {expanded && (
        <div className="font-mono text-xs overflow-x-auto border-t border-gray-700">
          {fileDiff.lines.map((line, idx) => (
            <DiffLineView key={idx} line={line} />
          ))}
        </div>
      )}
    </div>
  );
}

/** 单行Diff渲染 */
function DiffLineView({ line }: { line: ParsedDiffLine }) {
  switch (line.type) {
    case "header":
      return (
        <div className="px-3 py-0.5 bg-gray-800/60 text-gray-500 select-none overflow-hidden">
          {line.content}
        </div>
      );
    case "hunk":
      return (
        <div className="px-3 py-0.5 bg-blue-900/30 text-blue-300 select-none">
          {line.content}
        </div>
      );
    case "add":
      return (
        <div className="flex bg-green-900/30 hover:bg-green-900/40">
          <span className="w-12 text-right pr-2 text-gray-600 select-none border-r border-gray-700 min-w-[3rem]">
            {line.newLineNo ?? ""}
          </span>
          <span className="text-green-600 select-none w-5 text-center">+</span>
          <span className="text-green-300 pl-1 whitespace-pre">{line.content}</span>
        </div>
      );
    case "delete":
      return (
        <div className="flex bg-red-900/30 hover:bg-red-900/40">
          <span className="w-12 text-right pr-2 text-gray-600 select-none border-r border-gray-700 min-w-[3rem]">
            {line.oldLineNo ?? ""}
          </span>
          <span className="text-red-600 select-none w-5 text-center">-</span>
          <span className="text-red-300 pl-1 whitespace-pre">{line.content}</span>
        </div>
      );
    case "context":
      return (
        <div className="flex hover:bg-gray-800/30">
          <span className="w-12 text-right pr-2 text-gray-600 select-none border-r border-gray-700 min-w-[3rem]">
            {line.oldLineNo ?? ""}
          </span>
          <span className="w-12 text-right pr-2 text-gray-600 select-none border-r border-gray-700 min-w-[3rem]">
            {line.newLineNo ?? ""}
          </span>
          <span className="text-gray-400 pl-1 whitespace-pre">{line.content}</span>
        </div>
      );
    case "no-newline":
      return (
        <div className="px-3 py-0.5 text-gray-600 italic select-none">
          {line.content}
        </div>
      );
    default:
      return null;
  }
}

/** GitDiffPanel 属性 */
interface GitDiffPanelProps {
  /** diff文本内容（unified diff格式） */
  diffText?: string;
  /** 或者结构化的文件列表（来自git_diff工具返回） */
  files?: DiffFileData[];
  /** 摘要信息 */
  summary?: string;
}

/**
 * GitDiffPanel - Git diff可视化展示组件
 *
 * 将unified diff格式渲染为可视化展示：
 * - 新增行绿色背景
 * - 删除行红色背景
 * - 行号正确显示
 * - 多文件diff可折叠切换
 */
export default function GitDiffPanel({ diffText, files, summary }: GitDiffPanelProps): React.ReactElement | null {
  // 解析输入数据为统一的文件Diff列表
  const parsedFiles = useMemo<ParsedFileDiff[]>(() => {
    // 优先使用结构化的files数据
    if (files && files.length > 0) {
      return files.map((f) => ({
        path: f.path,
        status: f.status,
        additions: f.additions,
        deletions: f.deletions,
        lines: parseDiffLines(f.diff),
      }));
    }

    // 否则从纯文本解析
    if (diffText) {
      const parts = splitDiffByFile(diffText);
      return parts.map((part) => ({
        path: extractFilePath(part),
        status: detectStatus(part),
        additions: (part.match(/^\+/gm) || []).length - (part.match(/^\+\+\+/gm) || []).length,
        deletions: (part.match(/^-/gm) || []).length - (part.match(/^---/gm) || []).length,
        lines: parseDiffLines(part),
      }));
    }

    return [];
  }, [diffText, files]);

  if (parsedFiles.length === 0) {
    return (
      <div className="text-gray-500 text-xs p-3 italic">
        No changes to display
      </div>
    );
  }

  return (
    <div className="my-1">
      {/* 摘要栏 */}
      {summary && (
        <div className="text-gray-400 text-xs mb-2 px-1">
          {summary}
        </div>
      )}

      {/* 文件列表 */}
      {parsedFiles.map((file, idx) => (
        <FileDiffView key={idx} fileDiff={file} defaultExpanded={parsedFiles.length <= 3} />
      ))}
    </div>
  );
}

/**
 * 检测工具调用结果是否为git_diff的输出
 * 用于在ToolCallBlock中判断是否使用GitDiffPanel渲染
 */
export function isGitDiffResult(toolName: string, result: string): boolean {
  if (toolName !== "git_diff") return false;
  try {
    const parsed = JSON.parse(result);
    return parsed.success === true && parsed.data && parsed.data.files;
  } catch {
    return false;
  }
}
