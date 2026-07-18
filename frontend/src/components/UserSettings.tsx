import React, { useState, useEffect } from 'react';
import type { UserConfig } from '../services/api';
import { fetchUserConfig, updateUserConfig } from '../services/api';

export const UserSettings: React.FC = () => {
  const [, setConfig] = useState<UserConfig | null>(null);
  const [dailyCap, setDailyCap] = useState<number>(50000);
  const [strategy, setStrategy] = useState<'simple' | 'advanced'>('simple');
  const [preferredModels, setPreferredModels] = useState<string[]>([]);
  const [newModel, setNewModel] = useState<string>('');
  
  const [loading, setLoading] = useState<boolean>(true);
  const [saving, setSaving] = useState<boolean>(false);
  const [saveSuccess, setSaveSuccess] = useState<boolean>(false);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  useEffect(() => {
    loadConfig();
  }, []);

  const loadConfig = async () => {
    try {
      setLoading(true);
      setErrorMsg(null);
      const data = await fetchUserConfig();
      setConfig(data);
      setDailyCap(data.daily_token_cap);
      setStrategy(data.routing_strategy);
      setPreferredModels(data.preferred_models || []);
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to load settings');
    } finally {
      setLoading(false);
    }
  };

  const handleSave = async (e: React.FormEvent) => {
    e.preventDefault();
    if (dailyCap < 0) {
      setErrorMsg('Daily token cap cannot be negative');
      return;
    }

    try {
      setSaving(true);
      setErrorMsg(null);
      setSaveSuccess(false);
      const updated = await updateUserConfig({
        daily_token_cap: Number(dailyCap),
        routing_strategy: strategy,
        preferred_models: preferredModels,
      });
      setConfig(updated);
      setSaveSuccess(true);
      setTimeout(() => setSaveSuccess(false), 3000);
    } catch (err: any) {
      setErrorMsg(err.message || 'Failed to update preferences');
    } finally {
      setSaving(false);
    }
  };

  const addModel = () => {
    if (newModel.trim() && !preferredModels.includes(newModel.trim())) {
      setPreferredModels([...preferredModels, newModel.trim()]);
      setNewModel('');
    }
  };

  const removeModel = (modelToRemove: string) => {
    setPreferredModels(preferredModels.filter((m) => m !== modelToRemove));
  };

  if (loading) {
    return <div style={styles.card}>Loading preferences...</div>;
  }

  return (
    <div style={styles.card}>
      <h2 style={styles.heading}>User Preferences & Routing Settings</h2>
      
      {errorMsg && <div style={styles.errorAlert}>{errorMsg}</div>}
      {saveSuccess && <div style={styles.successBadge}>✓ Preferences updated successfully</div>}

      <form onSubmit={handleSave} style={styles.form}>
        <div style={styles.fieldGroup}>
          <label style={styles.label}>Daily Token Cap:</label>
          <input
            type="number"
            min="0"
            value={dailyCap}
            onChange={(e) => setDailyCap(Number(e.target.value))}
            style={styles.input}
            required
          />
          <small style={styles.hint}>Daily ceiling before Simple Mode enforces low-cost routing.</small>
        </div>

        <div style={styles.fieldGroup}>
          <label style={styles.label}>Optimization Strategy:</label>
          <select
            value={strategy}
            onChange={(e) => setStrategy(e.target.value as 'simple' | 'advanced')}
            style={styles.select}
          >
            <option value="simple">Simple Mode (Volumetric Daily Throttle)</option>
            <option value="advanced">Advanced Mode (Semantic & Contextual Classifier)</option>
          </select>
        </div>

        <div style={styles.fieldGroup}>
          <label style={styles.label}>Preferred Models Order:</label>
          <div style={styles.modelTagList}>
            {preferredModels.map((model) => (
              <span key={model} style={styles.tag}>
                {model}
                <button type="button" onClick={() => removeModel(model)} style={styles.tagRemoveBtn}>
                  ×
                </button>
              </span>
            ))}
          </div>
          <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
            <input
              type="text"
              placeholder="e.g. gemini-1.5-flash"
              value={newModel}
              onChange={(e) => setNewModel(e.target.value)}
              style={{ ...styles.input, flex: 1 }}
            />
            <button type="button" onClick={addModel} style={styles.secBtn}>
              Add Model
            </button>
          </div>
        </div>

        <button type="submit" disabled={saving} style={styles.primaryBtn}>
          {saving ? 'Saving...' : 'Save Configuration'}
        </button>
      </form>
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  card: {
    backgroundColor: '#1e1e2e',
    color: '#cdd6f4',
    borderRadius: '12px',
    padding: '24px',
    maxWidth: '600px',
    margin: '20px auto',
    boxShadow: '0 8px 24px rgba(0,0,0,0.3)',
    fontFamily: 'system-ui, -apple-system, sans-serif',
  },
  heading: {
    marginTop: 0,
    marginBottom: '20px',
    fontSize: '1.4rem',
    color: '#89b4fa',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  fieldGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
  },
  label: {
    fontWeight: 600,
    fontSize: '0.95rem',
  },
  hint: {
    fontSize: '0.8rem',
    color: '#a6adc8',
  },
  input: {
    padding: '10px',
    borderRadius: '6px',
    border: '1px solid #45475a',
    backgroundColor: '#313244',
    color: '#cdd6f4',
    fontSize: '1rem',
  },
  select: {
    padding: '10px',
    borderRadius: '6px',
    border: '1px solid #45475a',
    backgroundColor: '#313244',
    color: '#cdd6f4',
    fontSize: '1rem',
  },
  modelTagList: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: '8px',
  },
  tag: {
    backgroundColor: '#45475a',
    padding: '4px 10px',
    borderRadius: '16px',
    fontSize: '0.85rem',
    display: 'inline-flex',
    alignItems: 'center',
    gap: '6px',
  },
  tagRemoveBtn: {
    background: 'none',
    border: 'none',
    color: '#f38ba8',
    cursor: 'pointer',
    fontWeight: 'bold',
    fontSize: '1rem',
    lineHeight: 1,
  },
  primaryBtn: {
    backgroundColor: '#89b4fa',
    color: '#11111b',
    fontWeight: 'bold',
    padding: '12px',
    border: 'none',
    borderRadius: '6px',
    cursor: 'pointer',
    fontSize: '1rem',
    marginTop: '8px',
  },
  secBtn: {
    backgroundColor: '#45475a',
    color: '#cdd6f4',
    padding: '10px 14px',
    border: 'none',
    borderRadius: '6px',
    cursor: 'pointer',
  },
  successBadge: {
    backgroundColor: '#a6e3a1',
    color: '#11111b',
    padding: '10px',
    borderRadius: '6px',
    marginBottom: '16px',
    fontWeight: 600,
  },
  errorAlert: {
    backgroundColor: '#f38ba8',
    color: '#11111b',
    padding: '10px',
    borderRadius: '6px',
    marginBottom: '16px',
    fontWeight: 600,
  },
};
