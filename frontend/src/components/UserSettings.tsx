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
    return (
      <div style={styles.card} aria-busy="true" aria-live="polite">
        Loading preferences...
      </div>
    );
  }

  return (
    <div style={styles.card}>
      <h2 style={styles.heading}>User Preferences & Routing Settings</h2>

      {errorMsg && (
        <div style={styles.errorAlert} role="alert">
          ⚠️ {errorMsg}
        </div>
      )}
      {saveSuccess && (
        <div style={styles.successBadge} role="status" aria-live="polite">
          ✓ Preferences updated successfully
        </div>
      )}

      <form onSubmit={handleSave} style={styles.form} aria-label="User Preferences Form">
        <div style={styles.fieldGroup}>
          <label htmlFor="daily-cap-input" style={styles.label}>
            Daily Token Cap:
          </label>
          <input
            id="daily-cap-input"
            type="number"
            min="0"
            value={dailyCap}
            onChange={(e) => setDailyCap(Number(e.target.value))}
            style={styles.input}
            required
            aria-describedby="daily-cap-hint"
          />
          <small id="daily-cap-hint" style={styles.hint}>
            Daily volumetric ceiling before Simple Mode enforces low-cost routing (≥85% threshold).
          </small>
        </div>

        <div style={styles.fieldGroup}>
          <label htmlFor="strategy-select" style={styles.label}>
            Optimization Strategy:
          </label>
          <select
            id="strategy-select"
            value={strategy}
            onChange={(e) => setStrategy(e.target.value as 'simple' | 'advanced')}
            style={styles.select}
          >
            <option value="simple">Simple Mode (Volumetric Daily Throttle)</option>
            <option value="advanced">Advanced Mode (Semantic & Contextual Classifier)</option>
          </select>
        </div>

        <div style={styles.fieldGroup}>
          <label id="preferred-models-label" style={styles.label}>
            Preferred Models Order:
          </label>
          <div style={styles.modelTagList} aria-labelledby="preferred-models-label">
            {preferredModels.map((model) => (
              <span key={model} style={styles.tag}>
                {model}
                <button
                  type="button"
                  onClick={() => removeModel(model)}
                  style={styles.tagRemoveBtn}
                  aria-label={`Remove model ${model}`}
                >
                  ×
                </button>
              </span>
            ))}
          </div>
          <div style={{ display: 'flex', gap: '8px', marginTop: '8px' }}>
            <label htmlFor="add-model-input" className="visually-hidden">
              Add new model name
            </label>
            <input
              id="add-model-input"
              type="text"
              placeholder="e.g. gemini-1.5-flash"
              value={newModel}
              onChange={(e) => setNewModel(e.target.value)}
              style={{ ...styles.input, flex: 1 }}
            />
            <button
              type="button"
              onClick={addModel}
              style={styles.secBtn}
              aria-label="Add model to list"
            >
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
    backgroundColor: 'var(--bg-card)',
    color: 'var(--text-main)',
    borderRadius: '12px',
    padding: '24px',
    maxWidth: '600px',
    margin: '20px auto',
    border: '1px solid var(--border-color)',
  },
  heading: {
    marginTop: 0,
    marginBottom: '20px',
    fontSize: '1.4rem',
    color: 'var(--primary)',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '18px',
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
    color: 'var(--text-muted)',
  },
  input: {
    padding: '10px 12px',
    borderRadius: '8px',
    border: '1px solid var(--border-color)',
    backgroundColor: 'var(--bg-app)',
    color: 'var(--text-main)',
    fontSize: '1rem',
  },
  select: {
    padding: '10px 12px',
    borderRadius: '8px',
    border: '1px solid var(--border-color)',
    backgroundColor: 'var(--bg-app)',
    color: 'var(--text-main)',
    fontSize: '1rem',
  },
  modelTagList: {
    display: 'flex',
    flexWrap: 'wrap',
    gap: '8px',
  },
  tag: {
    backgroundColor: 'var(--bg-card-hover)',
    color: 'var(--text-main)',
    padding: '4px 10px',
    borderRadius: '16px',
    fontSize: '0.85rem',
    display: 'inline-flex',
    alignItems: 'center',
    gap: '6px',
    border: '1px solid var(--border-color)',
  },
  tagRemoveBtn: {
    background: 'none',
    border: 'none',
    color: 'var(--danger)',
    cursor: 'pointer',
    fontWeight: 'bold',
    fontSize: '1rem',
    lineHeight: 1,
  },
  primaryBtn: {
    backgroundColor: 'var(--primary)',
    color: '#0f172a',
    fontWeight: 'bold',
    padding: '12px',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
    fontSize: '1rem',
    marginTop: '8px',
  },
  secBtn: {
    backgroundColor: 'var(--bg-card-hover)',
    color: 'var(--text-main)',
    padding: '10px 14px',
    border: '1px solid var(--border-color)',
    borderRadius: '8px',
    cursor: 'pointer',
  },
  successBadge: {
    backgroundColor: 'rgba(74, 222, 128, 0.15)',
    color: 'var(--success)',
    border: '1px solid rgba(74, 222, 128, 0.3)',
    padding: '12px',
    borderRadius: '8px',
    marginBottom: '16px',
    fontWeight: 600,
  },
  errorAlert: {
    backgroundColor: 'rgba(248, 113, 113, 0.15)',
    color: 'var(--danger)',
    border: '1px solid rgba(248, 113, 113, 0.3)',
    padding: '12px',
    borderRadius: '8px',
    marginBottom: '16px',
    fontWeight: 600,
  },
};
