import { createContext, useCallback, useContext, useEffect, useMemo, useState } from 'react';
import type { ReactNode } from 'react';
import { apiFetch, clearTokens, getAccessToken, setTokens } from '../api/client';
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
  const [user, setUser] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);

  const refreshUser = useCallback(async () => {
    if (!getAccessToken()) {
      setUser(null);
      setLoading(false);
      return;
    }
    try {
      const me = await apiFetch<User>('/users/me');
      setUser(me);
    } catch {
      setUser(null);
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
    // Login returns no refresh_token in body (HttpOnly cookie); keep stored one.
    const prevRefresh = localStorage.getItem('rhytm.refresh_token') ?? '';
    setTokens(norm.access, prevRefresh || undefined);
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
    setTokens(data.access_token, data.refresh_token);
    setUser(data.user);
  }, []);

  const logout = useCallback(() => {
    clearTokens();
    setUser(null);
  }, []);

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
