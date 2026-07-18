import { useState } from 'react';
import { UserSettings } from './components/UserSettings';
import { handleCallback } from './services/api';

export function App() {
  const [authed, setAuthed] = useState<boolean>(false);
  const [loggingIn, setLoggingIn] = useState<boolean>(false);

  const triggerMockLogin = async () => {
    try {
      setLoggingIn(true);
      await handleCallback('mock_auth_code', 'developer-1');
      setAuthed(true);
    } catch (err) {
      console.error(err);
    } finally {
      setLoggingIn(false);
    }
  };

  return (
    <div style={{ backgroundColor: '#11111b', minHeight: '100vh', padding: '20px' }}>
      <header style={{ textAlign: 'center', marginBottom: '30px' }}>
        <h1 style={{ color: '#cba6f7', margin: 0 }}>Camper Vane</h1>
        <p style={{ color: '#a6adc8' }}>Cost-Aware Dynamic LLM Gateway & Dashboard</p>
      </header>

      {!authed ? (
        <div style={{ textAlign: 'center', margin: '40px auto', maxWidth: '400px' }}>
          <p style={{ color: '#cdd6f4' }}>Please authenticate via OAuth2 session to access your gateway profile.</p>
          <button
            onClick={triggerMockLogin}
            disabled={loggingIn}
            style={{
              backgroundColor: '#a6e3a1',
              color: '#11111b',
              padding: '12px 24px',
              fontSize: '1.1rem',
              fontWeight: 'bold',
              border: 'none',
              borderRadius: '8px',
              cursor: 'pointer',
            }}
          >
            {loggingIn ? 'Authenticating...' : 'Sign in with OAuth (Google / GitHub)'}
          </button>
        </div>
      ) : (
        <main>
          <UserSettings />
        </main>
      )}
    </div>
  );
}

export default App;
