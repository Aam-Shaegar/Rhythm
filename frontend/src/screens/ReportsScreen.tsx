import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiFetch } from '../api/client';
import type { DailyReport } from '../api/types';
import { EmptyBlock, ErrorBlock, Spinner } from '../components/ui';

function localDate(d: Date): string { return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`; }

function isoDate(d: Date): string {
  return localDate(d);
}

export default function ReportsScreen() {
  const [mode, setMode] = useState<'day' | 'period'>(() => (localStorage.getItem('rhytm.rep.mode') as 'day' | 'period') || 'day');
  const [date, setDate] = useState(() => isoDate(new Date()));
  const [from, setFrom] = useState(() => isoDate(new Date(Date.now() - 6 * 864e5)));
  const [to, setTo] = useState(() => isoDate(new Date()));
  const [daily, setDaily] = useState<DailyReport | null>(null);
  const [period, setPeriod] = useState<DailyReport[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    localStorage.setItem('rhytm.rep.mode', mode);
  }, [mode]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      if (mode === 'day') {
        const r = await apiFetch<DailyReport>(`/reports/daily?date=${encodeURIComponent(date)}`);
        setDaily(r);
      } else {
        if (new Date(from) > new Date(to)) {
          setError('Начало периода позже окончания');
          setPeriod([]);
          return;
        }
        const days = Math.round((+new Date(to) - +new Date(from)) / 864e5);
        if (days > 62) {
          setError('Период не больше 62 дней');
          setPeriod([]);
          return;
        }
        const list = await apiFetch<DailyReport[]>(
          `/reports/period?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`,
        );
        setPeriod(list);
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка загрузки');
    } finally {
      setLoading(false);
    }
  }, [mode, date, from, to]);

  useEffect(() => {
    void load();
    const refresh = () => void load();
    window.addEventListener('online', refresh);
    return () => window.removeEventListener('online', refresh);
  }, [load]);

  const agg = useMemo(() => {
    if (mode !== 'period' || period.length === 0) return null;
    const total = period.reduce((s, r) => s + r.total_tasks, 0);
    const done = period.reduce((s, r) => s + r.completed_tasks, 0);
    const late = period.reduce((s, r) => s + r.completed_late, 0);
    const overdue = period.reduce((s, r) => s + r.overdue_tasks, 0);
    const ev = period.reduce((s, r) => s + r.events_count, 0);
    return { total, done, late, overdue, pct: total ? (done / total) * 100 : 0, ev };
  }, [mode, period]);

  const shown: { total: number; done: number; pct: number; events: number; late: number; overdue: number } | null = useMemo(() => {
    if (mode === 'day') {
      if (!daily) return null;
      return { total: daily.total_tasks, done: daily.completed_tasks, pct: daily.completion_pct, events: daily.events_count, late: daily.completed_late, overdue: daily.overdue_tasks };
    }
    if (!agg) return null;
    return { total: agg.total, done: agg.done, pct: agg.pct, events: agg.ev, late: agg.late, overdue: agg.overdue };
  }, [mode, daily, agg]);

  const pct = Math.max(0, Math.min(100, Math.round(shown?.pct ?? 0)));

  return (
    <div className="screen active" data-screen="reports">
      <div className="screen-head">
        <div className="screen-title">Отчёты</div>
        <div className="pill-toggle" role="tablist" aria-label="Тип отчёта">
          <button className={mode === 'day' ? 'active' : ''} onClick={() => setMode('day')}>
            День
          </button>
          <button className={mode === 'period' ? 'active' : ''} onClick={() => setMode('period')}>
            Период
          </button>
        </div>
      </div>
      <div className="screen-body">
        {mode === 'day' ? (
          <label className="field">
            <span>Дата</span>
            <input type="date" value={date} max={isoDate(new Date())} onChange={(e) => setDate(e.target.value)} />
          </label>
        ) : (
          <div className="field-row">
            <label className="field">
              <span>С</span>
              <input type="date" value={from} onChange={(e) => setFrom(e.target.value)} />
            </label>
            <label className="field">
              <span>По</span>
              <input type="date" value={to} onChange={(e) => setTo(e.target.value)} />
            </label>
          </div>
        )}

        {loading ? (
          <Spinner />
        ) : error ? (
          <ErrorBlock text={error} onRetry={() => void load()} />
        ) : !shown || (shown.total === 0 && shown.events === 0) ? (
          <EmptyBlock text="Данных за выбранный период нет. Выполняйте задачи — и статистика появится." />
        ) : (
          <>
            <div className="donut-wrap">
              <div className="donut" style={{ background: `conic-gradient(var(--accent-deep) 0% ${pct}%, var(--surface-strong) ${pct}% 100%)` }}>
                <div className="donut-num">{pct}%</div>
              </div>
            </div>
            <div className="stat-row">
              <div className="stat-card">
                <div className="n">{shown.done}</div>
                <div className="l">Выполнено</div>
              </div>
              <div className="stat-card">
                <div className="n">{shown.total - shown.done}</div>
                <div className="l">Не выполнено</div>
              </div>
              <div className="stat-card">
                <div className="n">{shown.events}</div>
                <div className="l">Событий</div>
              </div>
            </div>
            <div className="stat-row report-deadlines">
              <div className="stat-card"><div className="n">{shown.done - shown.late}</div><div className="l">Выполнено вовремя</div></div>
              <div className="stat-card"><div className="n overdue">{shown.late}</div><div className="l">Выполнено с опозданием</div></div>
              <div className="stat-card"><div className="n overdue">{shown.overdue}</div><div className="l">Просрочено, не выполнено</div></div>
            </div>
            <p className="report-note">Общий процент учитывает все выполненные задачи. Соблюдение сроков: {shown.total ? Math.round((shown.done - shown.late) / shown.total * 100) : 0}%. Задачи сгруппированы по дате дедлайна.</p>
            {mode === 'period' && (
              <div className="period-list">
                {period.map((r) => (
                  <div key={r.date} className="period-row">
                    <span>{new Date(r.date).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })}</span>
                    <span className="period-bar">
                      <span style={{ width: `${Math.round(r.completion_pct)}%` }} />
                    </span>
                    <span className="period-num">
                      {r.completed_tasks}/{r.total_tasks} · опоздали: {r.completed_late}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
