import React, { useState, useCallback } from "react";
import type { Conversation } from "../types";

interface SidebarProps {
  visible: boolean;
  conversations: Conversation[];
  activeId: string;
  onSelect: (id: string) => void;
  onClose: () => void;
  onNew: () => void;
  onDelete: (id: string) => void;
  onGenerateTitle?: (id: string) => void;
  onUpdateTitle: (id: string, title: string) => void;
  generatingIds: Set<string>;
}

export default function Sidebar({
  visible,
  conversations,
  activeId,
  onSelect,
  onClose,
  onNew,
  onDelete,
  onGenerateTitle,
  onUpdateTitle,
  generatingIds,
}: SidebarProps): React.ReactElement {
  const [deleteTarget, setDeleteTarget] = useState<string | null>(null);
  const [editTarget, setEditTarget] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState("");
  const [hoveredId, setHoveredId] = useState<string | null>(null);

  const handleConfirmDelete = useCallback(() => {
    if (deleteTarget) {
      onDelete(deleteTarget);
      setDeleteTarget(null);
    }
  }, [deleteTarget, onDelete]);

  const handleStartEdit = useCallback((id: string, title: string) => {
    setEditTarget(id);
    setEditTitle(title);
  }, []);

  const handleConfirmEdit = useCallback(() => {
    if (editTarget && editTitle.trim()) {
      onUpdateTitle(editTarget, editTitle.trim());
    }
    setEditTarget(null);
    setEditTitle("");
  }, [editTarget, editTitle, onUpdateTitle]);

  const handleCancelEdit = useCallback(() => {
    setEditTarget(null);
    setEditTitle("");
  }, []);

  const handleEditKeyDown = useCallback(
    (e: React.KeyboardEvent) => {
      if (e.key === "Enter") {
        e.preventDefault();
        handleConfirmEdit();
      } else if (e.key === "Escape") {
        handleCancelEdit();
      }
    },
    [handleConfirmEdit, handleCancelEdit],
  );

  if (!visible) return <></>;

  return (
    <>
      <div
        role="navigation"
        aria-label="对话列表侧边栏"
        style={{
          width: "16rem",
          background: "var(--bg-sidebar)",
          borderRight: "1px solid var(--border-color)",
          display: "flex",
          flexDirection: "column",
          flexShrink: 0,
        }}
      >
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
            padding: "0.75rem 1rem",
            borderBottom: "1px solid var(--border-color)",
          }}
        >
          <span
            style={{ color: "var(--text-secondary)", fontWeight: 700, fontSize: "0.875rem" }}
          >
            对话列表
          </span>
          <button
            onClick={onClose}
            aria-label="关闭侧边栏"
            style={{
              background: "none",
              border: "none",
              color: "var(--text-muted)",
              cursor: "pointer",
              fontSize: "1rem",
            }}
          >
            ✕
          </button>
        </div>
        <button
          onClick={onNew}
          aria-label="新建对话"
          style={{
            margin: "0.5rem 0.75rem",
            padding: "0.5rem",
            background: "var(--bg-button)",
            color: "#fff",
            border: "none",
            borderRadius: "0.375rem",
            fontSize: "0.875rem",
            cursor: "pointer",
          }}
          onMouseEnter={(e) => {
            e.currentTarget.style.background = "var(--bg-button-hover)";
          }}
          onMouseLeave={(e) => {
            e.currentTarget.style.background = "var(--bg-button)";
          }}
        >
          + 新对话
        </button>
        <div role="list" aria-label="对话列表" style={{ flex: 1, overflowY: "auto" }}>
          {conversations.map((conv) => {
              const isGenerating = generatingIds.has(conv.id);
              return (
                <div
                   key={conv.id}
                   role="listitem"
                   tabIndex={0}
                   aria-label={`${conv.title}, ${conv.messageCount}条消息${conv.id === activeId ? " (当前)" : ""}`}
                   onClick={() => onSelect(conv.id)}
                  onMouseEnter={() => setHoveredId(conv.id)}
                  onMouseLeave={() => setHoveredId(null)}
                  style={{
                    padding: "0.5rem 1rem",
                    cursor: "pointer",
                    borderBottom: "1px solid var(--border-light)",
                    background: conv.id === activeId ? "var(--bg-active)" : "transparent",
                    position: "relative",
                  }}
                  onMouseOver={(e) => {
                    if (conv.id !== activeId) {
                      e.currentTarget.style.background = "var(--bg-hover)";
                    }
                  }}
                  onMouseOut={(e) => {
                    if (conv.id !== activeId) {
                      e.currentTarget.style.background = "transparent";
                    }
                  }}
                >
                  <div
                    style={{
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                      whiteSpace: "nowrap",
                      fontSize: "0.875rem",
                      color: conv.id === activeId ? "var(--text-primary)" : "var(--text-muted)",
                      paddingRight: "3.5rem",
                    }}
                  >
                    {isGenerating ? "⏳ 生成中..." : conv.title}
                  </div>
                  <div style={{ fontSize: "0.75rem", color: "var(--text-dim)", marginTop: "0.25rem" }}>
                    {conv.status === "active" ? "🔄" : "✓"} {conv.messageCount} 条消息
                  </div>
                  {hoveredId === conv.id && (
                    <div
                      style={{
                        position: "absolute",
                        top: "0.5rem",
                        right: "0.5rem",
                        display: "flex",
                        gap: "0.25rem",
                      }}
                      onClick={(e) => e.stopPropagation()}
                    >
                 {/* 重新生成标题按钮（始终显示，生成中时禁用） */}
                  {onGenerateTitle && (
                     <button
                       onClick={() => !isGenerating && onGenerateTitle(conv.id)}
                       disabled={isGenerating}
                       aria-label={isGenerating ? "正在生成标题" : "重新生成标题"}
                       style={{
                        background: isGenerating ? "var(--bg-disabled)" : "var(--bg-button)",
                        color: "#fff",
                        border: "none",
                        borderRadius: "0.25rem",
                        width: "1.25rem",
                        height: "1.25rem",
                        fontSize: "0.625rem",
                        cursor: isGenerating ? "not-allowed" : "pointer",
                        display: "flex",
                        alignItems: "center",
                        justifyContent: "center",
                        lineHeight: 1,
                        opacity: isGenerating ? 0.5 : 1,
                      }}
                      title={isGenerating ? "正在生成中..." : "重新生成标题"}
                    >
                      {isGenerating ? "⏳" : "🔄"}
                    </button>
                  )}
                      {/* 编辑标题按钮（生成中时禁用） */}
                       <button
                         onClick={() => !isGenerating && handleStartEdit(conv.id, conv.title)}
                         disabled={isGenerating}
                         aria-label="编辑标题"
                         style={{
                          background: isGenerating ? "var(--bg-disabled)" : "var(--bg-button)",
                          color: "#fff",
                          border: "none",
                          borderRadius: "0.25rem",
                          width: "1.25rem",
                          height: "1.25rem",
                          fontSize: "0.625rem",
                          cursor: isGenerating ? "not-allowed" : "pointer",
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          lineHeight: 1,
                          opacity: isGenerating ? 0.5 : 1,
                        }}
                        title={isGenerating ? "正在生成中..." : "编辑标题"}
                      >
                        ✏️
                      </button>
                      {/* 删除按钮 */}
                       <button
                         onClick={() => setDeleteTarget(conv.id)}
                         aria-label="删除对话"
                         style={{
                          background: "var(--bg-danger)",
                          color: "#fff",
                          border: "none",
                          borderRadius: "0.25rem",
                          width: "1.25rem",
                          height: "1.25rem",
                          fontSize: "0.625rem",
                          cursor: "pointer",
                          display: "flex",
                          alignItems: "center",
                          justifyContent: "center",
                          lineHeight: 1,
                        }}
                        title="删除对话"
                      >
                        ✕
                      </button>
                    </div>
                  )}
                </div>
              );
            })}
        </div>
      </div>

      {/* 删除确认对话框 */}
      {deleteTarget && (
        <div
          className="confirm-overlay"
          onClick={(e) => {
            if (e.target === e.currentTarget) setDeleteTarget(null);
          }}
        >
          <div className="confirm-box">
            <div style={{ fontSize: "0.9375rem", fontWeight: 600, marginBottom: "0.5rem" }}>
              确认删除
            </div>
            <div
              style={{
                fontSize: "0.875rem",
                color: "var(--text-muted)",
                marginBottom: "1rem",
              }}
            >
              确定要删除这个对话吗？此操作不可恢复。
            </div>
            <div style={{ display: "flex", justifyContent: "flex-end", gap: "0.5rem" }}>
              <button
                onClick={() => setDeleteTarget(null)}
                style={{
                  padding: "0.375rem 0.75rem",
                  fontSize: "0.8125rem",
                  borderRadius: "0.25rem",
                  border: "1px solid var(--border-color)",
                  background: "transparent",
                  color: "var(--text-muted)",
                  cursor: "pointer",
                }}
              >
                取消
              </button>
              <button
                onClick={handleConfirmDelete}
                style={{
                  padding: "0.375rem 0.75rem",
                  fontSize: "0.8125rem",
                  borderRadius: "0.25rem",
                  border: "none",
                  background: "var(--bg-danger)",
                  color: "#fff",
                  cursor: "pointer",
                }}
              >
                删除
              </button>
            </div>
          </div>
        </div>
      )}

      {/* 编辑标题对话框 */}
      {editTarget && (
        <div
          className="confirm-overlay"
          onClick={(e) => {
            if (e.target === e.currentTarget) handleCancelEdit();
          }}
        >
          <div className="confirm-box">
            <div style={{ fontSize: "0.9375rem", fontWeight: 600, marginBottom: "0.75rem" }}>
              编辑标题
            </div>
            <input
              type="text"
              value={editTitle}
              onChange={(e) => setEditTitle(e.target.value)}
              onKeyDown={handleEditKeyDown}
              aria-label="对话标题"
              autoFocus
              style={{
                width: "100%",
                padding: "0.5rem",
                fontSize: "0.875rem",
                borderRadius: "0.25rem",
                border: "1px solid var(--border-color)",
                background: "var(--bg-input)",
                color: "var(--text-primary)",
                marginBottom: "1rem",
                boxSizing: "border-box",
              }}
            />
            <div style={{ display: "flex", justifyContent: "flex-end", gap: "0.5rem" }}>
              <button
                onClick={handleCancelEdit}
                style={{
                  padding: "0.375rem 0.75rem",
                  fontSize: "0.8125rem",
                  borderRadius: "0.25rem",
                  border: "1px solid var(--border-color)",
                  background: "transparent",
                  color: "var(--text-muted)",
                  cursor: "pointer",
                }}
              >
                取消
              </button>
              <button
                onClick={handleConfirmEdit}
                style={{
                  padding: "0.375rem 0.75rem",
                  fontSize: "0.8125rem",
                  borderRadius: "0.25rem",
                  border: "none",
                  background: "var(--bg-button)",
                  color: "#fff",
                  cursor: "pointer",
                }}
              >
                保存
              </button>
            </div>
          </div>
        </div>
      )}
    </>
  );
}
