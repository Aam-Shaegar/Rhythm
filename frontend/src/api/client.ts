import type { ApiError } from './types';

declare global {
  interface Window {
    __ENV__?: { API_URL?: string };
  }
}

// Precedence: runtime config.js (deploy-time; empty string = same origin)
// → Vite build env (dev) → localhost fallback.
function resolveApiUrl(): string {
  const rt = window.__ENV__;
  if (rt && typeof rt.API_URL === 'string') {
    return rt.API_URL.replace(/\/$/, '') || '/api/v1';
  }
  const baked = import.meta.env.VITE_API_URL as string | undefined;
  return baked?.replace(/\/$/, '') || 'http://localhost:8080/api/v1';
}

const API_URL = resolveApiUrl();

const ACCESS_KEY = 'rhytm.access_token';
const REFRESH_KEY = 'rhytm.refresh_token';

export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_KEY);
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_KEY);
}

export function setTokens(access: string, refresh?: string): void {
  localStorage.setItem(ACCESS_KEY, access);
  if (refresh !== undefined) localStorage.setItem(REFRESH_KEY, refresh);
}

export function clearTokens(): void {
  localStorage.removeItem(ACCESS_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

// Клиентский страж времени отклика (ТЗ R.1: стандартные операции — до 2 c,
// отчёты — до 5 c на сервере). 12 c с запасом на сеть; при превышении —
// понятная ошибка с рекомендацией (ТЗ U.3), без бесконечного спиннера.
const REQUEST_TIMEOUT_MS = 12000;

export function isTimeoutError(e: unknown): boolean {
  return e instanceof Error && e.name === 'TimeoutError';
}

function timeoutMessage(): Error {
  const err = new Error('Превышено время ожидания ответа (12 c). Проверьте соединение с интернетом и повторите.');
  err.name = 'TimeoutError';
  return err;
}

export function toApiError(status: number, body: unknown): ApiError {
  if (body && typeof body === 'object' && 'message' in body) {
    const b = body as { message?: unknown; error?: unknown };
    return {
      status,
      message: typeof b.message === 'string' ? b.message : 'Ошибка запроса',
      error: typeof b.error === 'string' ? b.error : 'Error',
    };
  }
  return { status, message: `Ошибка ${status}`, error: 'Error' };
}

async function refreshAccessToken(): Promise<string | null> {
  const rt = getRefreshToken();
  if (!rt) return null;
  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), REQUEST_TIMEOUT_MS);
  try {
    const res = await fetch(`${API_URL}/auth/refresh`, {
      method: 'POST',
      headers: { 'X-Refresh-Token': rt },
      signal: ctrl.signal,
    });
    if (!res.ok) {
      // try cookie-based refresh (backend also reads Cookie: refresh_token)
      const res2 = await fetch(`${API_URL}/auth/refresh`, { method: 'POST', credentials: 'include' });
      if (!res2.ok) return null;
      const data = (await res2.json()) as { access_token: string };
      if (data.access_token) {
        setTokens(data.access_token);
        return data.access_token;
      }
      return null;
    }
    const data = (await res.json()) as { access_token: string; refresh_token?: string };
    if (data.access_token) {
      setTokens(data.access_token, data.refresh_token);
      return data.access_token;
    }
    return null;
  } catch {
    return null;
  } finally {
    clearTimeout(timer);
  }
}

export async function apiFetch<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
    ...((init.headers as Record<string, string> | undefined) ?? {}),
  };
  const token = getAccessToken();
  if (token) headers['Authorization'] = `Bearer ${token}`;

  const ctrl = new AbortController();
  const timer = setTimeout(() => ctrl.abort(), REQUEST_TIMEOUT_MS);
  let res: Response;
  try {
    res = await fetch(`${API_URL}${path}`, {
      ...init,
      headers,
      credentials: 'include',
      signal: ctrl.signal,
    });
  } catch (e) {
    if (e instanceof DOMException && e.name === 'AbortError') throw timeoutMessage();
    throw e;
  } finally {
    clearTimeout(timer);
  }

  if (res.status === 204) return undefined as unknown as T;

  if (res.status === 401 && retry && token) {
    const fresh = await refreshAccessToken();
    if (fresh) return apiFetch<T>(path, init, false);
    clearTokens();
    window.dispatchEvent(new Event('rhytm:unauthorized'));
  }

  let body: unknown = null;
  try {
    body = await res.json();
  } catch {
    body = null;
  }

  if (!res.ok) throw toApiError(res.status, body);
  return body as T;
}

export { API_URL };
