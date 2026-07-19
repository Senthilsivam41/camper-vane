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
    <div style={{ backgroundColor: '#11111b', minHeight: '100vh', padding: '20px', color: '#cdd6f4' }}>
      <header style={styles.header}>
        <div style={{ textAlign: 'center' }}>
          <h1 style={{ color: '#cba6f7', margin: 0, fontSize: '1.8rem' }}>Camper Vane</h1>
          <p style={{ color: '#a6adc8', margin: '4px 0 0 0', fontSize: '0.9rem' }}>
            Cost-Aware Dynamic LLM Gateway & Real-time Optimization Dashboard
          </p>
        </div>

        {authed && (
          <nav style={styles.navTabs}>
            <button
              onClick={() => setActiveTab('chat')}
              style={{
                ...styles.tabBtn,
                ...(activeTab === 'chat' ? styles.activeTab : {}),
              }}
            >
              💬 Chat & Real-Time Metrics
            </button>
            <button
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
        <div style={{ textAlign: 'center', margin: '60px auto', maxWidth: '450px', backgroundColor: '#181825', padding: '32px', borderRadius: '12px', border: '1px solid #313244' }}>
          <h3 style={{ marginTop: 0, color: '#89b4fa' }}>Authentication Required</h3>
          <p style={{ color: '#cdd6f4', lineHeight: 1.5 }}>
            Authenticate via OAuth2 session (Google / GitHub) to access the pre-configured routing platform without manual provider API keys.
          </p>

          {authError && (
            <div style={{ backgroundColor: '#f38ba8', color: '#11111b', padding: '12px', borderRadius: '8px', marginBottom: '16px', fontWeight: 'bold', fontSize: '0.85rem', textAlign: 'left' }}>
              ⚠️ {authError}
            </div>
          )}

          <button
            onClick={triggerMockLogin}
            disabled={loggingIn}
            style={{
              backgroundColor: '#a6e3a1',
              color: '#11111b',
              padding: '12px 24px',
              fontSize: '1rem',
              fontWeight: 'bold',
              border: 'none',
              borderRadius: '8px',
              cursor: 'pointer',
              marginTop: '8px',
            }}
          >
            {loggingIn ? 'Authenticating...' : 'Sign in with OAuth (Google / GitHub)'}
          </button>
        </div>
      ) : (
        <main>
          {activeTab === 'chat' ? <ChatInterface /> : <UserSettings />}
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
    backgroundColor: '#181825',
    padding: '4px',
    borderRadius: '10px',
    border: '1px solid #313244',
  },
  tabBtn: {
    backgroundColor: 'transparent',
    color: '#a6adc8',
    border: 'none',
    padding: '8px 16px',
    borderRadius: '8px',
    cursor: 'pointer',
    fontWeight: 600,
    fontSize: '0.9rem',
    transition: 'all 0.2s ease',
  },
  activeTab: {
    backgroundColor: '#313244',
    color: '#cba6f7',
  },
};

export default App;
