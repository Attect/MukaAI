import React, { useState, useEffect, useCallback } from "react";
import { getSettings, saveSettings, onEvent, chooseDirectory } from "../wailsRuntime";

interface SettingsForm {
  endpoint: string;
  api_key: string;
  model_name: string;
  context_size: number;
  temperature: number;
  max_iterations: number;
  work_dir: string;
}

const defaultForm: SettingsForm = {
  endpoint: "http://127.0.0.1:11453/v1/",
  api_key: "no-key",
  model_name: "",
  context_size: 200000,
  temperature: 0.7,
  max_iterations: 100,
  work_dir: ".",
};

export interface SettingsProps {
  visible: boolean;
  onClose: () => void;
}

export default function Settings({ visible, onClose }: SettingsProps): React.ReactElement {
  const [form, setForm] = useState<SettingsForm>({ ...defaultForm });
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<string | null>(null);
  const [messageType, setMessageType] = useState<"success" | "warning" | "error">("success");
  const [activeTab, setActiveTab] = useState<"model" | "mcp">("model");

  // MCP相关状态
  const [mcpEnabled, setMcpEnabled] = useState(true);
  const [mcpServers, setMcpServers] = useState<any[]>([]);

  useEffect(() => {
    if (visible) {
      loadSettings();
    }
  }, [visible]);

  // 监听热更新警告事件
  useEffect(() => {
    const unsub = onEvent("settings:hot-update-warning", (warning: string) => {
      setMessage(warning);
      setMessageType("warning");
      setTimeout(() => setMessage(null), 6000);
    });
    return () => {
      if (typeof unsub === "function") unsub();
    };
  }, []);

  const loadSettings = useCallback(async () => {
    setLoading(true);
    try {
      const result = await getSettings();
      if (result) {
        setForm({
          endpoint: String(result.endpoint || defaultForm.endpoint),
          api_key: String(result.api_key || defaultForm.api_key),
          model_name: String(result.model_name || defaultForm.model_name),
          context_size: Number(result.context_size || defaultForm.context_size),
          temperature: Number(result.temperature ?? defaultForm.temperature),
          max_iterations: Number(result.max_iterations || defaultForm.max_iterations),
          work_dir: String(result.work_dir || defaultForm.work_dir),
        });
        // 加载MCP配置
        setMcpEnabled(result.mcp_enabled !== undefined ? Boolean(result.mcp_enabled) : true);
        if (result.mcp_servers && Array.isArray(result.mcp_servers)) {
          setMcpServers(result.mcp_servers.map((s: any) => ({
            ...s,
            enabled: s.enabled !== undefined ? Boolean(s.enabled) : true,
            prefix: s.prefix || "",
            tools: s.tools || {},
          })));
        }
      }
    } catch (err) {
      console.error("Failed to load settings:", err);
    }
    setLoading(false);
  }, []);

  const handleSave = useCallback(async () => {
    setLoading(true);
    setMessage(null);
    try {
      await saveSettings({
        endpoint: form.endpoint,
        api_key: form.api_key,
        model_name: form.model_name,
        context_size: form.context_size,
        temperature: form.temperature,
        max_iterations: form.max_iterations,
        work_dir: form.work_dir,
        mcp_enabled: mcpEnabled,
        mcp_servers: mcpServers,
      });
      setMessage("设置已保存。模型名称和上下文大小已即时生效；API端点/密钥修改需重启生效。");
      setMessageType("success");
      setTimeout(() => setMessage(null), 4000);
    } catch (err: any) {
      setMessage("保存失败: " + (err?.message || String(err)));
      setMessageType("error");
    }
    setLoading(false);
  }, [form, mcpEnabled, mcpServers]);

  const handleChange = useCallback((key: keyof SettingsForm, value: string | number) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  }, []);

  const handleBrowseWorkDir = useCallback(async () => {
    try {
      const chosen = await chooseDirectory();
      if (chosen) {
        setForm((prev) => ({ ...prev, work_dir: chosen }));
      }
    } catch (err: any) {
      console.error("Failed to choose directory:", err);
    }
  }, []);

  // MCP服务器操作
  const updateServer = useCallback((index: number, updates: Partial<any>) => {
    setMcpServers((prev) => {
      const next = [...prev];
      next[index] = { ...next[index], ...updates };
      return next;
    });
  }, []);

  const toggleServerEnabled = useCallback((index: number) => {
    updateServer(index, { enabled: !mcpServers[index].enabled });
  }, [mcpServers, updateServer]);

  const updateServerPrefix = useCallback((index: number, prefix: string) => {
    updateServer(index, { prefix });
  }, [updateServer]);

  const updateToolSetting = useCallback((serverIndex: number, toolName: string, updates: Partial<any>) => {
    setMcpServers((prev) => {
      const next = [...prev];
      const server = { ...next[serverIndex] };
      const tools = { ...server.tools };
      if (!tools[toolName]) {
        tools[toolName] = { enabled: true, description: "" };
      }
      tools[toolName] = { ...tools[toolName], ...updates };
      server.tools = tools;
      next[serverIndex] = server;
      return next;
    });
  }, []);

  const toggleToolEnabled = useCallback((serverIndex: number, toolName: string) => {
    const server = mcpServers[serverIndex];
    const toolSetting = server.tools?.[toolName] || { enabled: true, description: "" };
    updateToolSetting(serverIndex, toolName, { enabled: !toolSetting.enabled });
  }, [mcpServers, updateToolSetting]);

  if (!visible) return <></>;

  const inputStyle = {
    width: "100%",
    background: "var(--bg-input)",
    color: "var(--text-primary)",
    border: "1px solid var(--border-color)",
    borderRadius: "0.375rem",
    padding: "0.5rem 0.75rem",
    fontSize: "0.875rem",
    outline: "none",
  };

  const labelStyle = {
    display: "block" as const,
    fontSize: "0.75rem",
    fontWeight: 600 as const,
    color: "var(--text-muted)",
    marginBottom: "0.25rem",
    marginTop: "0.75rem",
  };

  // Tab样式
  const tabStyle = (isActive: boolean) => ({
    padding: "0.5rem 1rem",
    fontSize: "0.875rem",
    fontWeight: isActive ? 600 : 400,
    borderRadius: "0.375rem",
    border: "none",
    background: isActive ? "var(--bg-button)" : "transparent",
    color: isActive ? "#fff" : "var(--text-muted)",
    cursor: "pointer",
  });

  // 渲染MCP标签页内容
  const renderMCPTab = () => (
    <div>
      {/* MCP总开关 */}
      <div style={{ display: "flex", alignItems: "center", gap: "0.5rem", marginBottom: "1rem" }}>
        <label style={{ ...labelStyle, marginTop: 0, marginBottom: 0 }}>启用 MCP</label>
        <button
          onClick={() => setMcpEnabled(!mcpEnabled)}
          style={{
            padding: "0.25rem 0.75rem",
            fontSize: "0.8125rem",
            borderRadius: "0.375rem",
            border: "1px solid var(--border-color)",
            background: mcpEnabled ? "var(--bg-button)" : "var(--bg-input)",
            color: mcpEnabled ? "#fff" : "var(--text-muted)",
            cursor: "pointer",
          }}
        >
          {mcpEnabled ? "已启用" : "已禁用"}
        </button>
      </div>

      {/* 服务器列表 */}
      <div style={{ marginTop: "1rem" }}>
        <label style={labelStyle}>MCP 服务器配置</label>
        {mcpServers.length === 0 && (
          <div style={{ padding: "0.75rem", fontSize: "0.8125rem", color: "var(--text-muted)", background: "var(--bg-input)", borderRadius: "0.375rem" }}>
            暂无MCP服务器配置。配置文件中的服务器将在此显示。
          </div>
        )}
        {mcpServers.map((server, index) => (
          <div key={server.id || `server-${index}`} style={{
            border: "1px solid var(--border-color)",
            borderRadius: "0.375rem",
            padding: "0.75rem",
            marginBottom: "0.75rem",
            background: "var(--bg-input)",
          }}>
            {/* 服务器头部 */}
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "0.5rem" }}>
              <span style={{ fontWeight: 600, fontSize: "0.875rem" }}>
                {server.id || `服务器 ${index + 1}`}
              </span>
              <button
                onClick={() => toggleServerEnabled(index)}
                style={{
                  padding: "0.25rem 0.5rem",
                  fontSize: "0.75rem",
                  borderRadius: "0.25rem",
                  border: "1px solid var(--border-color)",
                  background: server.enabled ? "var(--bg-button)" : "var(--bg-input)",
                  color: server.enabled ? "#fff" : "var(--text-muted)",
                  cursor: "pointer",
                }}
              >
                {server.enabled ? "启用" : "禁用"}
              </button>
            </div>

            {/* 前缀配置 */}
            <div style={{ marginBottom: "0.5rem" }}>
              <label style={{ ...labelStyle, marginTop: 0 }}>ID（只读）</label>
              <input
                style={{ ...inputStyle, background: "var(--bg-disabled)", color: "var(--text-muted)" }}
                type="text"
                value={server.id || ""}
                readOnly
              />
            </div>

            <div style={{ marginBottom: "0.5rem" }}>
              <label style={labelStyle}>工具名前缀</label>
              <input
                style={inputStyle}
                type="text"
                value={server.prefix || ""}
                onChange={(e) => updateServerPrefix(index, e.target.value)}
                placeholder="默认使用ID作为前缀"
              />
            </div>

            {/* 工具列表 */}
            <div style={{ marginTop: "0.75rem" }}>
              <label style={labelStyle}>已发现工具</label>
              {server.tools && Object.keys(server.tools).length > 0 ? (
                <div style={{ maxHeight: "200px", overflowY: "auto" }}>
                  {Object.entries(server.tools).map(([toolName, toolConfig]: [string, any]) => (
                    <div key={toolName} style={{
                      display: "flex",
                      alignItems: "flex-start",
                      gap: "0.5rem",
                      padding: "0.375rem 0",
                      borderBottom: "1px solid var(--border-color)",
                    }}>
                      <button
                        onClick={() => toggleToolEnabled(index, toolName)}
                        style={{
                          padding: "0.125rem 0.375rem",
                          fontSize: "0.6875rem",
                          borderRadius: "0.25rem",
                          border: "1px solid var(--border-color)",
                          background: toolConfig.enabled ? "var(--bg-button)" : "var(--bg-input)",
                          color: toolConfig.enabled ? "#fff" : "var(--text-muted)",
                          cursor: "pointer",
                          whiteSpace: "nowrap",
                          flexShrink: 0,
                        }}
                      >
                        {toolConfig.enabled ? "启用" : "禁用"}
                      </button>
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontSize: "0.8125rem", fontWeight: 500, marginBottom: "0.125rem" }}>
                          {toolName}
                        </div>
                        <input
                          style={{ ...inputStyle, fontSize: "0.75rem", padding: "0.25rem 0.5rem" }}
                          type="text"
                          value={toolConfig.description || ""}
                          onChange={(e) => updateToolSetting(index, toolName, { description: e.target.value })}
                          placeholder="自定义工具描述（可选）"
                        />
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div style={{ padding: "0.5rem", fontSize: "0.75rem", color: "var(--text-muted)" }}>
                  服务器连接后将显示已发现的工具。
                </div>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );

  return (
    <div className="modal-overlay" onClick={(e) => { if (e.target === e.currentTarget) onClose(); }}>
      <div className="modal-content">
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: "1rem" }}>
          <h2 style={{ fontSize: "1.125rem", fontWeight: 700, color: "var(--text-primary)", margin: 0 }}>设置</h2>
          <button
            onClick={onClose}
            style={{ background: "none", border: "none", color: "var(--text-muted)", cursor: "pointer", fontSize: "1.25rem" }}
          >
            ✕
          </button>
        </div>

        {/* Tab切换 */}
        <div style={{ display: "flex", gap: "0.5rem", marginBottom: "1rem" }}>
          <button onClick={() => setActiveTab("model")} style={tabStyle(activeTab === "model")}>
            模型
          </button>
          <button onClick={() => setActiveTab("mcp")} style={tabStyle(activeTab === "mcp")}>
            MCP
          </button>
        </div>

        {/* Model Tab */}
        {activeTab === "model" && (
          <>
            <label style={labelStyle}>API 端点</label>
            <input
              style={inputStyle}
              type="url"
              value={form.endpoint}
              onChange={(e) => handleChange("endpoint", e.target.value)}
              placeholder="http://127.0.0.1:11453/v1/"
            />

            <label style={labelStyle}>API 密钥</label>
            <input
              style={inputStyle}
              type="password"
              value={form.api_key}
              onChange={(e) => handleChange("api_key", e.target.value)}
              placeholder="no-key"
            />

            <label style={labelStyle}>模型名称</label>
            <input
              style={inputStyle}
              type="text"
              value={form.model_name}
              onChange={(e) => handleChange("model_name", e.target.value)}
              placeholder="模型名称"
            />

            <label style={labelStyle}>上下文大小</label>
            <input
              style={inputStyle}
              type="number"
              value={form.context_size}
              onChange={(e) => handleChange("context_size", Number(e.target.value))}
            />

            <label style={labelStyle}>温度 ({form.temperature.toFixed(1)})</label>
            <input
              style={{ ...inputStyle, padding: "0.25rem" }}
              type="range"
              min="0"
              max="2"
              step="0.1"
              value={form.temperature}
              onChange={(e) => handleChange("temperature", Number(e.target.value))}
            />

            <label style={labelStyle}>最大迭代次数</label>
            <input
              style={inputStyle}
              type="number"
              value={form.max_iterations}
              onChange={(e) => handleChange("max_iterations", Number(e.target.value))}
            />

            <div style={{ display: "flex", gap: "0.5rem", alignItems: "center" }}>
              <label style={labelStyle}>工作目录</label>
              <button
                onClick={handleBrowseWorkDir}
                style={{
                  padding: "0.375rem 0.625rem",
                  fontSize: "0.8125rem",
                  borderRadius: "0.375rem",
                  border: "1px solid var(--border-color)",
                  background: "var(--bg-input)",
                  color: "var(--text-muted)",
                  cursor: "pointer",
                  whiteSpace: "nowrap",
                }}
                title="选择目录"
              >
                浏览...
              </button>
            </div>
            <input
              style={inputStyle}
              type="text"
              value={form.work_dir}
              onChange={(e) => handleChange("work_dir", e.target.value)}
            />
          </>
        )}

        {/* MCP Tab */}
        {activeTab === "mcp" && renderMCPTab()}

        {message && (
          <div style={{
            marginTop: "0.75rem",
            fontSize: "0.8rem",
            color: messageType === "error" ? "var(--text-red)" : messageType === "warning" ? "#fbbf24" : "#4ade80",
          }}>
            {message}
          </div>
        )}

        <div style={{ display: "flex", justifyContent: "flex-end", gap: "0.5rem", marginTop: "1.5rem" }}>
          <button
            onClick={onClose}
            style={{
              padding: "0.5rem 1rem",
              fontSize: "0.875rem",
              borderRadius: "0.375rem",
              border: "1px solid var(--border-color)",
              background: "transparent",
              color: "var(--text-muted)",
              cursor: "pointer",
            }}
          >
            关闭
          </button>
          <button
            onClick={handleSave}
            disabled={loading}
            style={{
              padding: "0.5rem 1rem",
              fontSize: "0.875rem",
              borderRadius: "0.375rem",
              border: "none",
              background: "var(--bg-button)",
              color: "#fff",
              cursor: loading ? "not-allowed" : "pointer",
              opacity: loading ? 0.6 : 1,
            }}
          >
            {loading ? "保存中..." : "保存"}
          </button>
        </div>
      </div>
    </div>
  );
}
