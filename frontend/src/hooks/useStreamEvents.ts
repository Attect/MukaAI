import { useEffect } from "react";
import { onEvent } from "../wailsRuntime";
import type { ConversationData, TokenStats } from "../types";

interface UseStreamEventsProps {
  onConversationUpdated: (data: ConversationData) => void;
  onTokenStatsUpdated: (stats: TokenStats) => void;
  onStreamDone: () => void;
  onStreamError: (error: string) => void;
  onWorkDirChanged: (dir: string) => void;
}

export function useStreamEvents({
  onConversationUpdated,
  onTokenStatsUpdated,
  onStreamDone,
  onStreamError,
  onWorkDirChanged,
}: UseStreamEventsProps) {
  useEffect(() => {
    onEvent("conversation:updated", (data: ConversationData) => {
      onConversationUpdated(data);
    });
    onEvent("tokenstats:updated", (stats: TokenStats) => {
      onTokenStatsUpdated(stats);
    });
    onEvent("stream:done", () => {
      onStreamDone();
    });
    onEvent("stream:error", (error: string) => {
      onStreamError(error);
    });
    onEvent("workdir:changed", (dir: string) => {
      onWorkDirChanged(dir);
    });
  }, [onConversationUpdated, onTokenStatsUpdated, onStreamDone, onStreamError, onWorkDirChanged]);
}
