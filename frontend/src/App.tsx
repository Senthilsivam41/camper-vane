import { useState, useEffect } from 'react';
import { ChatInterface } from './components/ChatInterface';
import { UserSettings } from './components/UserSettings';
import { handleCallback, checkMe } from './services/api';

export function App() {
  const [authed, setAuthed] = useState<boolean>(false);
  const [loggingIn, setLoggingIn] = useState<boolean>(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'chat' | 'settings'>('chat');

  useEffect(() => {
    // Auto-check session on load
    checkMe()
      .then(() => setAuthed(true))
      .catch(() => setAuthed(false));
  }, []);

  const triggerMockLogin = async () => {
    try {
      setLoggingIn(true);
      setAuthError(null);
      await handleCallback('mock_auth_code', 'developer-1');
      setAuthed(true);
    } catch (err: any) {
      console.error('Login error:', err);
      setAuthError(
        err.message || 'Failed to connect to backend server on http://localhost:8080. Make sure `go run ./cmd/server/main.go` is running.'
      );
    } finally {
      setLoggingIn(false);
    }
  };

  return (
    <div style={{ backgroundColor: 'var(--bg-app)', minHeight: '100vh', padding: '20px', color: 'var(--text-main)' }}>
      <header style={styles.header}>
        <div style={{ textAlign: 'center' }}>
          <h1 style={{ color: 'var(--purple)', margin: 0, fontSize: '1.8rem', letterSpacing: '-0.5px' }}>
            Camper Vane
          </h1>
          <p style={{ color: 'var(--text-muted)', margin: '4px 0 0 0', fontSize: '0.9rem' }}>
            Cost-Aware Dynamic LLM Gateway & Real-Time Optimization Dashboard
          </p>
        </div>

        {authed && (
          <nav style={styles.navTabs} role="tablist" aria-label="Main Navigation">
            <button
              id="tab-chat"
              role="tab"
              aria-selected={activeTab === 'chat'}
              aria-controls="panel-chat"
              onClick={() => setActiveTab('chat')}
              style={{
                ...styles.tabBtn,
                ...(activeTab === 'chat' ? styles.activeTab : {}),
              }}
            >
              💬 Chat & Real-Time Metrics
            </button>
            <button
              id="tab-settings"
              role="tab"
              aria-selected={activeTab === 'settings'}
              aria-controls="panel-settings"
              onClick={() => setActiveTab('settings')}
              style={{
                ...styles.tabBtn,
                ...(activeTab === 'settings' ? styles.activeTab : {}),
              }}
            >
              ⚙️ Preferences & Token Caps
            </button>
          </nav>
        )}
      </header>

      {!authed ? (
        <main>
          <div style={styles.authCard}>
            <h2 style={{ marginTop: 0, color: 'var(--primary)' }}>Authentication Required</h2>
            <p style={{ color: 'var(--text-main)', lineHeight: 1.5, fontSize: '0.95rem' }}>
              Authenticate via OAuth2 session (Google / GitHub) to access the pre-configured routing platform without manual provider API keys.
            </p>

            {authError && (
              <div style={styles.authErrorBanner} role="alert">
                ⚠️ {authError}
              </div>
            )}

            <button
              onClick={triggerMockLogin}
              disabled={loggingIn}
              style={{
                ...styles.loginBtn,
                opacity: loggingIn ? 0.7 : 1,
              }}
              aria-label="Sign in with OAuth"
            >
              {loggingIn ? 'Authenticating...' : 'Sign in with OAuth (Google / GitHub)'}
            </button>
          </div>
        </main>
      ) : (
        <main>
          {activeTab === 'chat' ? (
            <div id="panel-chat" role="tabpanel" aria-labelledby="tab-chat">
              <ChatInterface />
            </div>
          ) : (
            <div id="panel-settings" role="tabpanel" aria-labelledby="tab-settings">
              <UserSettings />
            </div>
          )}
        </main>
      )}
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  header: {
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'center',
    marginBottom: '24px',
    gap: '16px',
  },
  navTabs: {
    display: 'flex',
    gap: '8px',
    backgroundColor: 'var(--bg-card)',
    padding: '4px',
    borderRadius: '10px',
    border: '1px solid var(--border-color)',
  },
  tabBtn: {
    backgroundColor: 'transparent',
    color: 'var(--text-muted)',
    border: 'none',
    padding: '8px 16px',
    borderRadius: '8px',
    cursor: 'pointer',
    fontWeight: 600,
    fontSize: '0.9rem',
    transition: 'all 0.2s ease',
  },
  activeTab: {
    backgroundColor: 'var(--bg-card-hover)',
    color: 'var(--purple)',
  },
  authCard: {
    textAlign: 'center',
    margin: '60px auto',
    maxWidth: '450px',
    backgroundColor: 'var(--bg-card)',
    padding: '32px',
    borderRadius: '12px',
    border: '1px solid var(--border-color)',
  },
  authErrorBanner: {
    backgroundColor: 'rgba(248, 113, 113, 0.15)',
    color: 'var(--danger)',
    border: '1px solid rgba(248, 113, 113, 0.3)',
    padding: '12px',
    borderRadius: '8px',
    marginBottom: '16px',
    fontWeight: 'bold',
    fontSize: '0.85rem',
    textAlign: 'left',
  },
  loginBtn: {
    backgroundColor: 'var(--success)',
    color: '#0f172a',
    padding: '12px 24px',
    fontSize: '1rem',
    fontWeight: 'bold',
    border: 'none',
    borderRadius: '8px',
    cursor: 'pointer',
    marginTop: '8px',
    transition: 'opacity 0.2s ease',
  },
};

export default App;
