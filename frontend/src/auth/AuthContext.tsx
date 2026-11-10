import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { apiFetch, clearTokens, SessionExpiredError, setTokens } from '../api/client';
import type { AuthResponse, LoginResponse, User } from '../api/types';

interface AuthState {
  user: User | null;
  loading: boolean;
  login: (email: string, password: string) => Promise<void>;
  register: (username: string, email: string, password: string) => Promise<void>;
  logout: () => void;
  refreshUser: () => Promise<void>;
}

const AuthContext = createContext<AuthState | null>(null);

function normalizeLogin(data: LoginResponse, fallbackEmail: string): { user: User; access: string } {
  return {
    access: data.access_token,
    user: {
      id: data.user_id,
      username: data.username,
      email: data.email || fallbackEmail,
      created_at: new Date().toISOString(),
      updated_at: new Date().toISOString(),
    },
  };
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(() => { try { return JSON.parse(localStorage.getItem('rhytm.user') ?? 'null'); } catch { return null; } });
  const [loading, setLoading] = useState(true);

  const refreshUser = useCallback(async () => {
    try {
      const me = await apiFetch<User>('/users/me');
      setUser(me);
    } catch (e) {
      if (e instanceof SessionExpiredError) setUser(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refreshUser();
    const onUnauth = () => setUser(null);
    window.addEventListener('rhytm:unauthorized', onUnauth);
    return () => window.removeEventListener('rhytm:unauthorized', onUnauth);
  }, [refreshUser]);

  const login = useCallback(async (email: string, password: string) => {
    const data = await apiFetch<LoginResponse>('/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    const norm = normalizeLogin(data, email);
    localStorage.removeItem('rhytm.refresh_token');
    setTokens(norm.access);
    // Re-fetch full profile (created_at etc.).
    try {
      const me = await apiFetch<User>('/users/me');
      setUser(me);
    } catch {
      setUser(norm.user);
    }
  }, []);

  const register = useCallback(async (username: string, email: string, password: string) => {
    const data = await apiFetch<AuthResponse>('/auth/register', {
      method: 'POST',
      body: JSON.stringify({ username, email, password }),
    });
    setTokens(data.access_token);
    localStorage.removeItem('rhytm.refresh_token');
    setUser(data.user);
  }, []);

  const logout = useCallback(() => {
    void apiFetch('/auth/logout', { method: 'POST' }).catch(() => {});
    clearTokens();
    localStorage.removeItem('rhytm.user');
    setUser(null);
  }, []);

  useEffect(() => {
    if (user) localStorage.setItem('rhytm.user', JSON.stringify(user));
    else if (!loading) localStorage.removeItem('rhytm.user');
  }, [user, loading]);

  const value = useMemo(
    () => ({ user, loading, login, register, logout, refreshUser }),
    [user, loading, login, register, logout, refreshUser],
  );
  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthState {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error('useAuth must be used within AuthProvider');
  return ctx;
}
