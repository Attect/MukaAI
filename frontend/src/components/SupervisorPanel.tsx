import React, { useState, useCallback } from "react";
import type { SupervisorResult, SupervisorLevel } from "../types";

interface SupervisorPanelProps {
  results: SupervisorResult[];
}

/** 不同干预级别对应的样式 */
const LEVEL_STYLES: Record<SupervisorLevel, { bg: string; border: string; text: string; icon: string; label: string }> = {
  info: {
    bg: "var(--supervisor-info-bg, rgba(59,130,246,0.1))",
    border: "var(--supervisor-info-border, rgba(59,130,246,0.4))",
    text: "var(--supervisor-info-text, #3b82f6)",
    icon: "ℹ️",
    label: "信息",
  },
  warning: {
    bg: "var(--supervisor-warning-bg, rgba(245,158,11,0.1))",
    border: "var(--supervisor-warning-border, rgba(245,158,11,0.4))",
    text: "var(--supervisor-warning-text, #f59e0b)",
    icon: "⚠️",
    label: "警告",
  },
  correction: {
    bg: "var(--supervisor-correction-bg, rgba(239,68,68,0.1))",
    border: "var(--supervisor-correction-border, rgba(239,68,68,0.4))",
    text: "var(--supervisor-correction-text, #ef4444)",
    icon: "🔧",
    label: "修正",
  },
  halt: {
    bg: "var(--supervisor-halt-bg, rgba(220,38,38,0.15))",
    border: "var(--supervisor-halt-border, rgba(220,38,38,0.6))",
    text: "var(--supervisor-halt-text, #dc2626)",
    icon: "🛑",
    label: "中止",
  },
};

/** 单条监督结果项 */
function SupervisorItem({ result }: { result: SupervisorResult }): React.ReactElement {
  const [expanded, setExpanded] = useState(false);
  const style = LEVEL_STYLES[result.level] || LEVEL_STYLES.info;

  const toggleExpand = useCallback(() => {
    setExpanded((prev) => !prev);
  }, []);

  return (
    <div
      style={{
        background: style.bg,
        border: `1px solid ${style.border}`,
        borderLeft: `4px solid ${style.text}`,
        borderRadius: "0.375rem",
        padding: "0.5rem 0.75rem",
        marginBottom: "0.375rem",
        fontSize: "0.8125rem",
      }}
    >
      {/* 横幅头部 - 摘要信息 */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: "0.5rem",
          cursor: result.details ? "pointer" : "default",
          userSelect: "none",
        }}
        onClick={result.details ? toggleExpand : undefined}
      >
        <span style={{ flexShrink: 0 }}>{style.icon}</span>
        <span
          style={{
            fontWeight: 700,
            color: style.text,
            flexShrink: 0,
            fontSize: "0.75rem",
            textTransform: "uppercase",
          }}
        >
          [{style.label}]
        </span>
        <span style={{ color: "var(--text-secondary)", flex: 1 }}>{result.summary}</span>
        {result.details && (
          <span
            style={{
              flexShrink: 0,
              color: "var(--text-dim)",
              fontSize: "0.75rem",
              transform: expanded ? "rotate(90deg)" : "rotate(0deg)",
              transition: "transform 0.2s",
            }}
          >
            ▶
          </span>
        )}
      </div>

      {/* 展开的详细信息 */}
      {expanded && result.details && (
        <div
          style={{
            marginTop: "0.5rem",
            paddingLeft: "1.5rem",
            color: "var(--text-secondary)",
            whiteSpace: "pre-wrap",
            borderTop: `1px solid ${style.border}`,
            paddingTop: "0.5rem",
            fontSize: "0.75rem",
          }}
        >
          {result.details}
        </div>
      )}
    </div>
  );
}

/** Supervisor监督结果面板 */
export default function SupervisorPanel({ results }: SupervisorPanelProps): React.ReactElement {
  if (results.length === 0) return <></>;

  return (
    <div style={{ padding: "0 1rem 0.25rem" }}>
      {results.map((result, index) => (
        <SupervisorItem key={`${result.timestamp || index}-${result.checkType}`} result={result} />
      ))}
    </div>
  );
}
