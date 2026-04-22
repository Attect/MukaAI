import React, { useState, useEffect, useRef } from "react";
import type { ToolCall } from "../types";
import GitDiffPanel, { isGitDiffResult } from "./GitDiffPanel";
import DiagnosticsPanel, { isDiagnosticsResult } from "./DiagnosticsPanel";
import type { DiagnosticsPanelProps } from "./DiagnosticsPanel";
import { useUI } from "../contexts/UIContext";

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

/** XML 语法高亮组件 - 识别标签名、属性名和值 */
function XmlHighlight({ xmlString }: { xmlString: string }): React.ReactElement {
  // 按行分割处理
  const lines = xmlString.split("\n");

  return (
    <pre className="text-xs p-2 rounded overflow-auto max-h-48 border xml-output" style={{ fontFamily: "'Consolas', 'Monaco', monospace", whiteSpace: "pre-wrap" }}>
      {lines.map((line, lineIdx) => {
        // 跳过空行
        if (line.trim() === "") return <div key={lineIdx} />;

        const result: React.ReactNode[] = [];
        let i = 0;
        let textBuffer = "";

        const flushText = () => {
          if (textBuffer) {
            result.push(<span key={`t-${lineIdx}-${i}`}>{textBuffer}</span>);
            textBuffer = "";
          }
        };

        while (i < line.length) {
          const ch = line[i];

          // 检测标签开始 < 或 </
          if (ch === "<") {
            flushText();
            let j = i + 1;
            let tagContent = "";
            let closingSlash = false;

            if (j < line.length && line[j] === "/") {
              closingSlash = true;
              j++;
            }

            // 读取标签名
            while (j < line.length && (line[j] !== " " && line[j] !== ">" && line[j] !== "/")) {
              tagContent += line[j];
              j++;
            }

            if (tagContent) {
              result.push(<span key={`tag-${lineIdx}-${i}`} style={{ color: "#569cd6" }}>&lt;{closingSlash ? "/" : ""}{tagContent}</span>);
            }

            // 读取属性部分
            while (j < line.length && line[j] !== ">") {
              if (line[j] === " ") {
                flushText();
                result.push(<span key={`sp-${lineIdx}-${i}`}>{" "}</span>);
                j++;
                continue;
              }

              // 读取属性名
              let attrName = "";
              while (j < line.length && line[j] !== "=" && line[j] !== " " && line[j] !== ">") {
                attrName += line[j];
                j++;
              }

              if (attrName) {
                flushText();
                result.push(<span key={`attr-${lineIdx}-${i}`} style={{ color: "#9cdcfe" }}>{attrName}</span>);

                // 跳过 =
                if (j < line.length && line[j] === "=") {
                  result.push(<span key={`eq-${lineIdx}-${i}`}>{"="}</span>);
                  j++;
                }

                // 读取属性值
                if (j < line.length && (line[j] === '"' || line[j] === "'")) {
                  const quote = line[j];
                  let valStart = j + 1;
                  while (j < line.length && line[j] !== quote) j++;
                  const attrValue = line.substring(valStart, j);
                  result.push(<span key={`val-${lineIdx}-${i}`} style={{ color: "#ce9178" }}>{quote}{attrValue}{quote}</span>);
                  j++; // 跳过结束引号
                }
              } else {
                textBuffer += line[j];
                j++;
              }
            }

            // 处理 /> 或 >
            if (j < line.length) {
              if (line[j] === "/" && j + 1 < line.length && line[j + 1] === ">") {
                flushText();
                result.push(<span key={`self-${lineIdx}-${i}`} style={{ color: "#569cd6" }}>/&gt;</span>);
                j += 2;
              } else if (line[j] === ">") {
                flushText();
                result.push(<span key={`end-${lineIdx}-${i}`} style={{ color: "#569cd6" }}>&gt;</span>);
                j++;
              }
            }

            i = j;
          } else {
            textBuffer += ch;
            i++;
          }
        }

        flushText();
        return <div key={lineIdx}>{result}</div>;
      })}
    </pre>
  );
}

/** 将JSON对象转为XML字符串 */
function jsonToXml(obj: unknown, rootTag: string = "result", indentLevel: number = 0): string {
  const indent = "  ".repeat(indentLevel);

  if (obj === null || obj === undefined) {
    return `${indent}<${rootTag}/>`;
  }

  if (typeof obj === "string") {
    const escaped = escapeXml(obj);
    return `${indent}<${rootTag}><![CDATA[${escaped}]]></${rootTag}>`;
  }

  if (typeof obj === "number" || typeof obj === "boolean") {
    return `${indent}<${rootTag}><![CDATA[${String(obj)}]]></${rootTag}>`;
  }

  if (Array.isArray(obj)) {
    let xml = `${indent}<${rootTag}>\n`;
    for (const item of obj) {
      xml += jsonToXml(item, "item", indentLevel + 1);
      xml += "\n";
    }
    xml += `${indent}</${rootTag}>`;
    return xml;
  }

  if (typeof obj === "object") {
    const entries = Object.entries(obj as Record<string, unknown>);
    if (entries.length === 0) {
      return `${indent}<${rootTag}/>`;
    }

    let xml = `${indent}<${rootTag}>\n`;
    for (const [key, value] of entries) {
      const safeKey = key.replace(/[^a-zA-Z0-9_-]/g, "_");
      xml += jsonToXml(value, safeKey, indentLevel + 1);
      xml += "\n";
    }
    xml += `${indent}</${rootTag}>`;
    return xml;
  }

  // fallback
  return `${indent}<${rootTag}><![CDATA[${escapeXml(String(obj))}]]></${rootTag}>`;
}

/** 转义XML特殊字符 */
function escapeXml(str: string): string {
  return str
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&apos;");
}

/** 尝试将字符串解析为JSON并转为XML */
function tryJsonToXml(result: string): string | null {
  try {
    const parsed = JSON.parse(result);
    // 只有当解析结果是对象或数组时才转为XML
    if (typeof parsed === "object" && parsed !== null) {
      return jsonToXml(parsed, "result", 0);
    }
    return null;
  } catch {
    return null;
  }
}

export default function ToolCallBlock({ toolCall, isStreaming }: ToolCallBlockProps): React.ReactElement {
  const { settings } = useUI();
  const [expanded, setExpanded] = useState(settings.defaultExpandToolCalls);
  const userToggledRef = useRef(false);

  // 当设置变化且用户未手动交互时，更新默认展开状态
  useEffect(() => {
    if (!userToggledRef.current) {
      setExpanded(settings.defaultExpandToolCalls);
    }
  }, [settings.defaultExpandToolCalls]);

  const handleToggle = () => {
    userToggledRef.current = true;
    setExpanded((prev) => !prev);
  };

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

  // 尝试将JSON结果转为XML展示（仅在isComplete=true时）
  let xmlOutput: string | null = null;
  const showXmlOutput = toolCall.isComplete && !toolCall.resultError && toolCall.result && !showDiffView && !showDiagnostics;
  if (showXmlOutput) {
    xmlOutput = tryJsonToXml(toolCall.result);
  }

  return (
    <div className="my-2 border border-amber-600/70 rounded bg-amber-950/60">
      <button
        onClick={handleToggle}
        className="flex items-center gap-2 w-full px-3 py-2 text-left text-amber-300 font-medium hover:bg-amber-950/40"
      >
        <span>{expanded ? "▼" : "▶"}</span>
        <span>🔧 {toolCall.name}</span>
        {isStreaming && !toolCall.isComplete && <span className="animate-pulse text-amber-200">▌</span>}
        {toolCall.isComplete && <span className="text-green-400 text-xs">✓</span>}
      </button>
      {expanded && (
        <div className="px-3 pb-2">
          <div className="text-amber-200/80 text-xs font-medium mb-1">Parameters:</div>
          <pre className="text-amber-100 text-xs bg-gray-900/70 p-2 rounded overflow-auto border border-gray-700/50 whitespace-pre-wrap">
            {formatArguments(toolCall.arguments)}
          </pre>
          {(toolCall.result || toolCall.resultError) && (
            <>
              <div className="text-amber-200/80 text-xs font-medium mt-2 mb-1">Result:</div>
              {showDiffView && diffData ? (
                <GitDiffPanel
                  files={diffData.files}
                  summary={diffData.summary}
                />
              ) : showDiagnostics && diagnosticsProps ? (
                <DiagnosticsPanel {...diagnosticsProps} />
              ) : xmlOutput ? (
                <XmlHighlight xmlString={xmlOutput} />
              ) : (
                <pre className={`text-xs p-2 rounded overflow-auto max-h-48 border whitespace-pre-wrap ${toolCall.resultError ? "text-red-300 bg-red-950/50 border-red-800/50" : "text-green-300 bg-green-950/50 border-green-800/50"}`}>
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
