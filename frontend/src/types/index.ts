export interface Message {
  role: "user" | "assistant" | "tool";
  content: string;
  thinking: string;
  toolCalls: ToolCall[];
  tokenUsage: number;
  isStreaming: boolean;
  streamingType: string;
  timestamp: string;
}

export interface ToolCall {
  id: string;
  name: string;
  arguments: string;
  isComplete: boolean;
  result: string;
  resultError: string;
}

export interface ConversationData {
  id: string;
  messages: Message[];
  isStreaming: boolean;
}

export interface TokenStats {
  totalTokens: number;
  inferenceCount: number;
}

export interface Conversation {
  id: string;
  title: string;
  createdAt: string;
  status: string;
  tokenUsage: number;
  messageCount: number;
}
