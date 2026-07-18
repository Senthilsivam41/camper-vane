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

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!prompt.trim() || status === 'streaming' || status === 'connecting') return;

    const p = prompt;
    setPrompt('');
    sendMessage(p, 'default-session');
  };

  return (
    <div style={styles.chatContainer}>
      {/* Messages Feed */}
      <div style={styles.messageFeed}>
        {messages.length === 0 ? (
          <div style={styles.emptyState}>
            <h3>Ask Camper Vane Anything</h3>
            <p>Prompts are dynamically analyzed for complexity & daily budget constraints to target optimal LLM execution.</p>
          </div>
        ) : (
          messages.map((msg) => (
            <div
              key={msg.id}
              style={{
                ...styles.messageBubble,
                alignSelf: msg.role === 'user' ? 'flex-end' : 'flex-start',
                backgroundColor: msg.role === 'user' ? '#313244' : '#1e1e2e',
                borderColor: msg.role === 'user' ? '#45475a' : '#313244',
              }}
            >
              <div style={styles.msgHeader}>
                <span style={styles.senderRole}>
                  {msg.role === 'user' ? 'You' : 'Camper Vane AI'}
                </span>
                {msg.metrics?.selected_model && (
                  <span style={styles.msgModelBadge}>
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

      {errorMsg && <div style={styles.errorBanner}>⚠️ {errorMsg}</div>}

      {/* Prompt Entry Box */}
      <form onSubmit={handleSubmit} style={styles.inputForm}>
        <div style={styles.inputRow}>
          <textarea
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
          />
          <button
            type="submit"
            disabled={!prompt.trim() || status === 'streaming' || status === 'connecting'}
            style={styles.sendButton}
          >
            {status === 'streaming' || status === 'connecting' ? 'Sending...' : 'Send'}
          </button>
        </div>
      </form>

      {/* Persistent Optimization & Metrics Sub-Panel (Positioned directly underneath prompt box) */}
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
    fontFamily: 'system-ui, -apple-system, sans-serif',
  },
  messageFeed: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column',
    gap: '12px',
    marginBottom: '20px',
    maxHeight: '450px',
    overflowY: 'auto',
    paddingRight: '6px',
  },
  emptyState: {
    textAlign: 'center',
    color: '#a6adc8',
    marginTop: '60px',
  },
  messageBubble: {
    maxWidth: '80%',
    padding: '12px 16px',
    borderRadius: '12px',
    border: '1px solid #313244',
    color: '#cdd6f4',
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
    color: '#89b4fa',
  },
  msgModelBadge: {
    backgroundColor: '#45475a',
    color: '#cdd6f4',
    padding: '2px 8px',
    borderRadius: '8px',
    fontSize: '0.75rem',
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
    backgroundColor: '#1e1e2e',
    color: '#cdd6f4',
    border: '1px solid #45475a',
    borderRadius: '8px',
    padding: '12px',
    fontSize: '0.95rem',
    resize: 'none',
    fontFamily: 'inherit',
  },
  sendButton: {
    backgroundColor: '#89b4fa',
    color: '#11111b',
    fontWeight: 'bold',
    border: 'none',
    borderRadius: '8px',
    padding: '0 24px',
    fontSize: '1rem',
    cursor: 'pointer',
  },
  errorBanner: {
    backgroundColor: '#f38ba8',
    color: '#11111b',
    padding: '10px',
    borderRadius: '6px',
    marginBottom: '10px',
    fontWeight: 'bold',
    fontSize: '0.85rem',
  },
};
