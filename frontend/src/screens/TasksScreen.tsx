import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiFetch } from '../api/client';
import type { RecurrenceType, Task } from '../api/types';
import { EmptyBlock, ErrorBlock, ConfirmDialog, FieldError, Modal, Spinner } from '../components/ui';

type Filter = 'all' | 'active' | 'done';

function toLocalInputValue(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}

function fmtDue(iso: string): string {
  const d = new Date(iso);
  const today = new Date();
  const sameDay =
    d.getFullYear() === today.getFullYear() && d.getMonth() === today.getMonth() && d.getDate() === today.getDate();
  const tm = d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' });
  const hm = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
  return `${sameDay ? 'Сегодня' : tm}, ${hm}`;
}

const REC_LABEL: Record<string, string> = {
  daily: 'ежедневно',
  weekly: 'еженедельно',
  monthly: 'ежемесячно',
  yearly: 'ежегодно',
};

function isOverdue(iso: string, now = new Date()): boolean {
  const day = new Date(now);
  day.setHours(0, 0, 0, 0);
  return new Date(iso).getTime() < day.getTime();
}

interface ModalState {
  mode: 'create' | 'edit';
  task?: Task;
}

export default function TasksScreen({ notify }: { notify: (k: 'ok' | 'err', t: string) => void }) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [filter, setFilter] = useState<Filter>('all');
  const [query, setQuery] = useState('');
  const [modal, setModal] = useState<ModalState | null>(null);
  const [toggling, setToggling] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const list = await apiFetch<Task[]>('/tasks');
      setTasks(list);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка загрузки');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return tasks
      .filter((t) => (filter === 'all' ? true : filter === 'done' ? t.is_completed : !t.is_completed))
      .filter((t) => (q ? t.title.toLowerCase().includes(q) || (t.description ?? '').toLowerCase().includes(q) : true))
      .sort((a, b) => +new Date(a.due_at) - +new Date(b.due_at));
  }, [tasks, filter, query]);

  async function toggleComplete(t: Task) {
    if (toggling) return;
    setToggling(t.id);
    try {
      if (t.is_completed) {
        await apiFetch<Task>(`/tasks/${t.id}`, { method: 'PATCH', body: JSON.stringify({ is_completed: false }) });
      } else {
        await apiFetch<Task>(`/tasks/${t.id}/complete`, { method: 'PATCH' });
      }
      await load();
      notify('ok', t.is_completed ? 'Задача снова в работе' : 'Задача выполнена');
    } catch (e) {
      notify('err', e instanceof Error ? e.message : 'Не получилось обновить');
    } finally {
      setToggling(null);
    }
  }

  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);

  async function removeTask(id: string) {
    setDeleting(true);
    try {
      await apiFetch(`/tasks/${id}`, { method: 'DELETE' });
      notify('ok', 'Задача удалена');
      setConfirmDelete(false);
      setModal(null);
      await load();
    } catch (e) {
      notify('err', e instanceof Error ? e.message : 'Не получилось удалить');
    } finally {
      setDeleting(false);
    }
  }

  return (
    <div className="screen active" data-screen="tasks">
      <div className="screen-head">
        <div className="screen-title">Задачи</div>
        <div className="pill-toggle" role="tablist" aria-label="Фильтр задач">
          {(['all', 'active', 'done'] as Filter[]).map((f) => (
            <button key={f} className={filter === f ? 'active' : ''} onClick={() => setFilter(f)}>
              {f === 'all' ? 'Все' : f === 'active' ? 'Активные' : 'Готовые'}
            </button>
          ))}
        </div>
      </div>
      <div className="toolbar">
        <input
          className="search"
          placeholder="Поиск задач…"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          aria-label="Поиск задач"
        />
      </div>
      <div className="screen-body">
        {loading ? (
          <Spinner />
        ) : error ? (
          <ErrorBlock text={error} onRetry={() => void load()} />
        ) : visible.length === 0 ? (
          <EmptyBlock text={query ? 'Ничего не найдено. Попробуйте другой запрос.' : 'Задач пока нет. Нажмите «+», чтобы создать.'} />
        ) : (
          visible.map((t) => (
            <div key={t.id} className={`task-card${t.is_completed ? ' done' : ''}`}>
              <button
                className="check"
                role="checkbox"
                aria-checked={t.is_completed}
                aria-label={t.is_completed ? 'Вернуть в работу' : 'Отметить выполненной'}
                onClick={() => void toggleComplete(t)}
                disabled={toggling === t.id}
              />
              <button className="task-main" onClick={() => setModal({ mode: 'edit', task: t })} title="Открыть задачу">
                <span className="task-title">{t.title}</span>
                <span className="task-meta">
                  <span>{fmtDue(t.due_at)}</span>
                  {!t.is_completed && isOverdue(t.due_at) && <span className="overdue">Просрочена</span>}
                  {t.recurrence_type && <span> · повтор {REC_LABEL[t.recurrence_type] ?? t.recurrence_type}</span>}
                </span>
                {t.description && <span className="task-desc">{t.description}</span>}
              </button>
            </div>
          ))
        )}
      </div>
      <button className="fab" aria-label="Добавить задачу" title="Добавить задачу" onClick={() => setModal({ mode: 'create' })}>
        +
      </button>
      {modal && (
        <TaskModal
          initial={modal.task}
          onClose={() => setModal(null)}
          onSaved={async () => {
            setModal(null);
            await load();
          }}
          onDelete={modal.task ? () => setConfirmDelete(true) : undefined}
          notify={notify}
        />
      )}
      {confirmDelete && modal?.task && (
        <ConfirmDialog
          title="Удалить задачу?"
          text={`«${modal.task.title}» будет удалена без возможности восстановления.`}
          busy={deleting}
          onCancel={() => setConfirmDelete(false)}
          onConfirm={() => void removeTask(modal.task!.id)}
        />
      )}
    </div>
  );
}

function TaskModal({
  initial,
  onClose,
  onSaved,
  onDelete,
  notify,
}: {
  initial?: Task;
  onClose: () => void;
  onSaved: () => Promise<void>;
  onDelete?: () => void;
  notify: (k: 'ok' | 'err', t: string) => void;
}) {
  const [title, setTitle] = useState(initial?.title ?? '');
  const [desc, setDesc] = useState(initial?.description ?? '');
  const [due, setDue] = useState(() =>
    toLocalInputValue(initial ? new Date(initial.due_at) : new Date(Date.now() + 60 * 60 * 1000)),
  );
  const [rec, setRec] = useState<'' | RecurrenceType>((initial?.recurrence_type as RecurrenceType) ?? '');
  const [recEnd, setRecEnd] = useState(initial?.recurrence_end ? toLocalInputValue(new Date(initial.recurrence_end)).slice(0, 16) : '');
  const [err, setErr] = useState('');
  const [busy, setBusy] = useState(false);

  async function save() {
    setErr('');
    if (title.trim().length < 1 || title.trim().length > 200) {
      setErr('Название: от 1 до 200 символов');
      return;
    }
    if (desc.length > 2000) {
      setErr('Описание: максимум 2000 символов');
      return;
    }
    const dueDate = new Date(due);
    if (Number.isNaN(+dueDate)) {
      setErr('Укажите корректный срок');
      return;
    }
    let recEndIso: string | null = null;
    if (rec) {
      if (!recEnd) {
        setErr('Для повтора укажите дату окончания');
        return;
      }
      const re = new Date(recEnd);
      if (Number.isNaN(+re) || re <= dueDate) {
        setErr('Окончание повтора должно быть позже срока');
        return;
      }
      recEndIso = re.toISOString();
    }
    setBusy(true);
    try {
      const payload: Record<string, unknown> = {
        title: title.trim(),
        description: desc.trim() ? desc.trim() : null,
        due_at: dueDate.toISOString(),
        recurrence_type: rec || null,
        recurrence_end: recEndIso,
      };
      if (initial) {
        await apiFetch(`/tasks/${initial.id}`, { method: 'PATCH', body: JSON.stringify(payload) });
        notify('ok', 'Задача обновлена');
      } else {
        await apiFetch('/tasks', { method: 'POST', body: JSON.stringify(payload) });
        notify('ok', 'Задача создана');
      }
      await onSaved();
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Не получилось сохранить';
      setErr(msg);
      notify('err', msg);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal title={initial ? 'Задача' : 'Новая задача'} onClose={onClose}>
      <label className="field">
        <span>Название</span>
        <input value={title} onChange={(e) => setTitle(e.target.value)} maxLength={200} placeholder="Что нужно сделать?" />
      </label>
      <label className="field">
        <span>Срок</span>
        <input type="datetime-local" value={due} onChange={(e) => setDue(e.target.value)} />
      </label>
      <label className="field">
        <span>Описание (необязательно)</span>
        <textarea value={desc} onChange={(e) => setDesc(e.target.value)} rows={3} maxLength={2000} />
      </label>
      <div className="field-row">
        <label className="field">
          <span>Повтор</span>
          <select value={rec} onChange={(e) => setRec(e.target.value as '' | RecurrenceType)}>
            <option value="">Без повтора</option>
            <option value="daily">Ежедневно</option>
            <option value="weekly">Еженедельно</option>
            <option value="monthly">Ежемесячно</option>
            <option value="yearly">Ежегодно</option>
          </select>
        </label>
        <label className="field">
          <span>Окончание повтора</span>
          <input type="datetime-local" value={recEnd} onChange={(e) => setRecEnd(e.target.value)} disabled={!rec} />
        </label>
      </div>
      <FieldError text={err} />
      <div className="modal-actions">
        {initial && onDelete && (
          <button className="btn-danger" onClick={onDelete} type="button">
            Удалить
          </button>
        )}
        <div className="spacer" />
        <button className="btn-secondary" onClick={onClose} type="button">
          Отмена
        </button>
        <button className="btn-primary" onClick={() => void save()} disabled={busy} type="button">
          {busy ? 'Сохранение…' : 'Сохранить'}
        </button>
      </div>
    </Modal>
  );
}
