import { useState, useCallback } from "react";
import type { ConversationData, TokenStats, Conversation } from "../types";
import {
  sendMessage as wailsSendMessage,
  interruptInference as wailsInterruptInference,
  clearConversation as wailsClearConversation,
  setWorkDir as wailsSetWorkDir,
  getConversations as wailsGetConversations,
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

  const changeWorkDir = useCallback(async (path: string) => {
    try {
      await wailsSetWorkDir(path);
    } catch (err: any) {
      setError(err?.message || String(err));
    }
  }, []);

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
    changeWorkDir,
  };
}
