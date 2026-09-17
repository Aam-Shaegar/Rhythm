import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '../api/client';
import type { ReminderSettings, ThemeName, User } from '../api/types';
import { useAuth } from '../auth/AuthContext';
import { ErrorBlock, FieldError, Spinner } from '../components/ui';

const THEME_KEY = 'rhytm.theme';
const THEME_COLOR_KEY = 'rhytm.theme.color';
const THEME_PHOTO_KEY = 'rhytm.theme.photo';

export function loadTheme(): { name: ThemeName; color: string; photo: string | null } {
  return {
    name: (localStorage.getItem(THEME_KEY) as ThemeName) || 'forest',
    color: localStorage.getItem(THEME_COLOR_KEY) || '#8a8f6e',
    photo: localStorage.getItem(THEME_PHOTO_KEY),
  };
}

const DEFAULT_SETTINGS: ReminderSettings = {
  event_reminders: true,
  task_reminders: true,
  event_before_15m: true,
  event_before_1h: true,
  event_before_24h: true,
  task_before_1h: true,
  task_before_24h: true,
};

const SETTING_LABELS: Array<{ key: keyof ReminderSettings; label: string }> = [
  { key: 'event_reminders', label: 'Напоминания о событиях' },
  { key: 'task_reminders', label: 'Напоминания о задачах' },
  { key: 'event_before_15m', label: 'За 15 минут до события' },
  { key: 'event_before_1h', label: 'За час до события' },
  { key: 'event_before_24h', label: 'За сутки до события' },
  { key: 'task_before_1h', label: 'За час до задачи' },
  { key: 'task_before_24h', label: 'За сутки до задачи' },
];

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length >= 2) return (parts[0][0] + parts[1][0]).toUpperCase();
  return name.slice(0, 2).toUpperCase();
}

export default function SettingsScreen({
  theme,
  setTheme,
  notify,
}: {
  theme: ThemeName;
  setTheme: (t: ThemeName) => void;
  notify: (k: 'ok' | 'err', t: string) => void;
}) {
  const { user, logout, refreshUser } = useAuth();
  const [profile, setProfile] = useState<User | null>(user);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [name, setName] = useState('');
  const [email, setEmail] = useState('');
  const [formErr, setFormErr] = useState('');
  const [saving, setSaving] = useState(false);
  const [settings, setSettings] = useState<ReminderSettings>(DEFAULT_SETTINGS);
  const [settingsBusy, setSettingsBusy] = useState(false);
  const [customColor, setCustomColor] = useState(() => localStorage.getItem(THEME_COLOR_KEY) || '#8a8f6e');
  const [photo, setPhoto] = useState<string | null>(() => localStorage.getItem(THEME_PHOTO_KEY));

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const [me, rem] = await Promise.all([
        apiFetch<User>('/users/me'),
        apiFetch<ReminderSettings>('/reminders/settings').catch(() => DEFAULT_SETTINGS),
      ]);
      setProfile(me);
      setName(me.username);
      setEmail(me.email);
      setSettings({ ...DEFAULT_SETTINGS, ...rem });
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка загрузки');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function saveProfile() {
    setFormErr('');
    if (name.trim().length < 3) {
      setFormErr('Имя: минимум 3 символа');
      return;
    }
    if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) {
      setFormErr('Введите корректную почту');
      return;
    }
    setSaving(true);
    try {
      const updated = await apiFetch<User>('/users/me', {
        method: 'PATCH',
        body: JSON.stringify({ username: name.trim(), email: email.trim() }),
      });
      setProfile(updated);
      await refreshUser();
      notify('ok', 'Профиль обновлён');
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Не получилось сохранить';
      setFormErr(msg);
      notify('err', msg);
    } finally {
      setSaving(false);
    }
  }

  async function toggleSetting(key: keyof ReminderSettings) {
    const next = { ...settings, [key]: !settings[key] };
    setSettings(next);
    setSettingsBusy(true);
    try {
      const saved = await apiFetch<ReminderSettings>('/reminders/settings', {
        method: 'PATCH',
        body: JSON.stringify({ [key]: next[key] }),
      });
      setSettings({ ...DEFAULT_SETTINGS, ...saved });
    } catch (e) {
      setSettings(settings);
      notify('err', e instanceof Error ? e.message : 'Не получилось сохранить настройку');
    } finally {
      setSettingsBusy(false);
    }
  }

  function pickTheme(t: ThemeName) {
    setTheme(t);
    localStorage.setItem(THEME_KEY, t);
  }

  function onPhoto(file: File | undefined) {
    if (!file) return;
    if (photo) URL.revokeObjectURL(photo);
    const url = URL.createObjectURL(file);
    // Хранить blob-URL между сессиями нельзя — читаем как dataURL
    const reader = new FileReader();
    reader.onload = () => {
      const dataUrl = String(reader.result ?? '');
      localStorage.setItem(THEME_PHOTO_KEY, dataUrl);
      setPhoto(dataUrl);
    };
    reader.readAsDataURL(file);
    void url;
  }

  function removePhoto() {
    if (photo && photo.startsWith('blob:')) URL.revokeObjectURL(photo);
    localStorage.removeItem(THEME_PHOTO_KEY);
    setPhoto(null);
  }

  useEffect(() => {
    localStorage.setItem(THEME_COLOR_KEY, customColor);
    if (theme === 'custom') {
      document.getElementById('phoneScreen')?.style.setProperty('--accent-deep', customColor);
      document.getElementById('phoneScreen')?.style.setProperty('--accent', customColor);
    }
  }, [customColor, theme]);

  if (loading) return <Spinner />;
  if (error)
    return (
      <div className="screen active">
        <div className="screen-head">
          <div className="screen-title">Настройки</div>
        </div>
        <div className="screen-body">
          <ErrorBlock text={error} onRetry={() => void load()} />
        </div>
      </div>
    );

  return (
    <div className="screen active" data-screen="settings">
      <div className="screen-head">
        <div className="screen-title">Настройки</div>
      </div>
      <div className="screen-body">
        <div className="settings-section">
          <div className="profile-row">
            <div className="avatar" aria-hidden="true">
              {initials(profile?.username ?? '??')}
            </div>
            <div>
              <div className="name">{profile?.username}</div>
              <div className="mail">{profile?.email}</div>
            </div>
          </div>
          <div className="form-card" style={{ marginTop: 10 }}>
            <label className="field">
              <span>Имя пользователя</span>
              <input value={name} onChange={(e) => setName(e.target.value)} maxLength={100} />
            </label>
            <label className="field">
              <span>Почта</span>
              <input value={email} onChange={(e) => setEmail(e.target.value)} inputMode="email" />
            </label>
            <FieldError text={formErr} />
            <button className="btn-primary" onClick={() => void saveProfile()} disabled={saving}>
              {saving ? 'Сохранение…' : 'Сохранить профиль'}
            </button>
          </div>
        </div>

        <div className="settings-section">
          <div className="settings-label">Тема оформления</div>
          <div className="theme-gallery" role="radiogroup" aria-label="Тема">
            {(
              [
                ['forest', 'Лес'],
                ['coffee', 'Кофе'],
                ['lavender', 'Лаванда'],
                ['custom', 'Своя тема'],
              ] as Array<[ThemeName, string]>
            ).map(([t, label]) => (
              <button
                key={t}
                className={`theme-card${theme === t ? ' selected' : ''}`}
                onClick={() => pickTheme(t)}
                role="radio"
                aria-checked={theme === t}
              >
                <span className={`theme-swatch t${t === 'forest' ? 'f' : t === 'coffee' ? 'c' : t === 'lavender' ? 'l' : 'u'}`}>
                  {t === 'custom' ? 'своя' : ''}
                </span>
                <span className="theme-name">{label}</span>
              </button>
            ))}
          </div>
          {theme === 'custom' && (
            <div className="custom-controls show">
              <label>
                Цвет
                <input type="color" value={customColor} onChange={(e) => setCustomColor(e.target.value)} aria-label="Цвет темы" />
              </label>
              <label>
                Фото
                <input type="file" accept="image/*" onChange={(e) => onPhoto(e.target.files?.[0])} aria-label="Фоновая фотография" />
              </label>
              {photo && (
                <button className="btn-secondary" onClick={removePhoto} type="button">
                  Убрать фото
                </button>
              )}
            </div>
          )}
        </div>

        <div className="settings-section">
          <div className="settings-label">Уведомления</div>
          {SETTING_LABELS.map(({ key, label }) => (
            <div className="switch-row" key={key}>
              <span className="t">{label}</span>
              <button
                className={`toggle${settings[key] ? ' on' : ''}`}
                role="switch"
                aria-checked={settings[key]}
                aria-label={label}
                disabled={settingsBusy}
                onClick={() => void toggleSetting(key)}
              />
            </div>
          ))}
        </div>

        <button
          className="logout-btn"
          onClick={() => {
            logout();
            notify('ok', 'Вы вышли из аккаунта');
          }}
        >
          Выйти из аккаунта
        </button>
      </div>
    </div>
  );
}
