import type { ConversationData, TokenStats, Conversation } from "./types";

import {
  SendMessage as wailsSendMessage,
  GetConversationData as wailsGetConversationData,
  GetConversations as wailsGetConversations,
  NewConversation as wailsNewConversation,
  SetWorkDir as wailsSetWorkDir,
  GetTokenStats as wailsGetTokenStats,
  GetWorkDir as wailsGetWorkDir,
  InterruptInference as wailsInterruptInference,
  ClearConversation as wailsClearConversation,
  SwitchConversation as wailsSwitchConversation,
  GetSettings as wailsGetSettings,
  SaveSettings as wailsSaveSettings,
  DeleteConversation as wailsDeleteConversation,
  UpdateConversationTitle as wailsUpdateConversationTitle,
  ExportConversation as wailsExportConversation,
  ChooseDirectory as wailsChooseDirectory,
  GenerateConversationTitle as wailsGenerateConversationTitle,
  RegenerateConversationTitle as wailsRegenerateConversationTitle,
  GetMCPToolList as wailsGetMCPToolList,
} from "../wailsjs/go/gui/App";

import { EventsOn } from "../wailsjs/runtime/runtime";

export function isWailsReady(): boolean {
  const w = window as any;
  return !!(w.go && w.go.gui && w.go.gui.App);
}

export async function sendMessage(content: string): Promise<void> {
  try {
    await wailsSendMessage(content);
  } catch (err) {
    console.error("sendMessage error:", err);
    throw err;
  }
}

export async function getConversationData(): Promise<ConversationData> {
  return (await wailsGetConversationData()) as unknown as ConversationData;
}

export async function setWorkDir(path: string): Promise<void> {
  await wailsSetWorkDir(path);
}

export async function getTokenStats(): Promise<TokenStats> {
  return (await wailsGetTokenStats()) as unknown as TokenStats;
}

export async function getWorkDir(): Promise<string> {
  return await wailsGetWorkDir();
}

export async function interruptInference(): Promise<void> {
  await wailsInterruptInference();
}

export async function clearConversation(): Promise<void> {
  await wailsClearConversation();
}

export async function newConversation(): Promise<void> {
  await wailsNewConversation();
}

export async function getConversations(): Promise<Conversation[]> {
  return (await wailsGetConversations()) as unknown as Conversation[];
}

export async function switchConversation(id: string): Promise<void> {
  await wailsSwitchConversation(id);
}

export async function getSettings(): Promise<Record<string, any>> {
  return (await wailsGetSettings()) as Record<string, any>;
}

export async function saveSettings(settings: Record<string, any>): Promise<void> {
  await wailsSaveSettings(settings);
}

export async function deleteConversation(id: string): Promise<void> {
  await wailsDeleteConversation(id);
}

export async function updateConversationTitle(id: string, title: string): Promise<void> {
  await wailsUpdateConversationTitle(id, title);
}

export async function exportConversation(id: string, filename: string): Promise<void> {
  await wailsExportConversation(id, filename);
}

export async function chooseDirectory(): Promise<string> {
  return await wailsChooseDirectory();
}

export async function generateConversationTitle(id: string): Promise<void> {
  await wailsGenerateConversationTitle(id);
}

export async function regenerateConversationTitle(id: string): Promise<void> {
  await wailsRegenerateConversationTitle(id);
}

export async function getMCPToolList(serverID: string): Promise<any[]> {
  return (await wailsGetMCPToolList(serverID)) as any[];
}

export function onEvent(event: string, callback: (...args: any[]) => void): () => void {
  return EventsOn(event, callback);
}
