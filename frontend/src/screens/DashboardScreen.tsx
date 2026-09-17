import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiFetch } from '../api/client';
import type { DailyReport, EventItem, Task } from '../api/types';
import type { Tab } from '../nav';
import { EmptyBlock, ErrorBlock, Spinner } from '../components/ui';

function startOfDay(d: Date): Date {
  const c = new Date(d);
  c.setHours(0, 0, 0, 0);
  return c;
}
function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}
function fmtTime(iso: string): string {
  return new Date(iso).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' });
}
function fmtDay(iso: string): string {
  return new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' });
}

export default function DashboardScreen({
  go,
  notify,
}: {
  go: (t: Tab) => void;
  notify: (k: 'ok' | 'err', t: string) => void;
}) {
  const [tasks, setTasks] = useState<Task[]>([]);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [report, setReport] = useState<DailyReport | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toggling, setToggling] = useState<string | null>(null);

  const loadEvents = async (from: string, to: string): Promise<EventItem[]> => {
    try {
      return await apiFetch<EventItem[]>(
        `/events?date_from=${encodeURIComponent(from)}&date_to=${encodeURIComponent(to)}`,
      );
    } catch {
      return apiFetch<EventItem[]>('/events');
    }
  };

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const today = new Date().toISOString().slice(0, 10);
      const from = startOfDay(new Date()).toISOString();
      const to = new Date(Date.now() + 7 * 864e5).toISOString();
      const [t, e, r] = await Promise.all([
        apiFetch<Task[]>('/tasks'),
        loadEvents(from, to),
        apiFetch<DailyReport>(`/reports/daily?date=${encodeURIComponent(today)}`),
      ]);
      setTasks(t);
      setEvents(e);
      setReport(r);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Ошибка загрузки');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const now = useMemo(() => new Date(), []);
  const todayTasks = useMemo(() => {
    const t = new Date();
    return tasks
      .filter((x) => sameDay(new Date(x.due_at), t))
      .sort((a, b) => +new Date(a.due_at) - +new Date(b.due_at));
  }, [tasks]);
  const overdue = useMemo(
    () => tasks.filter((x) => !x.is_completed && new Date(x.due_at).getTime() < startOfDay(now).getTime()),
    [tasks, now],
  );
  const upcoming = useMemo(
    () =>
      [...events]
        .filter((e) => new Date(e.start_at).getTime() >= startOfDay(now).getTime())
        .sort((a, b) => +new Date(a.start_at) - +new Date(b.start_at))
        .slice(0, 5),
    [events, now],
  );
  const doneToday = todayTasks.filter((t) => t.is_completed).length;

  async function toggle(t: Task) {
    if (toggling) return;
    setToggling(t.id);
    try {
      if (t.is_completed) {
        await apiFetch<Task>(`/tasks/${t.id}`, { method: 'PATCH', body: JSON.stringify({ is_completed: false }) });
        notify('ok', 'Задача снова в работе');
      } else {
        await apiFetch<Task>(`/tasks/${t.id}/complete`, { method: 'PATCH' });
        notify('ok', 'Задача выполнена');
      }
      await load();
    } catch (e) {
      notify('err', e instanceof Error ? e.message : 'Не получилось обновить');
    } finally {
      setToggling(null);
    }
  }

  const pct = report && report.total_tasks > 0 ? Math.round(report.completion_pct) : 0;

  return (
    <div className="screen active" data-screen="home">
      <div className="screen-head">
        <div>
          <div className="screen-title">Главная</div>
          <div className="screen-sub">
            {new Date().toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long' })}
          </div>
        </div>
      </div>
      <div className="screen-body">
        {loading ? (
          <Spinner />
        ) : error ? (
          <ErrorBlock text={error} onRetry={() => void load()} />
        ) : (
          <>
            <div className="stat-row dash-stats">
              <div className="stat-card">
                <div className="n">{doneToday}/{todayTasks.length}</div>
                <div className="l">Задачи сегодня</div>
              </div>
              <div className="stat-card">
                <div className="n">{pct}%</div>
                <div className="l">Выполнение</div>
              </div>
              <div className="stat-card">
                <div className="n">{upcoming.length}</div>
                <div className="l">Событий рядом</div>
              </div>
            </div>

            {overdue.length > 0 && (
              <section aria-label="Просроченные задачи">
                <div className="settings-label" style={{ marginTop: 4 }}>
                  Требуют внимания · {overdue.length}
                </div>
                {overdue.slice(0, 3).map((t) => (
                  <div key={t.id} className="task-card attention">
                    <button
                      className="check"
                      role="checkbox"
                      aria-checked={false}
                      aria-label="Отметить выполненной"
                      disabled={toggling === t.id}
                      onClick={() => void toggle(t)}
                    />
                    <button className="task-main" onClick={() => go('tasks')} title="К задачам">
                      <span className="task-title">{t.title}</span>
                      <span className="task-meta">
                        <span className="overdue">Просрочена · {fmtDay(t.due_at)}, {fmtTime(t.due_at)}</span>
                      </span>
                    </button>
                  </div>
                ))}
              </section>
            )}

            <div className="dash-cols">
              <div>
                <section aria-label="Задачи на сегодня">
                  <div className="row-head">
                    <div className="settings-label" style={{ margin: 0 }}>
                      Сегодня
                    </div>
                    <button className="link-btn" onClick={() => go('tasks')}>
                      Все задачи →
                    </button>
                  </div>
                  {todayTasks.length === 0 ? (
                    <EmptyBlock text="На сегодня задач нет. Хороший день, чтобы спланировать завтра." />
                  ) : (
                    todayTasks.slice(0, 5).map((t) => (
                      <div key={t.id} className={`task-card${t.is_completed ? ' done' : ''}`}>
                        <button
                          className="check"
                          role="checkbox"
                          aria-checked={t.is_completed}
                          aria-label={t.is_completed ? 'Вернуть в работу' : 'Отметить выполненной'}
                          disabled={toggling === t.id}
                          onClick={() => void toggle(t)}
                        />
                        <button className="task-main" onClick={() => go('tasks')} title="К задачам">
                          <span className="task-title">{t.title}</span>
                          <span className="task-meta">
                            <span>{fmtTime(t.due_at)}</span>
                          </span>
                        </button>
                      </div>
                    ))
                  )}
                </section>
              </div>
              <div>
                <section aria-label="Ближайшие события">
                  <div className="row-head">
                    <div className="settings-label" style={{ margin: 0 }}>
                      Ближайшие события
                    </div>
                    <button className="link-btn" onClick={() => go('calendar')}>
                      Расписание →
                    </button>
                  </div>
                  {upcoming.length === 0 ? (
                    <EmptyBlock text="Ближайших событий нет." />
                  ) : (
                    <div className="dash-events">
                      {upcoming.map((e) => (
                        <button key={e.id} className="dash-event" onClick={() => go('calendar')} title={e.title}>
                          <span className="de-date">
                            {fmtDay(e.start_at)}
                            <b>{fmtTime(e.start_at)}</b>
                          </span>
                          <span className="de-title">{e.title}</span>
                        </button>
                      ))}
                    </div>
                  )}
                </section>

                <div className="dash-actions">
                  <button className="btn-primary" onClick={() => go('tasks')}>
                    + Задача
                  </button>
                  <button className="btn-secondary" onClick={() => go('calendar')}>
                    + Событие
                  </button>
                  <button className="btn-secondary" onClick={() => go('reports')}>
                    Отчёты
                  </button>
                </div>
              </div>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
