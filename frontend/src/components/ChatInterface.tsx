import React, { useState, useEffect, useCallback } from 'react';
import { useChatSSE } from '../hooks/useChatSSE';
import { OptimizationMetricsPanel } from './OptimizationMetricsPanel';
import type { UserConfig, SessionSummary, UsageInfo } from '../services/api';
import { fetchUserConfig, fetchUsage, fetchSessions } from '../services/api';

function newSessionId(): string {
  return `sess-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

export const ChatInterface: React.FC = () => {
  const [prompt, setPrompt] = useState<string>('');
  const [userConfig, setUserConfig] = useState<UserConfig | null>(null);
  const [usage, setUsage] = useState<UsageInfo | null>(null);
  const [sessions, setSessions] = useState<SessionSummary[]>([]);
  const [sessionID, setSessionID] = useState<string>(() => newSessionId());
  const [sessionBusy, setSessionBusy] = useState(false);

  const {
    messages,
    status,
    currentMetrics,
    finalUsage,
    errorMsg,
    retryCount,
    sendMessage,
    loadSessionHistory,
    clearMessages,
  } = useChatSSE();

  const refreshMeta = useCallback(async () => {
    try {
      const [cfg, usageInfo, sess] = await Promise.all([
        fetchUserConfig(),
        fetchUsage(),
        fetchSessions(30),
      ]);
      setUserConfig(cfg);
      setUsage(usageInfo);
      setSessions(sess);
    } catch (err) {
      console.warn('Failed to refresh chat metadata:', err);
    }
  }, []);

  useEffect(() => {
    refreshMeta();
  }, [refreshMeta]);

  useEffect(() => {
    if (finalUsage) {
      setUsage((prev) =>
        prev
          ? {
              ...prev,
              tokens_used: finalUsage.updated_daily_total,
              utilization_pct: prev.daily_token_cap
                ? (finalUsage.updated_daily_total / prev.daily_token_cap) * 100
                : 0,
            }
          : prev
      );
      fetchSessions(30).then(setSessions).catch(() => undefined);
    }
  }, [finalUsage]);

  const handleSendPrompt = (textToSend: string) => {
    if (!textToSend.trim() || status === 'streaming' || status === 'connecting' || status === 'retrying') {
      return;
    }
    setPrompt('');
    sendMessage(textToSend, sessionID);
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    handleSendPrompt(prompt);
  };

  const handleNewSession = () => {
    setSessionID(newSessionId());
    clearMessages();
  };

  const handleOpenSession = async (id: string) => {
    if (id === sessionID && messages.length > 0) return;
    setSessionBusy(true);
    try {
      setSessionID(id);
      await loadSessionHistory(id);
    } catch (err: any) {
      console.error(err);
    } finally {
      setSessionBusy(false);
    }
  };

  const promptStarters = [
    { title: 'Optimize Go Concurrency', text: 'func HandleConcurrency(ch chan int) {\n  // Optimize channel deadlocks\n}' },
    { title: 'Analyze Architecture', text: 'Explain microservice event-driven architecture trade-offs vs monolith' },
    { title: 'Refactor SQL Query', text: 'SELECT * FROM users WHERE active = true ORDER BY created_at DESC' },
  ];

  const busy = status === 'streaming' || status === 'connecting' || status === 'retrying' || sessionBusy;

  return (
    <div style={styles.layout}>
      <aside style={styles.sidebar} aria-label="Chat sessions">
        <div style={styles.sidebarHeader}>
          <strong>Sessions</strong>
          <button type="button" onClick={handleNewSession} style={styles.newSessionBtn} aria-label="Start new session">
            New
          </button>
        </div>
        <div style={styles.sessionList}>
          <button
            type="button"
            onClick={() => handleOpenSession(sessionID)}
            style={{ ...styles.sessionItem, ...styles.sessionItemActive }}
            aria-current="true"
          >
            <span style={styles.sessionTitle}>Current session</span>
            <span style={styles.sessionMeta}>{sessionID}</span>
          </button>
          {sessions
            .filter((s) => s.session_id !== sessionID)
            .map((s) => (
              <button
                key={s.session_id}
                type="button"
                onClick={() => handleOpenSession(s.session_id)}
                style={styles.sessionItem}
              >
                <span style={styles.sessionTitle}>{s.preview || s.session_id}</span>
                <span style={styles.sessionMeta}>
                  {s.message_count} msgs · {new Date(s.updated_at).toLocaleString()}
                </span>
              </button>
            ))}
        </div>
      </aside>

      <div style={styles.chatContainer}>
        <div role="log" aria-live="polite" aria-label="Chat messages history" style={styles.messageFeed}>
          {messages.length === 0 ? (
            <div style={styles.emptyState}>
              <h2 style={styles.emptyTitle}>Camper Vane AI Gateway</h2>
              <p style={styles.emptySub}>
                Cost-aware router balances token budgets and model capability. Sessions persist server-side.
              </p>
              <div style={styles.startersGrid}>
                {promptStarters.map((s) => (
                  <button
                    key={s.title}
                    type="button"
                    onClick={() => handleSendPrompt(s.text)}
                    style={styles.starterCard}
                    aria-label={`Prompt starter: ${s.title}`}
                    disabled={busy}
                  >
                    <strong style={styles.starterTitle}>{s.title}</strong>
                    <span style={styles.starterText}>{s.text}</span>
                  </button>
                ))}
              </div>
            </div>
          ) : (
            messages.map((msg) => (
              <div
                key={msg.id}
                style={{
                  ...styles.messageBubble,
                  alignSelf: msg.role === 'user' ? 'flex-end' : 'flex-start',
                  backgroundColor: msg.role === 'user' ? 'var(--bg-card-hover)' : 'var(--bg-card)',
                  borderColor: msg.role === 'user' ? 'var(--border-focus)' : 'var(--border-color)',
                }}
              >
                <div style={styles.msgHeader}>
                  <span style={styles.senderRole}>{msg.role === 'user' ? 'You' : 'Camper Vane AI'}</span>
                  {msg.metrics?.selected_model && (
                    <span style={styles.msgModelBadge} aria-label={`Model: ${msg.metrics.selected_model}`}>
                      {msg.metrics.selected_model}
                    </span>
                  )}
                </div>
                <div style={styles.msgContent}>
                  {msg.content ||
                    (msg.role === 'assistant' && (status === 'connecting' || status === 'retrying')
                      ? status === 'retrying'
                        ? `Retrying (${retryCount})...`
                        : 'Thinking...'
                      : '')}
                </div>
              </div>
            ))
          )}
        </div>

        {errorMsg && (
          <div style={styles.errorBanner} role="alert">
            {errorMsg}
          </div>
        )}

        <form onSubmit={handleSubmit} style={styles.inputForm} aria-label="Prompt entry form">
          <label htmlFor="prompt-textarea" className="visually-hidden">
            Prompt message input
          </label>
          <div style={styles.inputRow}>
            <textarea
              id="prompt-textarea"
              value={prompt}
              onChange={(e) => setPrompt(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === 'Enter' && !e.shiftKey) {
                  e.preventDefault();
                  handleSubmit(e);
                }
              }}
              placeholder="Type your prompt here... (Shift+Enter for newline)"
              rows={2}
              style={styles.textarea}
              aria-label="Type prompt here"
            />
            <button
              type="submit"
              disabled={!prompt.trim() || busy}
              style={{
                ...styles.sendButton,
                opacity: !prompt.trim() || busy ? 0.5 : 1,
              }}
              aria-label="Send prompt"
            >
              {busy ? 'Sending...' : 'Send'}
            </button>
          </div>
        </form>

        <OptimizationMetricsPanel
          metrics={currentMetrics}
          finalUsage={finalUsage}
          userConfig={userConfig}
          usage={usage}
          status={status}
          retryCount={retryCount}
        />
      </div>
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  layout: {
    display: 'grid',
    gridTemplateColumns: '220px minmax(0, 1fr)',
    gap: '16px',
    maxWidth: '1100px',
    margin: '0 auto',
    padding: '16px',
    alignItems: 'start',
  },
  sidebar: {
    backgroundColor: 'var(--bg-card)',
    border: '1px solid var(--border-color)',
    borderRadius: '12px',
    padding: '12px',
    minHeight: '70vh',
  },
  sidebarHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '10px',
    color: 'var(--text-main)',
  },
  newSessionBtn: {
    backgroundColor: 'var(--primary)',
    color: '#0f172a',
    border: 'none',
    borderRadius: '6px',
    padding: '4px 10px',
    fontWeight: 700,
    cursor: 'pointer',
    fontSize: '0.8rem',
  },
  sessionList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
    maxHeight: '65vh',
    overflowY: 'auto',
  },
  sessionItem: {
    textAlign: 'left',
    backgroundColor: 'transparent',
    border: '1px solid var(--border-color)',
    borderRadius: '8px',
    padding: '8px 10px',
    cursor: 'pointer',
    color: 'var(--text-main)',
    display: 'flex',
    flexDirection: 'column',
    gap: '2px',
  },
  sessionItemActive: {
    borderColor: 'var(--primary)',
    backgroundColor: 'var(--bg-card-hover)',
  },
  sessionTitle: {
    fontSize: '0.8rem',
    fontWeight: 600,
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
  },
  sessionMeta: {
    fontSize: '0.7rem',
    color: 'var(--text-muted)',
  },
  chatContainer: {
    display: 'flex',
    flexDirection: 'column',
    minHeight: '75vh',
  },
  messageFeed: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    gap: '14px',
    marginBottom: '16px',
    maxHeight: '440px',
    overflowY: 'auto',
    paddingRight: '6px',
  },
  emptyState: {
    textAlign: 'center',
    marginTop: '40px',
  },
  emptyTitle: {
    color: 'var(--primary)',
    margin: '0 0 8px 0',
    fontSize: '1.6rem',
  },
  emptySub: {
    color: 'var(--text-muted)',
    maxWidth: '520px',
    margin: '0 auto 24px auto',
    fontSize: '0.95rem',
  },
  startersGrid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(220px, 1fr))',
    gap: '12px',
    marginTop: '16px',
  },
  starterCard: {
    backgroundColor: 'var(--bg-card)',
    border: '1px solid var(--border-color)',
    borderRadius: '10px',
    padding: '14px',
    textAlign: 'left',
    color: 'var(--text-main)',
    cursor: 'pointer',
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
  },
  starterTitle: {
    color: 'var(--primary)',
    fontSize: '0.9rem',
  },
  starterText: {
    color: 'var(--text-muted)',
    fontSize: '0.8rem',
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
  },
  messageBubble: {
    maxWidth: '82%',
    padding: '12px 16px',
    borderRadius: '12px',
    border: '1px solid',
    color: 'var(--text-main)',
    lineHeight: 1.5,
  },
  msgHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '6px',
    fontSize: '0.8rem',
  },
  senderRole: {
    fontWeight: 'bold',
    color: 'var(--primary)',
  },
  msgModelBadge: {
    backgroundColor: 'var(--bg-app)',
    color: 'var(--text-muted)',
    padding: '2px 8px',
    borderRadius: '8px',
    fontSize: '0.75rem',
    border: '1px solid var(--border-color)',
  },
  msgContent: {
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
  },
  inputForm: {
    marginTop: 'auto',
  },
  inputRow: {
    display: 'flex',
    gap: '10px',
  },
  textarea: {
    flex: 1,
    backgroundColor: 'var(--bg-card)',
    color: 'var(--text-main)',
    border: '1px solid var(--border-color)',
    borderRadius: '10px',
    padding: '12px',
    fontSize: '0.95rem',
    resize: 'none',
    fontFamily: 'inherit',
  },
  sendButton: {
    backgroundColor: 'var(--primary)',
    color: '#0f172a',
    fontWeight: 'bold',
    border: 'none',
    borderRadius: '10px',
    padding: '0 24px',
    fontSize: '1rem',
    cursor: 'pointer',
  },
  errorBanner: {
    backgroundColor: 'rgba(248, 113, 113, 0.15)',
    color: 'var(--danger)',
    border: '1px solid rgba(248, 113, 113, 0.3)',
    padding: '12px',
    borderRadius: '8px',
    marginBottom: '12px',
    fontWeight: 'bold',
    fontSize: '0.85rem',
  },
};
