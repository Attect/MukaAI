import React, { useState } from "react";
import type { ToolCall } from "../types";
import GitDiffPanel, { isGitDiffResult } from "./GitDiffPanel";
import DiagnosticsPanel, { isDiagnosticsResult } from "./DiagnosticsPanel";
import type { DiagnosticsPanelProps } from "./DiagnosticsPanel";

interface ToolCallBlockProps {
  toolCall: ToolCall;
  isStreaming: boolean;
}

/** 从 diagnose_code 工具结果中提取DiagnosticsPanel所需的props */
function parseSingleDiagnostics(data: Record<string, unknown>): DiagnosticsPanelProps {
  return {
    file: data.file as string | undefined,
    language: data.language as string | undefined,
    diagnostics: (data.diagnostics as DiagnosticsPanelProps["diagnostics"]) ?? [],
    summary: (data.summary as DiagnosticsPanelProps["summary"]) ?? { total: 0, errors: 0, warnings: 0 },
    degraded: data.degraded as boolean | undefined,
    degradeReason: data.degrade_reason as string | undefined,
  };
}

/** 从 get_diagnostics 工具结果中提取DiagnosticsPanel所需的props */
function parseMultiDiagnostics(data: Record<string, unknown>): DiagnosticsPanelProps {
  return {
    diagnostics: [],
    summary: { total: 0, errors: 0, warnings: 0 },
    files: (data.files as DiagnosticsPanelProps["files"]) ?? [],
    totalSummary: data.summary as DiagnosticsPanelProps["totalSummary"] | undefined,
  };
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

  // 检测是否为git_diff结果，使用专用的diff渲染
  const showDiffView = toolCall.isComplete && toolCall.result && isGitDiffResult(toolCall.name, toolCall.result);

  // 检测是否为LSP诊断结果，使用专用的诊断渲染
  const showDiagnostics = !showDiffView && toolCall.isComplete && toolCall.result && isDiagnosticsResult(toolCall.name, toolCall.result);

  // 解析git_diff结果用于diff视图
  let diffData: { files?: Array<{ path: string; status: string; additions: number; deletions: number; diff: string }>; summary?: string } | undefined;
  if (showDiffView && toolCall.result) {
    try {
      const parsed = JSON.parse(toolCall.result);
      diffData = parsed.data;
    } catch {
      diffData = undefined;
    }
  }

  // 解析诊断结果为DiagnosticsPanel props
  let diagnosticsProps: DiagnosticsPanelProps | undefined;
  if (showDiagnostics && toolCall.result) {
    try {
      const parsed = JSON.parse(toolCall.result);
      const data = parsed.data as Record<string, unknown>;
      // 区分单文件(diagnose_code)和多文件(get_diagnostics)
      if (data.files && Array.isArray(data.files)) {
        diagnosticsProps = parseMultiDiagnostics(data);
      } else {
        diagnosticsProps = parseSingleDiagnostics(data);
      }
    } catch {
      diagnosticsProps = undefined;
    }
  }

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
              {showDiffView && diffData ? (
                <GitDiffPanel
                  files={diffData.files}
                  summary={diffData.summary}
                />
              ) : showDiagnostics && diagnosticsProps ? (
                <DiagnosticsPanel {...diagnosticsProps} />
              ) : (
                <pre className={`text-xs p-2 rounded overflow-x-auto max-h-48 overflow-y-auto ${toolCall.resultError ? "text-red-400 bg-red-900/20" : "text-green-400 bg-green-900/20"}`}>
                  {toolCall.resultError || toolCall.result}
                </pre>
              )}
            </>
          )}
        </div>
      )}
    </div>
  );
}
