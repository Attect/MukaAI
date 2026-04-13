import React, { useEffect, useRef, useCallback } from "react";
import { Terminal } from "@xterm/xterm";
import { FitAddon } from "@xterm/addon-fit";
import { WebLinksAddon } from "@xterm/addon-web-links";
import "@xterm/xterm/css/xterm.css";
import { useTerminal } from "../hooks/useTerminal";

interface TerminalPanelProps {
  visible: boolean;
  onClose: () => void;
}

/**
 * TerminalPanel - 交互式终端面板组件
 *
 * 基于 xterm.js 的跨平台终端面板，通过 WebSocket 与后端 PTY 通信。
 * 支持用户直接输入命令，支持拖拽调整大小，可折叠/展开。
 */
export default function TerminalPanel({
  visible,
  onClose,
}: TerminalPanelProps): React.ReactElement | null {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<Terminal | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const resizeObserverRef = useRef<ResizeObserver | null>(null);

  const {
    connected,
    sendInput,
    sendResize,
    sendSignal,
    setOnOutput,
    setOnExit,
    setOnError,
  } = useTerminal();

  // 初始化 xterm.js 实例
  useEffect(() => {
    if (!terminalRef.current) return;

    const xterm = new Terminal({
      cursorBlink: true,
      cursorStyle: "bar",
      fontSize: 14,
      fontFamily: "'Cascadia Code', 'Fira Code', 'Consolas', 'Courier New', monospace",
      lineHeight: 1.2,
      scrollback: 5000,
      theme: {
        background: "#1e1e1e",
        foreground: "#d4d4d4",
        cursor: "#d4d4d4",
        cursorAccent: "#1e1e1e",
        selectionBackground: "#264f78",
        black: "#000000",
        red: "#cd3131",
        green: "#0dbc79",
        yellow: "#e5e510",
        blue: "#2472c8",
        magenta: "#bc3fbc",
        cyan: "#11a8cd",
        white: "#e5e5e5",
        brightBlack: "#666666",
        brightRed: "#f14c4c",
        brightGreen: "#23d18b",
        brightYellow: "#f5f543",
        brightBlue: "#3b8eea",
        brightMagenta: "#d670d6",
        brightCyan: "#29b8db",
        brightWhite: "#ffffff",
      },
      allowProposedApi: true,
    });

    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();

    xterm.loadAddon(fitAddon);
    xterm.loadAddon(webLinksAddon);
    xterm.open(terminalRef.current);

    xtermRef.current = xterm;
    fitAddonRef.current = fitAddon;

    // 初次适配
    try {
      fitAddon.fit();
    } catch {
      // 忽略适配错误（面板可能不可见）
    }

    return () => {
      xterm.dispose();
      xtermRef.current = null;
      fitAddonRef.current = null;
    };
  }, []);

  // 处理终端输入 → WebSocket
  useEffect(() => {
    const xterm = xtermRef.current;
    if (!xterm) return;

    const disposable = xterm.onData((data: string) => {
      sendInput(data);
    });

    return () => {
      disposable.dispose();
    };
  }, [sendInput]);

  // 处理 WebSocket 输出 → 终端显示
  useEffect(() => {
    setOnOutput((data: string) => {
      xtermRef.current?.write(data);
    });
  }, [setOnOutput]);

  // 处理终端退出
  useEffect(() => {
    setOnExit((code: number) => {
      xtermRef.current?.writeln(`\r\n[进程退出，退出码: ${code}]`);
    });
  }, [setOnExit]);

  // 处理错误
  useEffect(() => {
    setOnError((message: string) => {
      xtermRef.current?.writeln(`\r\n\x1b[31m[错误: ${message}]\x1b[0m`);
    });
  }, [setOnError]);

  // 处理面板可见性变化 → 重新适配大小
  useEffect(() => {
    if (visible && fitAddonRef.current) {
      // 延迟一帧，等 DOM 更新完成
      requestAnimationFrame(() => {
        try {
          fitAddonRef.current?.fit();
        } catch {
          // 忽略
        }
      });
    }
  }, [visible]);

  // 处理终端大小变化 → 通知后端
  const handleResize = useCallback(() => {
    if (!fitAddonRef.current || !xtermRef.current) return;
    try {
      fitAddonRef.current.fit();
      const cols = xtermRef.current.cols;
      const rows = xtermRef.current.rows;
      if (cols > 0 && rows > 0) {
        sendResize(cols, rows);
      }
    } catch {
      // 忽略
    }
  }, [sendResize]);

  // ResizeObserver 监听容器大小变化
  useEffect(() => {
    const container = terminalRef.current?.parentElement;
    if (!container) return;

    const observer = new ResizeObserver(() => {
      handleResize();
    });
    observer.observe(container);
    resizeObserverRef.current = observer;

    return () => {
      observer.disconnect();
    };
  }, [handleResize]);

  // 快捷键：Ctrl+C 发送中断信号
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
      if (e.ctrlKey && e.key === "c" && visible) {
        // 让 xterm.js 处理默认行为（写入 ^C），同时发送 SIGINT
        sendSignal("SIGINT");
      }
    };
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, [sendSignal, visible]);

  if (!visible) return null;

  return (
    <div
      style={{
        borderTop: "1px solid var(--border-color)",
        background: "#1e1e1e",
        display: "flex",
        flexDirection: "column",
        height: "300px",
        minHeight: "150px",
        position: "relative",
      }}
    >
      {/* 终端标题栏 */}
      <div
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "space-between",
          padding: "0.25rem 0.75rem",
          background: "var(--bg-toolbar)",
          borderBottom: "1px solid var(--border-color)",
          fontSize: "0.8rem",
        }}
      >
        <div style={{ display: "flex", alignItems: "center", gap: "0.5rem" }}>
          <span style={{ color: "var(--text-muted)" }}>终端</span>
          <span
            style={{
              display: "inline-block",
              width: "6px",
              height: "6px",
              borderRadius: "50%",
              background: connected ? "#0dbc79" : "#cd3131",
            }}
            title={connected ? "已连接" : "未连接"}
          />
        </div>
        <button
          onClick={onClose}
          style={{
            background: "none",
            border: "none",
            color: "var(--text-muted)",
            cursor: "pointer",
            fontSize: "1rem",
            lineHeight: 1,
          }}
          title="关闭终端面板"
        >
          ✕
        </button>
      </div>
      {/* xterm.js 终端容器 */}
      <div
        ref={terminalRef}
        style={{
          flex: 1,
          padding: "0 4px",
          overflow: "hidden",
        }}
      />
    </div>
  );
}
