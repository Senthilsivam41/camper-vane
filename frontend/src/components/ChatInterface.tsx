import React, { useState, useEffect } from 'react';
import { useChatSSE } from '../hooks/useChatSSE';
import { OptimizationMetricsPanel } from './OptimizationMetricsPanel';
import type { UserConfig } from '../services/api';
import { fetchUserConfig } from '../services/api';

export const ChatInterface: React.FC = () => {
  const [prompt, setPrompt] = useState<string>('');
  const [userConfig, setUserConfig] = useState<UserConfig | null>(null);

  const {
    messages,
    status,
    currentMetrics,
    finalUsage,
    errorMsg,
    sendMessage,
  } = useChatSSE();

  useEffect(() => {
    fetchUserConfig()
      .then(setUserConfig)
      .catch((err) => console.warn('Failed to load user config:', err));
  }, []);

  const handleSendPrompt = (textToSend: string) => {
    if (!textToSend.trim() || status === 'streaming' || status === 'connecting') return;
    setPrompt('');
    sendMessage(textToSend, 'default-session');
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    handleSendPrompt(prompt);
  };

  const promptStarters = [
    { title: '⚡ Optimize Go Concurrency', text: 'func HandleConcurrency(ch chan int) {\n  // Optimize channel deadlocks\n}' },
    { title: '🔍 Analyze Architecture', text: 'Explain microservice event-driven architecture trade-offs vs monolith' },
    { title: '📝 Refactor SQL Query', text: 'SELECT * FROM users WHERE active = true ORDER BY created_at DESC' },
  ];

  return (
    <div style={styles.chatContainer}>
      {/* Message Feed */}
      <div
        role="log"
        aria-live="polite"
        aria-label="Chat messages history"
        style={styles.messageFeed}
      >
        {messages.length === 0 ? (
          <div style={styles.emptyState}>
            <h2 style={styles.emptyTitle}>Camper Vane AI Gateway</h2>
            <p style={styles.emptySub}>
              Intelligent cost-aware router intercepting queries to dynamically balance token budgets and model capability.
            </p>
            <div style={styles.startersGrid}>
              {promptStarters.map((s) => (
                <button
                  key={s.title}
                  type="button"
                  onClick={() => handleSendPrompt(s.text)}
                  style={styles.starterCard}
                  aria-label={`Prompt starter: ${s.title}`}
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
                <span style={styles.senderRole}>
                  {msg.role === 'user' ? 'You' : 'Camper Vane AI'}
                </span>
                {msg.metrics?.selected_model && (
                  <span style={styles.msgModelBadge} aria-label={`Model: ${msg.metrics.selected_model}`}>
                    {msg.metrics.selected_model}
                  </span>
                )}
              </div>
              <div style={styles.msgContent}>
                {msg.content || (msg.role === 'assistant' && status === 'connecting' ? 'Thinking...' : '')}
              </div>
            </div>
          ))
        )}
      </div>

      {errorMsg && (
        <div style={styles.errorBanner} role="alert">
          ⚠️ {errorMsg}
        </div>
      )}

      {/* Prompt Entry Form */}
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
            disabled={!prompt.trim() || status === 'streaming' || status === 'connecting'}
            style={{
              ...styles.sendButton,
              opacity: !prompt.trim() || status === 'streaming' || status === 'connecting' ? 0.5 : 1,
            }}
            aria-label="Send prompt"
          >
            {status === 'streaming' || status === 'connecting' ? 'Sending...' : 'Send'}
          </button>
        </div>
      </form>

      {/* Persistent Metrics Sub-Panel (Positioned directly underneath prompt entry box) */}
      <OptimizationMetricsPanel
        metrics={currentMetrics}
        finalUsage={finalUsage}
        userConfig={userConfig}
        status={status}
      />
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  chatContainer: {
    display: 'flex',
    flexDirection: 'column',
    maxWidth: '850px',
    margin: '0 auto',
    padding: '16px',
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
    transition: 'all 0.2s ease-in-out',
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
    transition: 'background-color 0.2s ease',
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
