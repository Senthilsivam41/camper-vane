export interface UserConfig {
  user_id: string;
  daily_token_cap: number;
  routing_strategy: 'simple' | 'advanced';
  preferred_models: string[];
}

export interface LoginResponse {
  auth_url: string;
  provider: string;
  mock: boolean;
  available_providers: string[];
}

export interface UsageInfo {
  user_id: string;
  window_hours: number;
  tokens_used: number;
  daily_token_cap: number;
  utilization_pct: number;
  as_of: string;
}

export interface SessionSummary {
  session_id: string;
  updated_at: string;
  message_count: number;
  preview: string;
}

export interface SessionMessageDTO {
  session_id: string;
  user_id?: string;
  role: 'user' | 'assistant' | string;
  content: string;
  timestamp: string;
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

export async function fetchUsage(): Promise<UsageInfo> {
  const res = await fetch('/api/v1/user/usage', { credentials: 'include' });
  if (!res.ok) {
    throw new Error(`Failed to fetch usage (${res.status})`);
  }
  return res.json();
}

export async function fetchSessions(limit = 20): Promise<SessionSummary[]> {
  const res = await fetch(`/api/v1/sessions?limit=${limit}`, { credentials: 'include' });
  if (!res.ok) {
    throw new Error(`Failed to fetch sessions (${res.status})`);
  }
  return res.json();
}

export async function fetchSessionMessages(sessionId: string, limit = 100): Promise<SessionMessageDTO[]> {
  const res = await fetch(`/api/v1/sessions/${encodeURIComponent(sessionId)}/messages?limit=${limit}`, {
    credentials: 'include',
  });
  if (!res.ok) {
    throw new Error(`Failed to fetch session messages (${res.status})`);
  }
  return res.json();
}

export async function loginOAuth(provider: string = 'google'): Promise<LoginResponse> {
  const res = await fetch(`/api/v1/auth/login?provider=${provider}`, { credentials: 'include' });
  if (!res.ok) {
    const errText = await res.text();
    throw new Error(errText || 'Login failed');
  }
  return res.json();
}

export async function authStatus(): Promise<LoginResponse> {
  const res = await fetch('/api/v1/auth/login?intent=status', { credentials: 'include' });
  if (!res.ok) {
    throw new Error('Failed to load auth status');
  }
  return res.json();
}

export async function handleCallback(
  code: string,
  options?: { mockUserId?: string; provider?: string; state?: string }
): Promise<any> {
  const params = new URLSearchParams({ code });
  if (options?.mockUserId) params.set('mock_user_id', options.mockUserId);
  if (options?.provider) params.set('provider', options.provider);
  if (options?.state) params.set('state', options.state);
  params.set('format', 'json');

  const res = await fetch(`/api/v1/auth/callback?${params.toString()}`, {
    credentials: 'include',
    headers: { Accept: 'application/json' },
  });
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

export async function logout(): Promise<void> {
  const res = await fetch('/api/v1/auth/logout', {
    method: 'POST',
    credentials: 'include',
  });
  if (!res.ok) {
    const errText = await res.text();
    throw new Error(errText || `Logout failed (${res.status})`);
  }
}
