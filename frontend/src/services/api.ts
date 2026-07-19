export interface UserConfig {
  user_id: string;
  daily_token_cap: number;
  routing_strategy: 'simple' | 'advanced';
  preferred_models: string[];
}

export async function fetchUserConfig(): Promise<UserConfig> {
  const res = await fetch('/api/v1/user/config', { credentials: 'include' });
  if (!res.ok) {
    throw new Error(`Failed to fetch user config (${res.status})`);
  }
  return res.json();
}

export async function updateUserConfig(config: Partial<UserConfig>): Promise<UserConfig> {
  const res = await fetch('/api/v1/user/config', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'include',
    body: JSON.stringify(config),
  });
  if (!res.ok) {
    const errorText = await res.text();
    throw new Error(errorText || `Failed to update config (${res.status})`);
  }
  return res.json();
}

export async function loginOAuth(provider: string = 'google'): Promise<{ auth_url: string }> {
  const res = await fetch(`/api/v1/auth/login?provider=${provider}`, { credentials: 'include' });
  if (!res.ok) throw new Error('Login failed');
  return res.json();
}

export async function handleCallback(code: string, mockUserId?: string): Promise<any> {
  const url = mockUserId
    ? `/api/v1/auth/callback?code=${code}&mock_user_id=${mockUserId}`
    : `/api/v1/auth/callback?code=${code}`;
  const res = await fetch(url, { credentials: 'include' });
  if (!res.ok) {
    const errText = await res.text();
    throw new Error(errText || `Auth callback failed (${res.status})`);
  }
  return res.json();
}

export async function checkMe(): Promise<UserConfig> {
  const res = await fetch('/api/v1/auth/me', { credentials: 'include' });
  if (!res.ok) throw new Error('Not authenticated');
  return res.json();
}
