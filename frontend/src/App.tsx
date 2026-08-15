import { useState, useEffect } from 'react';
import { ChatInterface } from './components/ChatInterface';
import { UserSettings } from './components/UserSettings';
import { handleCallback, checkMe, loginOAuth, logout, authStatus } from './services/api';

export function App() {
  const [authed, setAuthed] = useState<boolean>(false);
  const [loggingIn, setLoggingIn] = useState<boolean>(false);
  const [loggingOut, setLoggingOut] = useState<boolean>(false);
  const [authError, setAuthError] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'chat' | 'settings'>('chat');
  const [mockAvailable, setMockAvailable] = useState<boolean>(true);
  const [flyToast, setFlyToast] = useState<string | null>(null);

  const handleSettingsSaved = () => {
    setFlyToast('Preferences saved');
    setActiveTab('chat');
  };

  useEffect(() => {
    if (!flyToast) return;
    const id = window.setTimeout(() => setFlyToast(null), 2800);
    return () => window.clearTimeout(id);
  }, [flyToast]);

  useEffect(() => {
    const params = new URLSearchParams(window.location.search);
    const authQuery = params.get('auth');
    if (authQuery === 'success') {
      window.history.replaceState({}, '', window.location.pathname);
    }

    checkMe()
      .then(() => setAuthed(true))
      .catch(() => setAuthed(false));

    // Probe whether mock auth is available (no IdP credentials locally).
    authStatus()
      .then((resp) => {
        setMockAvailable(resp.mock || (resp.available_providers || []).includes('mock'));
      })
      .catch(() => setMockAvailable(false));
  }, []);

  const startOAuth = async (provider: 'google' | 'github') => {
    try {
      setLoggingIn(true);
      setAuthError(null);
      const resp = await loginOAuth(provider);
      if (resp.mock) {
        await handleCallback('mock_auth_code', {
          mockUserId: 'developer-1',
          provider,
        });
        setAuthed(true);
        return;
      }
      window.location.href = resp.auth_url;
    } catch (err: any) {
      console.error('Login error:', err);
      setAuthError(
        err.message ||
          'Failed to start OAuth. Ensure the Go server is running and provider credentials are configured.'
      );
    } finally {
      setLoggingIn(false);
    }
  };

  const triggerMockLogin = async () => {
    try {
      setLoggingIn(true);
      setAuthError(null);
      await handleCallback('mock_auth_code', {
        mockUserId: 'developer-1',
        provider: 'mock',
      });
      setAuthed(true);
    } catch (err: any) {
      console.error('Login error:', err);
      setAuthError(
        err.message ||
          'Failed to connect to backend server on http://localhost:8080. Make sure `go run ./cmd/server/main.go` is running.'
      );
    } finally {
      setLoggingIn(false);
    }
  };

  const handleLogout = async () => {
    try {
      setLoggingOut(true);
      await logout();
      setAuthed(false);
      setActiveTab('chat');
    } catch (err: any) {
      console.error('Logout error:', err);
      setAuthError(err.message || 'Logout failed');
    } finally {
      setLoggingOut(false);
    }
  };

  return (
    <div style={{ backgroundColor: 'var(--bg-app)', minHeight: '100vh', padding: '20px', color: 'var(--text-main)' }}>
      {flyToast && (
        <div className="fly-toast" role="status" aria-live="polite">
          {flyToast}
        </div>
      )}
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
          <div style={styles.headerActions}>
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
                Chat & Real-Time Metrics
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
                Preferences & Token Caps
              </button>
            </nav>
            <button
              onClick={handleLogout}
              disabled={loggingOut}
              style={styles.logoutBtn}
              aria-label="Log out"
            >
              {loggingOut ? 'Signing out...' : 'Log out'}
            </button>
          </div>
        )}
      </header>

      {!authed ? (
        <main>
          <div style={styles.authCard}>
            <h2 style={{ marginTop: 0, color: 'var(--primary)' }}>Authentication Required</h2>
            <p style={{ color: 'var(--text-main)', lineHeight: 1.5, fontSize: '0.95rem' }}>
              Sign in with Google or GitHub. Provider API keys stay on the server — you never paste them into the UI.
            </p>

            {authError && (
              <div style={styles.authErrorBanner} role="alert">
                {authError}
              </div>
            )}

            <div style={styles.loginStack}>
              <button
                onClick={() => startOAuth('google')}
                disabled={loggingIn}
                style={{ ...styles.loginBtn, opacity: loggingIn ? 0.7 : 1 }}
                aria-label="Sign in with Google"
              >
                {loggingIn ? 'Authenticating...' : 'Sign in with Google'}
              </button>
              <button
                onClick={() => startOAuth('github')}
                disabled={loggingIn}
                style={{ ...styles.loginBtnSecondary, opacity: loggingIn ? 0.7 : 1 }}
                aria-label="Sign in with GitHub"
              >
                Sign in with GitHub
              </button>
              {mockAvailable && (
                <button
                  onClick={triggerMockLogin}
                  disabled={loggingIn}
                  style={{ ...styles.loginBtnGhost, opacity: loggingIn ? 0.7 : 1 }}
                  aria-label="Continue with local mock auth"
                >
                  Continue with local mock auth
                </button>
              )}
            </div>
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
              <UserSettings onSaved={handleSettingsSaved} />
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
  headerActions: {
    display: 'flex',
    flexWrap: 'wrap',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '12px',
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
  logoutBtn: {
    backgroundColor: 'transparent',
    color: 'var(--danger)',
    border: '1px solid var(--border-color)',
    padding: '8px 14px',
    borderRadius: '8px',
    cursor: 'pointer',
    fontWeight: 600,
    fontSize: '0.85rem',
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
  loginStack: {
    display: 'flex',
    flexDirection: 'column',
    gap: '10px',
    marginTop: '8px',
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
    transition: 'opacity 0.2s ease',
  },
  loginBtnSecondary: {
    backgroundColor: 'var(--bg-card-hover)',
    color: 'var(--text-main)',
    padding: '12px 24px',
    fontSize: '1rem',
    fontWeight: 'bold',
    border: '1px solid var(--border-color)',
    borderRadius: '8px',
    cursor: 'pointer',
    transition: 'opacity 0.2s ease',
  },
  loginBtnGhost: {
    backgroundColor: 'transparent',
    color: 'var(--text-muted)',
    padding: '10px 24px',
    fontSize: '0.9rem',
    fontWeight: 600,
    border: '1px dashed var(--border-color)',
    borderRadius: '8px',
    cursor: 'pointer',
  },
};

export default App;
