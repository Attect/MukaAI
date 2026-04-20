import { useState, useCallback } from "react";
import type { ConversationData, TokenStats, Conversation } from "../types";
import {
  sendMessage as wailsSendMessage,
  interruptInference as wailsInterruptInference,
  clearConversation as wailsClearConversation,
  newConversation as wailsNewConversation,
  setWorkDir as wailsSetWorkDir,
  getConversations as wailsGetConversations,
  switchConversation as wailsSwitchConversation,
  deleteConversation as wailsDeleteConversation,
  exportConversation as wailsExportConversation,
  generateConversationTitle as wailsGenerateConversationTitle,
  regenerateConversationTitle as wailsRegenerateConversationTitle,
  updateConversationTitle as wailsUpdateConversationTitle,
} from "../wailsRuntime";

export function useConversation() {
  const [conversationData, setConversationData] = useState<ConversationData>({
    id: "",
    messages: [],
    isStreaming: false,
  });
  const [tokenStats, setTokenStats] = useState<TokenStats>({
    totalTokens: 0,
    inferenceCount: 0,
  });
  const [workDir, setWorkDir] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [generatingIds, setGeneratingIds] = useState<Set<string>>(new Set());

  const loadConversations = useCallback(async () => {
    try {
      const result = await wailsGetConversations();
      setConversations(result || []);
    } catch (err: any) {
      console.error("Failed to load conversations:", err);
    }
  }, []);

  const sendMessage = useCallback(async (content: string) => {
    setError(null);
    try {
      await wailsSendMessage(content);
    } catch (err: any) {
      setError(err?.message || String(err));
    }
  }, []);

  const interruptInference = useCallback(async () => {
    try {
      await wailsInterruptInference();
    } catch (err: any) {
      setError(err?.message || String(err));
    }
  }, []);

  const clearConv = useCallback(async () => {
    try {
      await wailsClearConversation();
      await loadConversations();
    } catch (err: any) {
      setError(err?.message || String(err));
    }
  }, [loadConversations]);

  const newConv = useCallback(async () => {
    setError(null);
    try {
      await wailsNewConversation();
      await loadConversations();
    } catch (err: any) {
      setError(err?.message || String(err));
    }
  }, [loadConversations]);

  const changeWorkDir = useCallback(async (path: string) => {
    try {
      await wailsSetWorkDir(path);
    } catch (err: any) {
      setError(err?.message || String(err));
    }
  }, []);

  const switchConv = useCallback(async (id: string) => {
    setError(null);
    try {
      await wailsSwitchConversation(id);
      await loadConversations();
    } catch (err: any) {
      setError(err?.message || String(err));
    }
  }, [loadConversations]);

  const deleteConv = useCallback(async (id: string) => {
    try {
      await wailsDeleteConversation(id);
      await loadConversations();
    } catch (err: any) {
      setError(err?.message || String(err));
    }
  }, [loadConversations]);

  const exportConv = useCallback(async (id: string, filename: string) => {
    try {
      await wailsExportConversation(id, filename);
      setError(null);
    } catch (err: any) {
      setError(err?.message || String(err));
    }
  }, []);

  const generateTitle = useCallback(async (id: string) => {
    setGeneratingIds((prev) => new Set(prev).add(id));
    try {
      await wailsGenerateConversationTitle(id);
      await loadConversations();
    } catch (err: any) {
      console.error("Failed to generate title:", err);
      setError(err?.message || String(err));
    } finally {
      setGeneratingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  }, [loadConversations]);

  const regenerateTitle = useCallback(async (id: string) => {
    setGeneratingIds((prev) => new Set(prev).add(id));
    try {
      await wailsRegenerateConversationTitle(id);
      await loadConversations();
    } catch (err: any) {
      console.error("Failed to regenerate title:", err);
      setError(err?.message || String(err));
    } finally {
      setGeneratingIds((prev) => {
        const next = new Set(prev);
        next.delete(id);
        return next;
      });
    }
  }, [loadConversations]);

  const updateTitle = useCallback(async (id: string, title: string) => {
    try {
      await wailsUpdateConversationTitle(id, title);
      await loadConversations();
    } catch (err: any) {
      console.error("Failed to update title:", err);
      setError(err?.message || String(err));
    }
  }, [loadConversations]);

  return {
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
    generateTitle,
    regenerateTitle,
    updateTitle,
    generatingIds,
  };
}
