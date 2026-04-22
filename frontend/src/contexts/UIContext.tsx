import React, { createContext, useContext, useState, useEffect, useCallback, useRef } from "react";

export interface UISettings {
  defaultExpandThinking: boolean;
  defaultExpandToolCalls: boolean;
}

const STORAGE_KEY = "mukaui-ui-settings";

function loadUISettings(): UISettings {
  try {
    const saved = localStorage.getItem(STORAGE_KEY);
    if (saved) {
      const parsed = JSON.parse(saved);
      return {
        defaultExpandThinking: parsed.defaultExpandThinking !== undefined ? Boolean(parsed.defaultExpandThinking) : true,
        defaultExpandToolCalls: parsed.defaultExpandToolCalls !== undefined ? Boolean(parsed.defaultExpandToolCalls) : true,
      };
    }
  } catch {
    // ignore parse errors
  }
  return {
    defaultExpandThinking: true,
    defaultExpandToolCalls: true,
  };
}

function saveUISettings(settings: UISettings): void {
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(settings));
  } catch {
    // ignore storage errors
  }
}

interface UIContextType {
  settings: UISettings;
  setSetting: <K extends keyof UISettings>(key: K, value: boolean) => void;
}

const UIContext = createContext<UIContextType | null>(null);

export function UIProvider({ children }: { children: React.ReactNode }): React.ReactElement {
  const [settings, setSettings] = useState<UISettings>(loadUISettings);
  const settingsRef = useRef<UISettings>(settings);

  // 同步到 ref
  useEffect(() => {
    settingsRef.current = settings;
  }, [settings]);

  // 持久化到 localStorage
  useEffect(() => {
    saveUISettings(settings);
  }, [settings]);

  const setSetting = useCallback(<K extends keyof UISettings>(key: K, value: boolean) => {
    setSettings((prev) => ({ ...prev, [key]: value }));
  }, []);

  return (
    <UIContext.Provider value={{ settings, setSetting }}>
      {children}
    </UIContext.Provider>
  );
}

export function useUI(): UIContextType {
  const ctx = useContext(UIContext);
  if (!ctx) throw new Error("useUI must be used within UIProvider");
  return ctx;
}
