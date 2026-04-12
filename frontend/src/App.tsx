import React, { useState, useCallback, useEffect } from "react";
import { useConversation } from "./hooks/useConversation";
import { useStreamEvents } from "./hooks/useStreamEvents";
import { isWailsReady, getWorkDir as wailsGetWorkDir } from "./wailsRuntime";
import ErrorBoundary from "./components/ErrorBoundary";
import Toolbar from "./components/Toolbar";
import MessageList from "./components/MessageList";
import InputArea from "./components/InputArea";
import Sidebar from "./components/Sidebar";
import type { ConversationData, TokenStats } from "./types";

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
  } = useConversation();

  const [sidebarVisible, setSidebarVisible] = useState(false);
  const [wailsReady, setWailsReady] = useState(false);

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

  const handleWorkDirChanged = useCallback((dir: string) => {
    setWorkDir(dir);
  }, [setWorkDir]);

  useStreamEvents({
    onConversationUpdated: handleConversationUpdated,
    onTokenStatsUpdated: handleTokenStatsUpdated,
    onStreamDone: handleStreamDone,
    onStreamError: handleStreamError,
    onWorkDirChanged: handleWorkDirChanged,
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
      case "/help":
        alert("可用命令：/cd, /clear, /save, /help, /exit");
        break;
      case "/exit":
        window.close();
        break;
      default:
        setError(`未知命令: ${command}`);
    }
  }, [changeWorkDir, clearConv, setError]);

  if (!wailsReady) {
    return (
      <div className="flex h-screen items-center justify-center bg-gray-900 text-white">
        <div className="text-center">
          <div className="text-2xl font-bold mb-2">AgentPlus</div>
          <div className="text-gray-400 animate-pulse">正在初始化...</div>
        </div>
      </div>
    );
  }

  return (
    <ErrorBoundary>
    <div className="flex h-screen bg-gray-900 text-white">
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
      />
      <div className="flex-1 flex flex-col min-w-0">
        <Toolbar
          workDir={workDir}
          tokenStats={tokenStats}
          isStreaming={conversationData.isStreaming}
          onInterrupt={interruptInference}
          onClear={clearConv}
          onToggleSidebar={() => setSidebarVisible(!sidebarVisible)}
        />
        {error && (
          <div className="bg-red-900/50 text-red-300 px-4 py-2 text-sm flex justify-between">
            <span>{error}</span>
            <button onClick={() => setError(null)} className="text-red-400 hover:text-red-200">✕</button>
          </div>
        )}
        <MessageList messages={conversationData.messages} />
        <InputArea
          isStreaming={conversationData.isStreaming}
          onSend={sendMessage}
          onCommand={handleCommand}
        />
      </div>
    </div>
    </ErrorBoundary>
  );
}

export default App;
