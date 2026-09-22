import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '../api/client';
import {
  currentPushState,
  disablePush,
  enablePush,
  isIOS,
  isStandalone,
  type PushState,
} from '../api/push';
import type { ReminderSettings, ThemeName, User } from '../api/types';
import { useAuth } from '../auth/AuthContext';
import { ErrorBlock, FieldError, Spinner } from '../components/ui';
import {
  DEFAULT_CUSTOM,
  loadCustom,
  loadPhoto,
  saveCustom,
  saveThemeName,
  type CustomPalette,
} from '../theme';

// Фото храним как dataURL в localStorage: лимит ~2.5 МБ, иначе квота.
const MAX_PHOTO_BYTES = 2.5 * 1024 * 1024;

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
  artOn,
  onArtChange,
  onThemeChanged,
  notify,
}: {
  theme: ThemeName;
  setTheme: (t: ThemeName) => void;
  artOn: boolean;
  onArtChange: (on: boolean) => void;
  onThemeChanged: () => void;
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
  const [custom, setCustom] = useState<CustomPalette>(() => loadCustom());
  const [photo, setPhoto] = useState<string | null>(() => loadPhoto());

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
    saveThemeName(t);
    onThemeChanged();
  }

  function setPalette(patch: Partial<CustomPalette>) {
    const next = { ...custom, ...patch };
    setCustom(next);
    saveCustom(next);
    onThemeChanged();
  }

  function resetPalette() {
    setCustom({ ...DEFAULT_CUSTOM });
    saveCustom({ ...DEFAULT_CUSTOM });
    onThemeChanged();
    notify('ok', 'Цвета сброшены');
  }

  function onPhoto(file: File | undefined) {
    if (!file) return;
    if (!file.type.startsWith('image/')) {
      notify('err', 'Нужен файл-картинка');
      return;
    }
    if (file.size > MAX_PHOTO_BYTES) {
      notify('err', 'Картинка тяжелее 2.5 МБ — сожмите её и попробуйте снова');
      return;
    }
    if (photo) URL.revokeObjectURL(photo);
    const reader = new FileReader();
    reader.onload = () => {
      const dataUrl = String(reader.result ?? '');
      try {
        localStorage.setItem('rhytm.theme.photo', dataUrl);
      } catch {
        notify('err', 'Не хватило места в хранилище — уберите старое фото');
        return;
      }
      setPhoto(dataUrl);
      onThemeChanged();
      notify('ok', 'Фоновая картинка установлена');
    };
    reader.onerror = () => notify('err', 'Не получилось прочитать файл');
    reader.readAsDataURL(file);
  }

  function removePhoto() {
    if (photo && photo.startsWith('blob:')) URL.revokeObjectURL(photo);
    localStorage.removeItem('rhytm.theme.photo');
    setPhoto(null);
    onThemeChanged();
  }

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
          {theme !== 'custom' && (
            <div className="switch-row" style={{ marginTop: 10 }}>
              <span className="t">Фоновые рисунки</span>
              <button
                className={`toggle${artOn ? ' on' : ''}`}
                role="switch"
                aria-checked={artOn}
                aria-label="Фоновые рисунки"
                onClick={() => onArtChange(!artOn)}
              />
            </div>
          )}
          {theme === 'custom' && (
            <div className="custom-controls show">
              <div className="settings-label" style={{ width: '100%', margin: 0 }}>
                Цвета
              </div>
              {(
                [
                  ['accent', 'Основной цвет'],
                  ['bgA', 'Фон сверху'],
                  ['bgB', 'Фон снизу'],
                  ['text', 'Цвет текста'],
                ] as Array<[keyof CustomPalette, string]>
              ).map(([key, label]) => (
                <label key={key} className="color-row">
                  <span>{label}</span>
                  <input
                    type="color"
                    value={custom[key]}
                    onChange={(e) => setPalette({ [key]: e.target.value })}
                    aria-label={label}
                  />
                </label>
              ))}
              <div className="settings-label" style={{ width: '100%', margin: '4px 0 0' }}>
                Фоновая картинка
              </div>
              {photo && <img className="photo-preview" src={photo} alt="Превью фоновой картинки" />}
              <div className="photo-row">
                <label className="btn-secondary photo-upload">
                  Загрузить
                  <input
                    className="file-hidden"
                    type="file"
                    accept="image/*"
                    onChange={(e) => onPhoto(e.target.files?.[0])}
                    aria-label="Загрузить фоновую картинку"
                  />
                </label>
                {photo && (
                  <button className="btn-secondary" onClick={removePhoto} type="button">
                    Убрать
                  </button>
                )}
              </div>
              <button className="btn-secondary" onClick={resetPalette} type="button">
                Сбросить цвета
              </button>
            </div>
          )}
        </div>

        <div className="settings-section">
          <div className="settings-label">Уведомления</div>
          <PushRow notify={notify} />
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
// apiFetch бросает plain-объект {status,message}: достаём текст причины.
function pushErrorMessage(e: unknown): string {
  if (e instanceof Error && e.message) return e.message;
  if (e && typeof e === 'object' && 'message' in e) {
    const m = (e as { message?: unknown }).message;
    if (typeof m === 'string' && m) return m;
  }
  return 'Не получилось включить уведомления';
}

function PushRow({ notify }: { notify: (k: 'ok' | 'err', t: string) => void }) {
  const [state, setState] = useState<PushState>('off');
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    void currentPushState().then(setState);
  }, []);

  async function toggle() {
    if (busy || state === 'unsupported' || state === 'denied') return;
    setBusy(true);
    setState('busy');
    try {
      if (state === 'on') {
        await disablePush();
        setState('off');
        notify('ok', 'Push-уведомления выключены');
      } else {
        await enablePush();
        setState('on');
        notify('ok', 'Push-уведомления включены');
      }
    } catch (e) {
      setState(await currentPushState());
      notify('err', pushErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  if (state === 'unsupported') {
    return (
      <div className="switch-row">
        <span className="t">Push на телефон недоступен в этом браузере</span>
      </div>
    );
  }

  return (
    <>
      <div className="switch-row">
        <span className="t">Push-уведомления на телефон</span>
        <button
          className={`toggle${state === 'on' ? ' on' : ''}`}
          role="switch"
          aria-checked={state === 'on'}
          aria-label="Push-уведомления на телефон"
          disabled={busy || state === 'busy' || state === 'denied'}
          onClick={() => void toggle()}
        />
      </div>
      {state === 'denied' && (
        <div className="field-error" role="alert">
          Уведомления запрещены — разрешите их в настройках браузера
        </div>
      )}
      {isIOS() && !isStandalone() && state !== 'on' && (
        <div className="state-text">
          На iPhone пуши работают, если добавить «Ритм» на экран «Домой» (Поделиться → На экран «Домой»), iOS 16.4+
        </div>
      )}
    </>
  );
}
