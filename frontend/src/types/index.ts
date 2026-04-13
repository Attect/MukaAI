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

/** Supervisor监督级别 */
export type SupervisorLevel = "info" | "warning" | "correction" | "halt";

/** Supervisor监督结果 */
export interface SupervisorResult {
  /** 消息索引，关联到哪条消息 */
  messageIndex?: number;
  /** 干预级别 */
  level: SupervisorLevel;
  /** 检查类型 */
  checkType: string;
  /** 摘要信息 */
  summary: string;
  /** 详细信息 */
  details?: string;
  /** 时间戳 */
  timestamp?: string;
}
