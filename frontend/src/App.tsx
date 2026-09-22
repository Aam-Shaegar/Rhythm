import { useCallback, useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { AuthProvider, useAuth } from './auth/AuthContext';
import { apiFetch } from './api/client';
import type { EventItem, ReminderSettings, Task, ThemeName } from './api/types';
import { TABS, type Tab } from './nav';
import { deriveNotices, type Notice } from './notifications';
import AuthScreen from './screens/AuthScreen';
import DashboardScreen from './screens/DashboardScreen';
import ScheduleScreen from './screens/ScheduleScreen';
import TasksScreen from './screens/TasksScreen';
import ReportsScreen from './screens/ReportsScreen';
import SettingsScreen from './screens/SettingsScreen';
import ThemeArt from './components/ThemeArt';
import { applyThemeToDom, loadArt, loadThemeName, saveArt } from './theme';
import NotificationsPanel from './components/NotificationsPanel';
import { Toasts, type Toast } from './components/ui';

let toastId = 1;

const DEFAULT_SETTINGS: ReminderSettings = {
  event_reminders: true,
  task_reminders: true,
  event_before_15m: true,
  event_before_1h: true,
  event_before_24h: true,
  task_before_1h: true,
  task_before_24h: true,
};

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return name.slice(0, 2).toUpperCase();
}

function BellIcon() {
  return (
    <svg viewBox="0 0 24 24" fill="none" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M18 8a6 6 0 1 0-12 0c0 7-3 8-3 8h18s-3-1-3-8" />
      <path d="M13.7 20a2 2 0 0 1-3.4 0" />
    </svg>
  );
}

function Shell() {
  const { user, loading, logout } = useAuth();
  const [tab, setTab] = useState<Tab>('home');
  const [theme, setTheme] = useState<ThemeName>(() => loadThemeName());
  const [artOn, setArtOn] = useState<boolean>(() => loadArt());
  // Счётчик изменений палитры/фото своей темы — триггер переприменения к DOM.
  const [themeRev, setThemeRev] = useState(0);
  const refreshTheme = useCallback(() => setThemeRev((r) => r + 1), []);
  const [toasts, setToasts] = useState<Toast[]>([]);
  const [clock, setClock] = useState(() => new Date());
  const [notices, setNotices] = useState<Notice[]>([]);
  const [notifOpen, setNotifOpen] = useState(false);
  const btnRefs = useRef<Array<HTMLButtonElement | null>>([]);
  const [highlight, setHighlight] = useState({ left: 6, width: 52 });

  const notify = useCallback((kind: 'ok' | 'err', text: string) => {
    const id = toastId++;
    setToasts((prev) => [...prev.slice(-2), { id, kind, text }]);
    setTimeout(() => setToasts((prev) => prev.filter((t) => t.id !== id)), 3200);
  }, []);

  const go = useCallback((t: Tab) => setTab(t), []);

  // Уведомления в интерфейсе.
  const loadNotices = useCallback(async () => {
    try {
      const from = new Date();
      from.setHours(0, 0, 0, 0);
      const to = new Date(Date.now() + 7 * 864e5);
      const [tasks, events, settings] = await Promise.all([
        apiFetch<Task[]>('/tasks'),
        apiFetch<EventItem[]>(
          `/events?date_from=${encodeURIComponent(from.toISOString())}&date_to=${encodeURIComponent(to.toISOString())}`,
        ).catch(() => apiFetch<EventItem[]>('/events')),
        apiFetch<ReminderSettings>('/reminders/settings').catch(() => DEFAULT_SETTINGS),
      ]);
      setNotices(deriveNotices(tasks, events, settings));
    } catch {
      // Тихо: бейдж не покажем, ошибки видны на экранах.
    }
  }, []);

  useEffect(() => {
    if (!user) return;
    void loadNotices();
    const t = setInterval(() => void loadNotices(), 60000);
    return () => clearInterval(t);
  }, [user, loadNotices, tab]);

  // Единственное место применения темы к DOM.
  useEffect(() => {
    applyThemeToDom();
  }, [theme, artOn, themeRev]);

  const handleArtChange = useCallback(
    (on: boolean) => {
      saveArt(on);
      setArtOn(on);
    },
    [],
  );

  useEffect(() => {
    const t = setInterval(() => setClock(new Date()), 20000);
    return () => clearInterval(t);
  }, []);

  const measure = useCallback(() => {
    const idx = TABS.findIndex((t) => t.id === tab);
    const el = btnRefs.current[idx];
    if (el) setHighlight({ left: el.offsetLeft, width: el.offsetWidth });
  }, [tab]);

  useLayoutEffect(() => {
    measure();
  }, [measure]);
  useEffect(() => {
    window.addEventListener('resize', measure);
    return () => window.removeEventListener('resize', measure);
  }, [measure]);

  const clockLabel = clock.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  const dateLabel = useMemo(
    () => clock.toLocaleDateString('ru-RU', { weekday: 'short', day: 'numeric', month: 'short' }),
    [clock],
  );

  const bell = (
    <button
      className="bell-btn"
      onClick={() => {
        setNotifOpen(true);
        void loadNotices();
      }}
      aria-label={notices.length > 0 ? `Уведомления: ${notices.length}` : 'Уведомления'}
      title="Уведомления"
    >
      <BellIcon />
      {notices.length > 0 && (
        <span className="bell-badge" aria-hidden="true">
          {notices.length > 9 ? '9+' : notices.length}
        </span>
      )}
    </button>
  );

  if (loading) {
    return (
      <div className="phone">
        <div className="phone-screen" data-theme={theme} id="phoneScreen">
          <div className="bg-layer" />
          <div className="boot">Загрузка…</div>
        </div>
      </div>
    );
  }

  if (!user) {
    return (
      <div className="phone">
        <div className="phone-screen" data-theme={theme} id="phoneScreen">
          <div className="bg-layer" />
          <ThemeArt theme={theme} artOn={artOn} />
          <div className="status-bar">
            <span>{clockLabel}</span>
            <span>Ритм</span>
          </div>
          <div className="screens">
            <AuthScreen notify={notify} />
          </div>
          <Toasts items={toasts} />
        </div>
      </div>
    );
  }

  return (
    <div className="phone">
      <div className="phone-screen" data-theme={theme} id="phoneScreen">
        <div className="bg-layer" />
        <ThemeArt theme={theme} artOn={artOn} />
        <div className="bg-custom-photo" id="customPhoto" />
        <div className="bg-custom-overlay" />

        <aside className="sidebar" aria-label="Навигация">
          <div className="brand">
            <span className="brand-mark" aria-hidden="true">
              Р
            </span>
            <span className="brand-name">Ритм</span>
          </div>
          <nav className="side-nav" role="tablist" aria-label="Разделы">
            {TABS.map((t) => (
              <button
                key={t.id}
                className={tab === t.id ? 'active' : ''}
                role="tab"
                aria-selected={tab === t.id}
                onClick={() => setTab(t.id)}
              >
                <span className="side-ico">{t.icon}</span>
                <span>{t.label}</span>
              </button>
            ))}
          </nav>
          <div className="side-footer">
            <button className="side-user" onClick={() => setTab('settings')} title="Профиль и настройки">
              <span className="avatar sm" aria-hidden="true">
                {initials(user.username)}
              </span>
              <span className="side-user-name">{user.username}</span>
            </button>
            <button
              className="icon-btn"
              title="Выйти из аккаунта"
              aria-label="Выйти из аккаунта"
              onClick={() => {
                logout();
                notify('ok', 'Вы вышли из аккаунта');
              }}
            >
              ⏻
            </button>
          </div>
        </aside>

        <div className="main-col">
          <header className="topbar">
            <div className="topbar-date">
              {dateLabel} · {clockLabel}
            </div>
            <div className="topbar-right">
              {bell}
              <button className="topbar-user" onClick={() => setTab('settings')} title="Профиль и настройки">
                <span className="avatar sm" aria-hidden="true">
                  {initials(user.username)}
                </span>
                <span>{user.username}</span>
              </button>
            </div>
          </header>

          <div className="status-bar">
            <span>{clockLabel}</span>
            <span>Ритм</span>
            <span className="status-bell">{bell}</span>
          </div>

          <div className="screens">
            {tab === 'home' && <DashboardScreen go={go} notify={notify} />}
            {tab === 'calendar' && <ScheduleScreen notify={notify} />}
            {tab === 'tasks' && <TasksScreen notify={notify} />}
            {tab === 'reports' && <ReportsScreen />}
            {tab === 'settings' && (
              <SettingsScreen
                theme={theme}
                setTheme={setTheme}
                artOn={artOn}
                onArtChange={handleArtChange}
                onThemeChanged={refreshTheme}
                notify={notify}
              />
            )}
          </div>

          <div className="switcher-wrap">
            <div className="switcher" role="tablist" aria-label="Разделы">
              <div
                className="highlight"
                aria-hidden="true"
                style={{ transform: `translateX(${highlight.left - 6}px)`, width: highlight.width }}
              />
              {TABS.map((t, i) => (
                <button
                  key={t.id}
                  ref={(el) => {
                    btnRefs.current[i] = el;
                  }}
                  className={tab === t.id ? 'active' : ''}
                  role="tab"
                  aria-selected={tab === t.id}
                  title={t.label}
                  aria-label={t.label}
                  onClick={() => setTab(t.id)}
                >
                  {t.icon}
                </button>
              ))}
            </div>
          </div>
        </div>

        <NotificationsPanel open={notifOpen} notices={notices} onClose={() => setNotifOpen(false)} onGo={go} />
        <Toasts items={toasts} />
      </div>
    </div>
  );
}

export default function App() {
  return (
    <AuthProvider>
      <div className="page-label">Умный органайзер «Ритм» — расписание, задачи, отчёты и напоминания.</div>
      <Shell />
    </AuthProvider>
  );
}
