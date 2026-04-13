import React, { useState, useEffect, useCallback } from "react";
import { getSettings, saveSettings } from "../wailsRuntime";

interface SettingsProps {
  visible: boolean;
  onClose: () => void;
}

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

export default function Settings({ visible, onClose }: SettingsProps): React.ReactElement {
  const [form, setForm] = useState<SettingsForm>({ ...defaultForm });
  const [loading, setLoading] = useState(false);
  const [message, setMessage] = useState<string | null>(null);

  useEffect(() => {
    if (visible) {
      loadSettings();
    }
  }, [visible]);

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
      });
      setMessage("设置已保存。API端点/密钥修改需重启生效。");
      setTimeout(() => setMessage(null), 4000);
    } catch (err: any) {
      setMessage("保存失败: " + (err?.message || String(err)));
    }
    setLoading(false);
  }, [form]);

  const handleChange = useCallback((key: keyof SettingsForm, value: string | number) => {
    setForm((prev) => ({ ...prev, [key]: value }));
  }, []);

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

        <label style={labelStyle}>工作目录</label>
        <input
          style={inputStyle}
          type="text"
          value={form.work_dir}
          onChange={(e) => handleChange("work_dir", e.target.value)}
        />

        {message && (
          <div style={{ marginTop: "0.75rem", fontSize: "0.8rem", color: message.startsWith("保存失败") ? "var(--text-red)" : "#4ade80" }}>
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
