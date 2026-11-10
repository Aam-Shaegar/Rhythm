import type { ApiError } from './types';

declare global {
  interface Window {
    __ENV__?: { API_URL?: string };
  }
}

// Precedence: runtime config.js → Vite env → localhost fallback.
function resolveApiUrl(): string {
  const rt = window.__ENV__;
  if (rt && typeof rt.API_URL === 'string') {
    return rt.API_URL.replace(/\/$/, '') || '/api/v1';
  }
  const baked = import.meta.env?.VITE_API_URL as string | undefined;
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
  if (getAccessToken() !== access) invalidateCache();
  localStorage.setItem(ACCESS_KEY, access);
  if (refresh !== undefined) localStorage.setItem(REFRESH_KEY, refresh);
}

export function clearTokens(): void {
  invalidateCache();
  localStorage.removeItem(ACCESS_KEY);
  localStorage.removeItem(REFRESH_KEY);
}

// 12 c с запасом на сеть; при превышении — понятная ошибка без вечного спиннера.
const REQUEST_TIMEOUT_MS = 12000;

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

let refreshing: Promise<string> | null = null;
export class SessionExpiredError extends Error {}

export function refreshAccessToken(): Promise<string> {
  if (refreshing) return refreshing;
  const refresh = async () => {
    const ctrl = new AbortController();
    const timer = setTimeout(() => ctrl.abort(), REQUEST_TIMEOUT_MS);
    try {
      // Prefer the HttpOnly cookie; migrate older registrations that only stored a header token.
      let res = await fetch(`${API_URL}/auth/refresh`, { method: 'POST', credentials: 'include', signal: ctrl.signal });
      const legacy = getRefreshToken();
      if (res.status === 401 && legacy) res = await fetch(`${API_URL}/auth/refresh`, { method: 'POST', credentials: 'include', headers: { 'X-Refresh-Token': legacy }, signal: ctrl.signal });
      if (res.status === 401) throw new SessionExpiredError('Сессия истекла. Войдите снова.');
      if (!res.ok) throw new Error('Сервер временно недоступен. Повторите позже.');
      const data = await res.json() as { access_token: string };
      if (!data.access_token) throw new Error('Не удалось обновить сессию');
      setTokens(data.access_token);
      localStorage.removeItem(REFRESH_KEY);
      return data.access_token;
    } finally { clearTimeout(timer); }
  };
  refreshing = (async () => {
    // Serialize refresh across tabs, then reuse the token another tab just obtained.
    const previous = getAccessToken();
    if (typeof navigator !== 'undefined' && navigator.locks) {
      return navigator.locks.request('rhytm:refresh', async () => {
        const latest = getAccessToken();
        return latest && latest !== previous ? latest : refresh();
      });
    }
    return refresh();
  })().finally(() => { refreshing = null; });
  return refreshing;
}

async function request<T>(path: string, init: RequestInit = {}, retry = true): Promise<T> {
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

  if (res.status === 401 && retry && !path.startsWith('/auth/')) {
    try {
      const latest = getAccessToken();
      if (!latest || latest === token) await refreshAccessToken();
      return request<T>(path, init, false);
    } catch (e) {
      if (e instanceof SessionExpiredError) {
        clearTokens();
        window.dispatchEvent(new Event('rhytm:unauthorized'));
      }
      throw e;
    }
  }

  let body: unknown = null;
  try {
    body = await res.json();
  } catch {
    body = null;
  }

  if (!res.ok) {
    const error = toApiError(res.status, body);
    throw Object.assign(new Error(error.message), error);
  }
  return body as T;
}

const responseCache = new Map<string, { at: number; value: unknown }>();
const pendingGets = new Map<string, Promise<unknown>>();
let cacheEpoch = 0;
export function invalidateCache(): void { cacheEpoch++; responseCache.clear(); pendingGets.clear(); }
function accountKey(): string {
 const token = getAccessToken();
 if (!token) return 'cookie-session';
 try { return JSON.parse(atob(token.split('.')[1].replace(/-/g,'+').replace(/_/g,'/'))).uid; }
 catch { return token; }
}
export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<T> {
 const method = (init.method ?? 'GET').toUpperCase();
 if (method !== 'GET') {
  invalidateCache();
  const result = await request<T>(path, init);
  invalidateCache();
  if (!path.startsWith('/auth/')) window.dispatchEvent(new Event('rhytm:data-changed'));
  return result;
 }
 const key = `${accountKey()}:${path}`;
 const cached = responseCache.get(key);
 if (cached && Date.now() - cached.at < 15000) return structuredClone(cached.value) as T;
 const existing = pendingGets.get(key);
 if (existing) return structuredClone(await existing) as T;
 const epoch = cacheEpoch;
 const promise = request<T>(path, init).then(value => {
  if (epoch === cacheEpoch) {
   if (responseCache.size >= 50) responseCache.delete(responseCache.keys().next().value!);
   responseCache.set(key, { at: Date.now(), value: structuredClone(value) });
  }
  return value;
 }).finally(() => { if (pendingGets.get(key) === promise) pendingGets.delete(key); });
 pendingGets.set(key, promise);
 return structuredClone(await promise);
}
