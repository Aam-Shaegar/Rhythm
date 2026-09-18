import { useState } from 'react';
import type { FormEvent } from 'react';
import { useAuth } from '../auth/AuthContext';
import { FieldError } from '../components/ui';
import type { ApiError } from '../api/types';

function friendlyAuthError(e: unknown, fallback: string): string {
  const err = e as Partial<ApiError>;
  if (err && typeof err.status === 'number') {
    if (err.status === 409) return 'Такой пользователь уже существует';
    if (err.status === 401) return 'Неверный логин или пароль';
    if (err.status === 400) return 'Проверьте поля формы';
    if (err.message) return err.message;
  }
  return fallback;
}

export default function AuthScreen({ notify }: { notify: (kind: 'ok' | 'err', text: string) => void }) {
  const { login, register } = useAuth();
  const [mode, setMode] = useState<'login' | 'register'>('login');
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [errors, setErrors] = useState<{ username?: string; email?: string; password?: string }>({});
  const [busy, setBusy] = useState(false);
  const [serverError, setServerError] = useState('');

  function validate(): boolean {
    const next: typeof errors = {};
    if (mode === 'register') {
      if (username.trim().length < 3) next.username = 'Минимум 3 символа';
      else if (username.trim().length > 100) next.username = 'Максимум 100 символов';
      if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email.trim())) next.email = 'Введите корректную почту';
    } else {
      // ТЗ U.8: вход по логину или почте — минимум 3 символа, формат не требуем.
      if (email.trim().length < 3) next.email = 'Введите логин или почту';
    }
    if (password.length < 8) next.password = 'Минимум 8 символов';
    else if (password.length > 72) next.password = 'Максимум 72 символа';
    setErrors(next);
    return Object.keys(next).length === 0;
  }

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setServerError('');
    if (!validate()) return;
    setBusy(true);
    try {
      if (mode === 'login') {
        await login(email.trim(), password);
        notify('ok', 'С возвращением!');
      } else {
        await register(username.trim(), email.trim(), password);
        notify('ok', 'Аккаунт создан');
      }
    } catch (err) {
      const msg = friendlyAuthError(err, 'Не получилось войти');
      setServerError(msg);
      notify('err', msg);
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="screen active" data-screen="auth">
      <div className="screen-head">
        <div className="screen-title">{mode === 'login' ? 'Вход' : 'Регистрация'}</div>
        <div className="pill-toggle" role="tablist" aria-label="Режим входа">
          <button className={mode === 'login' ? 'active' : ''} onClick={() => setMode('login')}>
            Вход
          </button>
          <button className={mode === 'register' ? 'active' : ''} onClick={() => setMode('register')}>
            Регистрация
          </button>
        </div>
      </div>
      <div className="screen-body auth-body">
        <div className="auth-split">
          <div className="auth-hero">
            <div className="auth-hero-mark" aria-hidden="true">
              Р
            </div>
            <div className="auth-hero-title">Ритм</div>
            <div className="auth-hero-sub">Умный органайзер: расписание, задачи и отчёты — в одном окне</div>
            <ul className="auth-hero-list">
              <li>Расписание дня и недели с напоминаниями</li>
              <li>Задачи с повторениями и отметками</li>
              <li>Отчёты о выполнении и статистика</li>
            </ul>
          </div>
          <form className="form-card" onSubmit={onSubmit} noValidate>
          {mode === 'register' && (
            <label className="field">
              <span>Имя пользователя</span>
              <input
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                autoComplete="username"
                placeholder="Например, dmitry"
                minLength={3}
                maxLength={100}
              />
              <FieldError text={errors.username} />
            </label>
          )}
          <label className="field">
            <span>{mode === 'login' ? 'Логин или почта' : 'Электронная почта'}</span>
            <input
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              autoComplete={mode === 'login' ? 'username' : 'email'}
              inputMode={mode === 'login' ? 'text' : 'email'}
              placeholder={mode === 'login' ? 'dmitry или you@example.com' : 'you@example.com'}
            />
            <FieldError text={errors.email} />
          </label>
          <label className="field">
            <span>Пароль</span>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              autoComplete={mode === 'login' ? 'current-password' : 'new-password'}
              placeholder="Минимум 8 символов"
            />
            <FieldError text={errors.password} />
          </label>
          {serverError && (
            <div className="form-error" role="alert">
              {serverError}
            </div>
          )}
          <button className="btn-primary" type="submit" disabled={busy}>
            {busy ? 'Подождите…' : mode === 'login' ? 'Войти' : 'Создать аккаунт'}
          </button>
          </form>
        </div>
      </div>
    </div>
  );
}
