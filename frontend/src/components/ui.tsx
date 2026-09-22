import { useEffect, useRef } from 'react';

export interface Toast {
  id: number;
  kind: 'ok' | 'err';
  text: string;
}

export function Toasts({ items }: { items: Toast[] }) {
  if (items.length === 0) return null;
  return (
    <div className="toasts" role="status" aria-live="polite">
      {items.map((t) => (
        <div key={t.id} className={`toast ${t.kind}`}>
          {t.text}
        </div>
      ))}
    </div>
  );
}

export function Spinner({ label = 'Загрузка…' }: { label?: string }) {
  return (
    <div className="state-block" role="status" aria-label={label}>
      <div className="spinner" aria-hidden="true" />
      <div className="state-text">{label}</div>
    </div>
  );
}

export function ErrorBlock({ text, onRetry }: { text: string; onRetry?: () => void }) {
  return (
    <div className="state-block" role="alert">
      <div className="state-title">Не получилось загрузить</div>
      <div className="state-text">{text}</div>
      {onRetry && (
        <button className="btn-secondary" onClick={onRetry}>
          Повторить
        </button>
      )}
    </div>
  );
}

export function EmptyBlock({ text }: { text: string }) {
  return (
    <div className="state-block">
      <div className="state-title">Пока пусто</div>
      <div className="state-text">{text}</div>
    </div>
  );
}

export function Modal({
  title,
  onClose,
  children,
}: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [onClose]);

  // a11y: фокус на первое поле при открытии.
  const boxRef = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const el = boxRef.current?.querySelector<HTMLElement>('input, textarea, select, button:not(.icon-btn)');
    el?.focus({ preventScroll: true });
  }, []);

  return (
    <div
      className="modal-backdrop"
      onMouseDown={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="modal" ref={boxRef} role="dialog" aria-modal="true" aria-label={title}>
        <div className="modal-head">
          <div className="modal-title">{title}</div>
          <button className="icon-btn" onClick={onClose} aria-label="Закрыть">
            ✕
          </button>
        </div>
        <div className="modal-body">{children}</div>
      </div>
    </div>
  );
}

export function FieldError({ text }: { text?: string }) {
  if (!text) return null;
  return (
    <div className="field-error" role="alert">
      {text}
    </div>
  );
}

export function ConfirmDialog({
  title,
  text,
  confirmLabel = 'Удалить',
  busy = false,
  onCancel,
  onConfirm,
}: {
  title: string;
  text: string;
  confirmLabel?: string;
  busy?: boolean;
  onCancel: () => void;
  onConfirm: () => void;
}) {
  return (
    <Modal title={title} onClose={onCancel}>
      <div className="state-text">{text}</div>
      <div className="modal-actions">
        <div className="spacer" />
        <button className="btn-secondary" onClick={onCancel} type="button" disabled={busy}>
          Отмена
        </button>
        <button className="btn-danger" onClick={onConfirm} type="button" disabled={busy} autoFocus>
          {busy ? 'Подождите…' : confirmLabel}
        </button>
      </div>
    </Modal>
  );
}
