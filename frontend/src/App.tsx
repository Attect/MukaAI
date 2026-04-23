import React, { useState, useCallback, useEffect } from "react";
import { useConversation } from "./hooks/useConversation";
import { useStreamEvents } from "./hooks/useStreamEvents";
import { isWailsReady, getWorkDir as wailsGetWorkDir, chooseDirectory } from "./wailsRuntime";
import ErrorBoundary from "./components/ErrorBoundary";
import Toolbar from "./components/Toolbar";
import MessageList from "./components/MessageList";
import InputArea from "./components/InputArea";
import Sidebar from "./components/Sidebar";
import Settings from "./components/Settings";
import SupervisorPanel from "./components/SupervisorPanel";
import TerminalPanel from "./components/TerminalPanel";
import { UIProvider } from "./contexts/UIContext";
import type { ConversationData, TokenStats, SupervisorResult, CompressionEvent, MessageListItem } from "./types";

// 主题管理工具函数
function getInitialTheme(): boolean {
  const stored = localStorage.getItem("theme");
  if (stored === "light") return false;
  if (stored === "dark") return true;
  // 跟随系统偏好，无偏好时默认暗色
  return !window.matchMedia("(prefers-color-scheme: light)").matches;
}

function applyTheme(dark: boolean): void {
  document.documentElement.setAttribute("data-theme", dark ? "dark" : "light");
  localStorage.setItem("theme", dark ? "dark" : "light");
}

function App(): React.ReactElement {
  const {
    conversationData,
    setConversationData,
    tokenStats,
    setTokenStats,
    workDir,
    setWorkDir,
    error,
    setError,
    conversations,
    loadConversations,
    sendMessage,
    interruptInference,
    clearConv,
    newConv,
    changeWorkDir,
    switchConv,
    deleteConv,
    exportConv,
    regenerateTitle,
    updateTitle,
    generatingIds,
  } = useConversation();

  const [sidebarVisible, setSidebarVisible] = useState(true);
  const [settingsVisible, setSettingsVisible] = useState(false);
  const [terminalVisible, setTerminalVisible] = useState(false);
  const [wailsReady, setWailsReady] = useState(false);
  const [isDarkTheme, setIsDarkTheme] = useState(getInitialTheme);
  const [supervisorResults, setSupervisorResults] = useState<SupervisorResult[]>([]);
  const [compressionEvents, setCompressionEvents] = useState<CompressionEvent[]>([]);
  const [autoNewConv, setAutoNewConv] = useState(false);

  // 初始化主题
  useEffect(() => {
    applyTheme(isDarkTheme);
  }, [isDarkTheme]);

  useEffect(() => {
    const checkReady = () => {
      if (isWailsReady()) {
        setWailsReady(true);
        wailsGetWorkDir().then((dir) => {
          if (dir) setWorkDir(dir);
        }).catch(console.error);
        loadConversations();
      } else {
        setTimeout(checkReady, 100);
      }
    };
    checkReady();
  }, [loadConversations, setWorkDir]);

  // 启动时自动创建新对话（如果设置了 autoNewConv）
  useEffect(() => {
    if (wailsReady && autoNewConv && conversations.length > 0) {
      newConv();
    }
  }, [wailsReady, autoNewConv, conversations.length, newConv]);

  const handleConversationUpdated = useCallback((data: ConversationData) => {
    setConversationData(data);
  }, [setConversationData]);

  const handleTokenStatsUpdated = useCallback((stats: TokenStats) => {
    setTokenStats(stats);
  }, [setTokenStats]);

  const handleStreamDone = useCallback(() => {
    setConversationData((prev) => ({ ...prev, isStreaming: false }));
    loadConversations();
  }, [setConversationData, loadConversations]);

  const handleStreamError = useCallback((err: string) => {
    setError(err);
  }, [setError]);

  const handleSupervisorResult = useCallback((result: SupervisorResult) => {
    setSupervisorResults((prev) => [...prev, result]);
  }, []);

  const handleCompression = useCallback((event: CompressionEvent) => {
    setCompressionEvents((prev) => [...prev, event]);
  }, []);

  // 新对话时清空监督结果和压缩事件
  useEffect(() => {
    if (!conversationData.isStreaming && conversationData.messages.length === 0) {
      setSupervisorResults([]);
      setCompressionEvents([]);
    }
  }, [conversationData.isStreaming, conversationData.messages.length]);

  const handleWorkDirChanged = useCallback((dir: string) => {
    setWorkDir(dir);
  }, [setWorkDir]);

  useStreamEvents({
    onConversationUpdated: handleConversationUpdated,
    onTokenStatsUpdated: handleTokenStatsUpdated,
    onStreamDone: handleStreamDone,
    onStreamError: handleStreamError,
    onWorkDirChanged: handleWorkDirChanged,
    onSupervisorResult: handleSupervisorResult,
    onCompression: handleCompression,
  });

  const handleCommand = useCallback((cmd: string) => {
    const parts = cmd.split(" ");
    const command = parts[0].toLowerCase();
    switch (command) {
      case "/cd":
        if (parts[1]) changeWorkDir(parts[1]);
        break;
      case "/clear":
        clearConv();
        break;
      case "/save": {
        const filename = parts.slice(1).join(" ").trim();
        exportConv(conversationData.id || "", filename);
        break;
      }
      case "/help":
        alert("可用命令：/cd, /clear, /save, /help, /exit");
        break;
      case "/exit":
        window.close();
        break;
      default:
        setError(`未知命令: ${command}`);
    }
  }, [changeWorkDir, clearConv, conversationData.id, exportConv, setError]);

  const handleToggleTheme = useCallback(() => {
    setIsDarkTheme((prev) => {
      const next = !prev;
      applyTheme(next);
      return next;
    });
  }, []);

  const handleChooseWorkDir = useCallback(async () => {
    try {
      const chosen = await chooseDirectory();
      if (chosen) {
        changeWorkDir(chosen);
      }
    } catch (err: any) {
      setError(err?.message || String(err));
    }
  }, [changeWorkDir, setError]);

  // 构建合并后的消息列表（消息 + 压缩事件）
  const messageListItems: MessageListItem[] = React.useMemo(() => {
    const messages = conversationData.messages || [];
    
    // 按时间戳排序，将消息和压缩事件合并
    const allEvents: Array<{ type: "message" | "compression"; timestamp: string; item: MessageListItem }> = [];
    
    messages.forEach((msg, idx) => {
      allEvents.push({
        type: "message",
        timestamp: msg.timestamp || "",
        item: { type: "message", data: msg, key: `msg-${idx}` },
      });
    });
    
    compressionEvents.forEach((event, idx) => {
      allEvents.push({
        type: "compression",
        timestamp: event.timestamp || "",
        item: { type: "compression", data: event, key: `comp-${idx}` },
      });
    });
    
    // 按时间戳排序
    allEvents.sort((a, b) => a.timestamp.localeCompare(b.timestamp));
    
    return allEvents.map((e) => e.item);
  }, [conversationData.messages, compressionEvents]);

  if (!wailsReady) {
    return (
      <div style={{ display: "flex", height: "100vh", alignItems: "center", justifyContent: "center", background: "var(--bg-app)", color: "var(--text-primary)" }}>
        <div style={{ textAlign: "center" }}>
          <div style={{ fontSize: "1.5rem", fontWeight: 700, marginBottom: "0.5rem" }}>MukaAI</div>
          <div style={{ color: "var(--text-muted)" }} className="animate-pulse">正在初始化...</div>
        </div>
      </div>
    );
  }

  return (
    <UIProvider>
      <ErrorBoundary>
        <div role="application" aria-label="MukaAI 智能编程助手" style={{ display: "flex", height: "100vh", background: "var(--bg-app)", color: "var(--text-primary)" }}>
        <Sidebar
          visible={sidebarVisible}
          conversations={conversations}
          activeId={conversationData.id}
          onSelect={(id) => {
            if (id !== conversationData.id) {
              switchConv(id);
            }
            setSidebarVisible(false);
          }}
          onClose={() => setSidebarVisible(false)}
          onNew={newConv}
          onDelete={deleteConv}
          onGenerateTitle={regenerateTitle}
          onUpdateTitle={updateTitle}
          generatingIds={generatingIds}
        />
        <div role="main" aria-label="对话区域" style={{ flex: 1, display: "flex", flexDirection: "column", minWidth: 0 }}>
          <Toolbar
            workDir={workDir}
            tokenStats={tokenStats}
            isStreaming={conversationData.isStreaming}
            onInterrupt={interruptInference}
            onClear={clearConv}
            onToggleSidebar={() => setSidebarVisible(!sidebarVisible)}
            onToggleSettings={() => setSettingsVisible(!settingsVisible)}
            onToggleTerminal={() => setTerminalVisible(!terminalVisible)}
            isTerminalVisible={terminalVisible}
            onToggleTheme={handleToggleTheme}
            isDarkTheme={isDarkTheme}
            onChooseWorkDir={handleChooseWorkDir}
          />
          {error && (
            <div style={{ background: "var(--bg-error)", color: "var(--text-red)", padding: "0.5rem 1rem", fontSize: "0.875rem", display: "flex", justifyContent: "space-between" }}>
              <span>{error}</span>
              <button onClick={() => setError(null)} style={{ background: "none", border: "none", color: "var(--text-red-hover)", cursor: "pointer" }}>✕</button>
            </div>
          )}
          <MessageList items={messageListItems} isStreaming={conversationData.isStreaming} />
          <SupervisorPanel results={supervisorResults} />
          <TerminalPanel
            visible={terminalVisible}
            onClose={() => setTerminalVisible(false)}
          />
          <InputArea
            isStreaming={conversationData.isStreaming}
            onSend={sendMessage}
            onCommand={handleCommand}
          />
        </div>
        <Settings
          visible={settingsVisible}
          onClose={() => setSettingsVisible(false)}
        />
       </div>
      </ErrorBoundary>
    </UIProvider>
  );
}

export default App;
