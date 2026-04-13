import React, { useState, useCallback, useEffect } from "react";
import { useConversation } from "./hooks/useConversation";
import { useStreamEvents } from "./hooks/useStreamEvents";
import { isWailsReady, getWorkDir as wailsGetWorkDir } from "./wailsRuntime";
import ErrorBoundary from "./components/ErrorBoundary";
import Toolbar from "./components/Toolbar";
import MessageList from "./components/MessageList";
import InputArea from "./components/InputArea";
import Sidebar from "./components/Sidebar";
import Settings from "./components/Settings";
import SupervisorPanel from "./components/SupervisorPanel";
import TerminalPanel from "./components/TerminalPanel";
import type { ConversationData, TokenStats, SupervisorResult } from "./types";

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
    changeWorkDir,
    switchConv,
    deleteConv,
    exportConv,
  } = useConversation();

  const [sidebarVisible, setSidebarVisible] = useState(false);
  const [settingsVisible, setSettingsVisible] = useState(false);
  const [terminalVisible, setTerminalVisible] = useState(false);
  const [wailsReady, setWailsReady] = useState(false);
  const [isDarkTheme, setIsDarkTheme] = useState(getInitialTheme);
  const [supervisorResults, setSupervisorResults] = useState<SupervisorResult[]>([]);

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

  // 新对话时清空监督结果
  useEffect(() => {
    if (!conversationData.isStreaming && conversationData.messages.length === 0) {
      setSupervisorResults([]);
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
    <ErrorBoundary>
      <div style={{ display: "flex", height: "100vh", background: "var(--bg-app)", color: "var(--text-primary)" }}>
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
          onNew={clearConv}
          onDelete={deleteConv}
        />
        <div style={{ flex: 1, display: "flex", flexDirection: "column", minWidth: 0 }}>
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
          />
          {error && (
            <div style={{ background: "var(--bg-error)", color: "var(--text-red)", padding: "0.5rem 1rem", fontSize: "0.875rem", display: "flex", justifyContent: "space-between" }}>
              <span>{error}</span>
              <button onClick={() => setError(null)} style={{ background: "none", border: "none", color: "var(--text-red-hover)", cursor: "pointer" }}>✕</button>
            </div>
          )}
          <MessageList messages={conversationData.messages} />
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
  );
}

export default App;
