import React, { useState, useMemo } from "react";

/** 单条诊断项 */
export interface DiagnosticItem {
  severity: string;
  line: number;
  column: number;
  message: string;
  source?: string;
  code?: string;
}

/** 诊断摘要 */
export interface DiagnosticSummary {
  total: number;
  errors: number;
  warnings: number;
  info?: number;
  hints?: number;
}

/** 单文件诊断结果 */
export interface FileDiagnosticResult {
  file: string;
  language: string;
  diagnostics: DiagnosticItem[];
  summary: DiagnosticSummary;
  degraded?: boolean;
  degrade_reason?: string;
}

/** DiagnosticsPanel 属性 */
export interface DiagnosticsPanelProps {
  file?: string;
  language?: string;
  diagnostics: DiagnosticItem[];
  summary: DiagnosticSummary;
  degraded?: boolean;
  degradeReason?: string;
  /** get_diagnostics 多文件模式 */
  files?: FileDiagnosticResult[];
  totalSummary?: {
    total_files: number;
    total_errors: number;
    total_warnings: number;
  };
}

/** severity 排序权重（error > warning > info > hint） */
function severityOrder(severity: string): number {
  switch (severity) {
    case "error":
      return 0;
    case "warning":
      return 1;
    case "info":
      return 2;
    case "hint":
      return 3;
    default:
      return 4;
  }
}

/** severity 对应的图标 */
function severityIcon(severity: string): string {
  switch (severity) {
    case "error":
      return "❌";
    case "warning":
      return "⚠️";
    case "info":
      return "ℹ️";
    case "hint":
      return "💡";
    default:
      return "•";
  }
}

/** severity 对应的文字颜色 */
function severityTextColor(severity: string): string {
  switch (severity) {
    case "error":
      return "text-red-400";
    case "warning":
      return "text-yellow-400";
    case "info":
      return "text-blue-400";
    case "hint":
      return "text-gray-400";
    default:
      return "text-gray-300";
  }
}

/** severity 对应的背景颜色 */
function severityBgColor(severity: string): string {
  switch (severity) {
    case "error":
      return "bg-red-900/20 hover:bg-red-900/30";
    case "warning":
      return "bg-yellow-900/20 hover:bg-yellow-900/30";
    case "info":
      return "bg-blue-900/20 hover:bg-blue-900/30";
    case "hint":
      return "bg-gray-800/30 hover:bg-gray-800/40";
    default:
      return "bg-gray-800/20 hover:bg-gray-800/30";
  }
}

/** 诊断摘要栏 */
function SummaryBar({ summary }: { summary: DiagnosticSummary }) {
  return (
    <div className="flex items-center gap-3 text-xs">
      {summary.errors > 0 && (
        <span className="flex items-center gap-1 text-red-400">
          ❌ {summary.errors} 错误
        </span>
      )}
      {summary.warnings > 0 && (
        <span className="flex items-center gap-1 text-yellow-400">
          ⚠️ {summary.warnings} 警告
        </span>
      )}
      {(summary.info ?? 0) > 0 && (
        <span className="flex items-center gap-1 text-blue-400">
          ℹ️ {summary.info} 信息
        </span>
      )}
      {(summary.hints ?? 0) > 0 && (
        <span className="flex items-center gap-1 text-gray-400">
          💡 {summary.hints} 提示
        </span>
      )}
      {summary.total === 0 && (
        <span className="text-green-400">✅ 无诊断问题</span>
      )}
    </div>
  );
}

/** 降级模式提示 */
function DegradedNotice({ reason }: { reason?: string }) {
  return (
    <div className="flex items-center gap-2 px-3 py-1.5 mb-2 rounded bg-yellow-900/30 border border-yellow-700/40 text-xs text-yellow-300">
      <span>⚠️</span>
      <span>降级模式：{reason || "LSP功能未启用"}</span>
    </div>
  );
}

/** 单条诊断项渲染 */
function DiagnosticItemView({ item }: { item: DiagnosticItem }) {
  return (
    <div className={`flex items-start gap-2 px-3 py-1.5 ${severityBgColor(item.severity)}`}>
      <span className="shrink-0 text-sm leading-5">{severityIcon(item.severity)}</span>
      <span className={`font-mono text-xs shrink-0 leading-5 ${severityTextColor(item.severity)}`}>
        {item.line}:{item.column}
      </span>
      <span className={`text-xs leading-5 flex-1 ${severityTextColor(item.severity)}`}>
        {item.message}
      </span>
      {item.source && (
        <span className="text-gray-500 text-xs shrink-0 leading-5">
          [{item.source}{item.code ? ` ${item.code}` : ""}]
        </span>
      )}
    </div>
  );
}

/** 单文件诊断列表 */
function FileDiagnosticsView({ result, defaultExpanded }: { result: FileDiagnosticResult; defaultExpanded: boolean }) {
  const [expanded, setExpanded] = useState(defaultExpanded);

  const sortedDiagnostics = useMemo(
    () => [...result.diagnostics].sort((a, b) => severityOrder(a.severity) - severityOrder(b.severity)),
    [result.diagnostics]
  );

  return (
    <div className="border border-gray-700 rounded mb-2">
      {/* 文件头部 */}
      <button
        onClick={() => setExpanded(!expanded)}
        className="flex items-center gap-2 w-full px-3 py-1.5 text-left text-sm hover:bg-gray-800/50"
      >
        <span className="text-gray-500 text-xs">{expanded ? "▼" : "▶"}</span>
        <span className="font-mono text-gray-300 text-xs flex-1 truncate" title={result.file}>
          {result.file}
        </span>
        {result.language && (
          <span className="text-gray-500 text-xs border border-gray-600 px-1 rounded">
            {result.language}
          </span>
        )}
        {result.summary.errors > 0 && (
          <span className="text-red-400 text-xs">{result.summary.errors}E</span>
        )}
        {result.summary.warnings > 0 && (
          <span className="text-yellow-400 text-xs">{result.summary.warnings}W</span>
        )}
        {result.summary.total === 0 && (
          <span className="text-green-400 text-xs">✓</span>
        )}
      </button>

      {/* 诊断内容 */}
      {expanded && (
        <div className="border-t border-gray-700">
          {result.degraded && <DegradedNotice reason={result.degrade_reason} />}
          {sortedDiagnostics.length > 0 ? (
            sortedDiagnostics.map((diag, idx) => <DiagnosticItemView key={idx} item={diag} />)
          ) : (
            <div className="px-3 py-2 text-gray-500 text-xs italic">
              {result.degraded ? "无诊断数据（降级模式）" : "无诊断问题"}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

/**
 * DiagnosticsPanel - LSP诊断结果可视化展示组件
 *
 * 支持：
 * - 单文件诊断（diagnose_code）渲染
 * - 多文件批量诊断（get_diagnostics）分组渲染
 * - 降级模式提示
 * - 诊断项按severity排序
 * - 可折叠/展开
 */
export default function DiagnosticsPanel(props: DiagnosticsPanelProps): React.ReactElement {
  // 判断是否为多文件模式
  const isMultiFile = Boolean(props.files && props.files.length > 0);

  if (isMultiFile && props.files) {
    // 多文件模式
    return (
      <div className="my-1">
        {/* 全局摘要 */}
        {props.totalSummary && (
          <div className="flex items-center gap-3 px-3 py-2 mb-2 bg-gray-800/40 rounded text-xs">
            <span className="text-gray-400">
              {props.totalSummary.total_files} 个文件
            </span>
            {props.totalSummary.total_errors > 0 && (
              <span className="text-red-400">
                ❌ {props.totalSummary.total_errors} 错误
              </span>
            )}
            {props.totalSummary.total_warnings > 0 && (
              <span className="text-yellow-400">
                ⚠️ {props.totalSummary.total_warnings} 警告
              </span>
            )}
            {props.totalSummary.total_errors === 0 && props.totalSummary.total_warnings === 0 && (
              <span className="text-green-400">✅ 全部通过</span>
            )}
          </div>
        )}

        {/* 文件列表 */}
        {props.files.map((fileResult, idx) => (
          <FileDiagnosticsView
            key={idx}
            result={fileResult}
            defaultExpanded={props.files!.length <= 5}
          />
        ))}
      </div>
    );
  }

  // 单文件模式
  const sortedDiagnostics = useMemo(
    () => [...props.diagnostics].sort((a, b) => severityOrder(a.severity) - severityOrder(b.severity)),
    [props.diagnostics]
  );

  return (
    <div className="my-1">
      {/* 降级提示 */}
      {props.degraded && <DegradedNotice reason={props.degradeReason} />}

      {/* 文件信息 + 摘要 */}
      <div className="flex items-center gap-2 px-3 py-2 mb-2 bg-gray-800/40 rounded">
        {props.file && (
          <span className="font-mono text-gray-300 text-xs truncate" title={props.file}>
            {props.file}
          </span>
        )}
        {props.language && (
          <span className="text-gray-500 text-xs border border-gray-600 px-1 rounded">
            {props.language}
          </span>
        )}
        <div className="flex-1" />
        <SummaryBar summary={props.summary} />
      </div>

      {/* 诊断列表 */}
      {sortedDiagnostics.length > 0 ? (
        <div className="border border-gray-700 rounded">
          {sortedDiagnostics.map((diag, idx) => (
            <DiagnosticItemView key={idx} item={diag} />
          ))}
        </div>
      ) : (
        <div className="px-3 py-2 text-gray-500 text-xs italic border border-gray-700 rounded">
          {props.degraded ? "无诊断数据（降级模式）" : "无诊断问题"}
        </div>
      )}
    </div>
  );
}

/**
 * 检测工具调用结果是否为LSP诊断输出
 * 用于在ToolCallBlock中判断是否使用DiagnosticsPanel渲染
 */
export function isDiagnosticsResult(toolName: string, result: string): boolean {
  if (toolName !== "diagnose_code" && toolName !== "get_diagnostics") return false;
  try {
    const parsed = JSON.parse(result);
    return parsed.success === true && parsed.data !== undefined;
  } catch {
    return false;
  }
}
