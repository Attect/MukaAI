import type { ConversationData, TokenStats, Conversation } from "./types";

import {
  SendMessage as wailsSendMessage,
  GetConversationData as wailsGetConversationData,
  GetConversations as wailsGetConversations,
  SetWorkDir as wailsSetWorkDir,
  GetTokenStats as wailsGetTokenStats,
  GetWorkDir as wailsGetWorkDir,
  InterruptInference as wailsInterruptInference,
  ClearConversation as wailsClearConversation,
  SwitchConversation as wailsSwitchConversation,
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

export async function getConversations(): Promise<Conversation[]> {
  return (await wailsGetConversations()) as unknown as Conversation[];
}

export async function switchConversation(id: string): Promise<void> {
  await wailsSwitchConversation(id);
}

export function onEvent(event: string, callback: (...args: any[]) => void): void {
  EventsOn(event, callback);
}
