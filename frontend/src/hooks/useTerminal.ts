import { useState, useEffect, useRef, useCallback } from "react";

// 终端 WebSocket 消息类型
interface TerminalMessage {
  type: string;
  data?: string;
  rows?: number;
  cols?: number;
  signal?: string;
  code?: number;
  message?: string;
}

// Hook 返回值
interface UseTerminalReturn {
  /** WebSocket 连接状态 */
  connected: boolean;
  /** 终端 WebSocket URL */
  wsUrl: string;
  /** 发送输入到终端 */
  sendInput: (data: string) => void;
  /** 调整终端尺寸 */
  sendResize: (cols: number, rows: number) => void;
  /** 发送信号 */
  sendSignal: (signal: string) => void;
  /** 终端输出回调注册 */
  onOutput: ((data: string) => void) | null;
  setOnOutput: (cb: ((data: string) => void) | null) => void;
  /** 终端退出回调注册 */
  onExit: ((code: number) => void) | null;
  setOnExit: (cb: ((code: number) => void) | null) => void;
  /** 错误回调注册 */
  onError: ((message: string) => void) | null;
  setOnError: (cb: ((message: string) => void) | null) => void;
  /** 手动重连 */
  reconnect: () => void;
}

/**
 * useTerminal - 终端 WebSocket 连接 Hook
 *
 * 管理与后端终端 WebSocket 服务器的连接，
 * 提供发送输入、调整大小、发送信号等功能。
 *
 * 连接地址通过 Wails 绑定的 GetTerminalWSUrl 方法获取。
 */
export function useTerminal(): UseTerminalReturn {
  const [connected, setConnected] = useState(false);
  const [wsUrl, setWsUrl] = useState("");
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const outputCbRef = useRef<((data: string) => void) | null>(null);
  const exitCbRef = useRef<((code: number) => void) | null>(null);
  const errorCbRef = useRef<((message: string) => void) | null>(null);
  const [onOutput, setOnOutput] = useState<((data: string) => void) | null>(null);
  const [onExit, setOnExit] = useState<((code: number) => void) | null>(null);
  const [onError, setOnError] = useState<((message: string) => void) | null>(null);

  // 同步回调引用
  useEffect(() => { outputCbRef.current = onOutput; }, [onOutput]);
  useEffect(() => { exitCbRef.current = onExit; }, [onExit]);
  useEffect(() => { errorCbRef.current = onError; }, [onError]);

  // 获取 WebSocket URL
  useEffect(() => {
    const fetchUrl = async () => {
      try {
        // 动态获取 Wails 运行时中的 GetTerminalWSUrl 方法
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const wailsGo = (window as any).go;
        if (wailsGo && wailsGo.gui && wailsGo.gui.App && wailsGo.gui.App.GetTerminalWSUrl) {
          const url: string = await wailsGo.gui.App.GetTerminalWSUrl();
          if (url) {
            setWsUrl(url);
          }
        }
      } catch {
        // Wails 运行时未就绪，忽略
      }
    };
    fetchUrl();
    // 定期检查 URL（Wails 可能还未设置）
    const interval = setInterval(fetchUrl, 2000);
    return () => clearInterval(interval);
  }, []);

  // 建立 WebSocket 连接
  const connect = useCallback(() => {
    if (!wsUrl) return;
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    // 清理旧连接
    if (wsRef.current) {
      wsRef.current.close();
      wsRef.current = null;
    }

    try {
      const ws = new WebSocket(wsUrl);

      ws.onopen = () => {
        setConnected(true);
      };

      ws.onmessage = (event) => {
        try {
          const msg: TerminalMessage = JSON.parse(event.data);
          switch (msg.type) {
            case "output":
              outputCbRef.current?.(msg.data || "");
              break;
            case "exit":
              setConnected(false);
              exitCbRef.current?.(msg.code || 0);
              break;
            case "error":
              errorCbRef.current?.(msg.message || "Unknown error");
              break;
          }
        } catch {
          // 忽略解析错误
        }
      };

      ws.onclose = () => {
        setConnected(false);
        // 自动重连（3秒后）
        reconnectTimerRef.current = setTimeout(() => {
          connect();
        }, 3000);
      };

      ws.onerror = () => {
        // 错误由 onclose 处理
      };

      wsRef.current = ws;
    } catch {
      // 连接失败，3秒后重试
      reconnectTimerRef.current = setTimeout(() => {
        connect();
      }, 3000);
    }
  }, [wsUrl]);

  // 当 URL 可用时自动连接
  useEffect(() => {
    if (wsUrl) {
      connect();
    }
    return () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
    };
  }, [wsUrl, connect]);

  // 发送输入
  const sendInput = useCallback((data: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      const msg: TerminalMessage = { type: "input", data };
      wsRef.current.send(JSON.stringify(msg));
    }
  }, []);

  // 调整大小
  const sendResize = useCallback((cols: number, rows: number) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      const msg: TerminalMessage = { type: "resize", cols, rows };
      wsRef.current.send(JSON.stringify(msg));
    }
  }, []);

  // 发送信号
  const sendSignal = useCallback((signal: string) => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      const msg: TerminalMessage = { type: "signal", signal };
      wsRef.current.send(JSON.stringify(msg));
    }
  }, []);

  return {
    connected,
    wsUrl,
    sendInput,
    sendResize,
    sendSignal,
    onOutput,
    setOnOutput,
    onExit,
    setOnExit,
    onError,
    setOnError,
    reconnect: connect,
  };
}
