import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiFetch } from '../api/client';
import type { EventItem } from '../api/types';
import { EmptyBlock, ErrorBlock, ConfirmDialog, Modal, Spinner, FieldError } from '../components/ui';

const HOURS_START = 0;
const HOURS_END = 24;
const HOURS_COUNT = HOURS_END - HOURS_START;

const DAY_LABELS = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс'];
const MONTHS = [
  'января', 'февраля', 'марта', 'апреля', 'мая', 'июня',
  'июля', 'августа', 'сентября', 'октября', 'ноября', 'декабря',
];

function startOfDay(d: Date): Date {
  const c = new Date(d);
  c.setHours(0, 0, 0, 0);
  return c;
}
function endOfDay(d: Date): Date {
  const c = startOfDay(d);
  c.setDate(c.getDate() + 1);
  return c;
}
function mondayOf(d: Date): Date {
  const c = startOfDay(d);
  const dow = (c.getDay() + 6) % 7; // Mon=0
  c.setDate(c.getDate() - dow);
  return c;
}
function addDays(d: Date, n: number): Date {
  const c = new Date(d);
  c.setDate(c.getDate() + n);
  return c;
}
function sameDay(a: Date, b: Date): boolean {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}
function toLocalInputValue(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}
function fmtDayTitle(d: Date): string {
  return `${d.getDate()} ${MONTHS[d.getMonth()]}`;
}

interface PlacedEvent extends EventItem {
  topPx: number;
  heightPx: number;
  leftPct: number;
  widthPct: number;
}

function layoutDay(events: EventItem[], hourHeight: number): PlacedEvent[] {
  const sorted = [...events].sort((a, b) => +new Date(a.start_at) - +new Date(b.start_at));
  const lanes: EventItem[][] = [];
  const laneOf = new Map<string, number>();
  for (const ev of sorted) {
    const s = +new Date(ev.start_at);
    const e = +new Date(ev.end_at);
    let lane = 0;
    while (true) {
      const conflict = (lanes[lane] ?? []).some((o) => {
        const os = +new Date(o.start_at);
        const oe = +new Date(o.end_at);
        return s < oe && e > os;
      });
      if (!conflict) break;
      lane += 1;
    }
    if (!lanes[lane]) lanes[lane] = [];
    lanes[lane].push(ev);
    laneOf.set(ev.id, lane);
  }
  const laneCount = Math.max(1, lanes.length);
  return sorted.map((ev) => {
    const start = new Date(ev.start_at);
    const end = new Date(ev.end_at);
    const startMin = start.getHours() * 60 + start.getMinutes() - HOURS_START * 60;
    const durMin = Math.max(20, (end.getTime() - start.getTime()) / 60000);
    const lane = laneOf.get(ev.id) ?? 0;
    return {
      ...ev,
      topPx: (startMin / 60) * hourHeight,
      heightPx: Math.max(24, (durMin / 60) * hourHeight),
      leftPct: (lane / laneCount) * 100,
      widthPct: 100 / laneCount,
    };
  });
}

interface ModalState {
  mode: 'create' | 'edit';
  event?: EventItem;
}

export default function ScheduleScreen({ notify }: { notify: (k: 'ok' | 'err', t: string) => void }) {
  const [view, setView] = useState<'day' | 'week'>(() => (localStorage.getItem('rhytm.sched.view') as 'day' | 'week') || 'day');
  const [selected, setSelected] = useState<Date>(() => new Date());
  const [hourHeight, setHourHeight] = useState<number>(() => Number(localStorage.getItem('rhytm.sched.zoom') ?? 56) || 56);
  const [events, setEvents] = useState<EventItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [modal, setModal] = useState<ModalState | null>(null);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [now, setNow] = useState(() => new Date());

  useEffect(() => {
    localStorage.setItem('rhytm.sched.view', view);
  }, [view]);
  useEffect(() => {
    localStorage.setItem('rhytm.sched.zoom', String(hourHeight));
  }, [hourHeight]);
  useEffect(() => {
    const t = setInterval(() => setNow(new Date()), 60000);
    return () => clearInterval(t);
  }, []);

  const range = useMemo(() => {
    if (view === 'day') return { from: startOfDay(selected), to: endOfDay(selected) };
    const m = mondayOf(selected);
    return { from: m, to: addDays(m, 7) };
  }, [view, selected]);

  const load = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const q = `?date_from=${encodeURIComponent(range.from.toISOString())}&date_to=${encodeURIComponent(range.to.toISOString())}`;
      let list: EventItem[] = [];
      try {
        list = await apiFetch<EventItem[]>(`/events${q}`);
      } catch {
        list = await apiFetch<EventItem[]>('/events');
      }
      // Страховка, если сервер проигнорирует диапазон.
      const f = list.filter((e) => {
        const s = new Date(e.start_at).getTime();
        return s >= range.from.getTime() && s < range.to.getTime();
      });
      // Фильтр всё вырезал, а сервер что-то вернул — показываем как есть.
      setEvents(list.length > 0 && f.length === 0 ? list : f);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Ошибка загрузки');
    } finally {
      setLoading(false);
    }
  }, [range]);

  useEffect(() => {
    void load();
  }, [load]);

  const weekDays = useMemo(() => {
    const m = mondayOf(selected);
    return Array.from({ length: 7 }, (_, i) => addDays(m, i));
  }, [selected]);

  const dayEvents = useMemo(() => {
    if (view !== 'day') return [];
    return layoutDay(
      events.filter((e) => sameDay(new Date(e.start_at), selected)),
      hourHeight,
    );
  }, [events, selected, view, hourHeight]);

  const weekPlaced = useMemo(() => {
    if (view !== 'week') return [];
    return weekDays.map((d) =>
      layoutDay(
        events.filter((e) => sameDay(new Date(e.start_at), d)),
        hourHeight,
      ),
    );
  }, [events, weekDays, view, hourHeight]);

  const nowOffsetPx = useMemo(() => {
    const mins = now.getHours() * 60 + now.getMinutes() - HOURS_START * 60;
    return (mins / 60) * hourHeight;
  }, [now, hourHeight]);

  const showNowLine = view === 'day' && sameDay(now, selected);

  async function removeEvent(id: string) {
    setDeleting(true);
    try {
      await apiFetch(`/events/${id}`, { method: 'DELETE' });
      notify('ok', 'Событие удалено');
      setConfirmDelete(false);
      setModal(null);
      await load();
    } catch (e) {
      notify('err', e instanceof Error ? e.message : 'Не получилось удалить');
    } finally {
      setDeleting(false);
    }
  }

  const hours = Array.from({ length: HOURS_COUNT }, (_, i) => HOURS_START + i);

  return (
    <div className="screen active" data-screen="calendar">
      <div className="screen-head">
        <div className="screen-title">Расписание</div>
        <div className="pill-toggle" role="tablist" aria-label="Вид расписания">
          <button className={view === 'day' ? 'active' : ''} onClick={() => setView('day')}>
            День
          </button>
          <button className={view === 'week' ? 'active' : ''} onClick={() => setView('week')}>
            Неделя
          </button>
        </div>
      </div>

      <div className="date-nav">
        <button className="nav-btn" aria-label="Предыдущий период" onClick={() => setSelected((d) => addDays(d, view === 'day' ? -1 : -7))}>
          ‹
        </button>
        <button className="nav-title" onClick={() => setSelected(new Date())} title="Сегодня">
          {view === 'day' ? fmtDayTitle(selected) : `${fmtDayTitle(weekDays[0])} — ${fmtDayTitle(weekDays[6])}`}
        </button>
        <button className="nav-btn" aria-label="Следующий период" onClick={() => setSelected((d) => addDays(d, view === 'day' ? 1 : 7))}>
          ›
        </button>
      </div>

      <div className="date-strip" role="listbox" aria-label="Дни недели">
        {weekDays.map((d) => {
          const active = sameDay(d, selected);
          const isToday = sameDay(d, new Date());
          return (
            <button
              key={d.toISOString()}
              className={`date-chip${active ? ' active' : ''}${isToday ? ' today' : ''}`}
              onClick={() => setSelected(d)}
              aria-selected={active}
            >
              <span className="d">{DAY_LABELS[(d.getDay() + 6) % 7]}</span>
              <span className="n">{d.getDate()}</span>
            </button>
          );
        })}
      </div>

      <div className="zoom-row">
        <span aria-hidden="true">−</span>
        <input
          type="range"
          className="zoom"
          aria-label="Масштаб сетки"
          min={32}
          max={88}
          step={4}
          value={hourHeight}
          onChange={(e) => setHourHeight(Number(e.target.value))}
        />
        <span aria-hidden="true">+</span>
      </div>

      <div className="screen-body sched-body">
        {loading ? (
          <Spinner />
        ) : error ? (
          <ErrorBlock text={error} onRetry={() => void load()} />
        ) : view === 'day' ? (
          dayEvents.length === 0 ? (
            <EmptyBlock text="На этот день событий нет. Нажмите «+», чтобы добавить." />
          ) : null
        ) : events.length === 0 ? (
          <EmptyBlock text="На этой неделе событий нет. Нажмите «+», чтобы добавить." />
        ) : null}

        {!loading && !error && view === 'day' && (
          <div className="cal-wrap">
            <div className="hour-col" aria-hidden="true">
              {hours.map((h) => (
                <div key={h} className="hour-label" style={{ top: (h - HOURS_START) * hourHeight }}>
                  {h}:00
                </div>
              ))}
            </div>
            <div className="grid-body" style={{ height: HOURS_COUNT * hourHeight }}>
              {dayEvents.map((ev) => (
                <button
                  key={ev.id}
                  className="event"
                  style={{ top: ev.topPx, height: ev.heightPx, left: `calc(${ev.leftPct}% + 5px)`, width: `calc(${ev.widthPct}% - 10px)` }}
                  onClick={() => setModal({ mode: 'edit', event: ev })}
                  title={`${ev.title}`}
                >
                  {ev.title}
                  <span>
                    {new Date(ev.start_at).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}–
                    {new Date(ev.end_at).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })}
                  </span>
                </button>
              ))}
              {showNowLine && <div className="now-line" style={{ top: nowOffsetPx }} />}
            </div>
          </div>
        )}

        {!loading && !error && view === 'week' && (
          <div className="cal-wrap">
            <div className="hour-col" aria-hidden="true">
              {hours.map((h) => (
                <div key={h} className="hour-label" style={{ top: (h - HOURS_START) * hourHeight }}>
                  {h}:00
                </div>
              ))}
            </div>
            <div className="week-scroll">
              <div className="week-heads">
                {weekDays.map((d) => (
                  <span key={d.toISOString()} className={sameDay(d, new Date()) ? 'wh-today' : ''}>
                    {DAY_LABELS[(d.getDay() + 6) % 7]} {d.getDate()}
                  </span>
                ))}
              </div>
              <div className="week-grid">
                {weekPlaced.map((col, i) => (
                  <div key={weekDays[i].toISOString()} className="week-col" style={{ height: HOURS_COUNT * hourHeight }}>
                    {col.map((ev) => (
                      <button
                        key={ev.id}
                        className="event"
                        style={{ top: ev.topPx, height: Math.max(20, ev.heightPx) }}
                        onClick={() => setModal({ mode: 'edit', event: ev })}
                        title={ev.title}
                      >
                        {ev.title}
                      </button>
                    ))}
                  </div>
                ))}
              </div>
            </div>
          </div>
        )}
      </div>

      <button className="fab" aria-label="Добавить событие" title="Добавить событие" onClick={() => setModal({ mode: 'create' })}>
        +
      </button>

      {modal && (
        <EventModal
          initial={modal.event}
          selectedDate={selected}
          onClose={() => setModal(null)}
          onSaved={async () => {
            setModal(null);
            await load();
          }}
          onDelete={modal.event ? () => setConfirmDelete(true) : undefined}
          notify={notify}
        />
      )}
      {confirmDelete && modal?.event && (
        <ConfirmDialog
          title="Удалить событие?"
          text={`«${modal.event.title}» будет удалено без возможности восстановления.`}
          busy={deleting}
          onCancel={() => setConfirmDelete(false)}
          onConfirm={() => void removeEvent(modal.event!.id)}
        />
      )}
    </div>
  );
}

function EventModal({
  initial,
  selectedDate,
  onClose,
  onSaved,
  onDelete,
  notify,
}: {
  initial?: EventItem;
  selectedDate: Date;
  onClose: () => void;
  onSaved: () => Promise<void>;
  onDelete?: () => void;
  notify: (k: 'ok' | 'err', t: string) => void;
}) {
  const defStart = initial ? new Date(initial.start_at) : new Date(selectedDate.getFullYear(), selectedDate.getMonth(), selectedDate.getDate(), 10, 0);
  const defEnd = initial ? new Date(initial.end_at) : new Date(selectedDate.getFullYear(), selectedDate.getMonth(), selectedDate.getDate(), 11, 0);
  const [title, setTitle] = useState(initial?.title ?? '');
  const [desc, setDesc] = useState(initial?.description ?? '');
  const [start, setStart] = useState(toLocalInputValue(defStart));
  const [end, setEnd] = useState(toLocalInputValue(defEnd));
  const [fieldErr, setFieldErr] = useState('');
  const [busy, setBusy] = useState(false);

  async function save() {
    setFieldErr('');
    if (title.trim().length < 1 || title.trim().length > 200) {
      setFieldErr('Название: от 1 до 200 символов');
      return;
    }
    const s = new Date(start);
    const e = new Date(end);
    if (Number.isNaN(+s) || Number.isNaN(+e)) {
      setFieldErr('Укажите корректные дату и время');
      return;
    }
    if (e <= s) {
      setFieldErr('Окончание должно быть позже начала');
      return;
    }
    if (desc.length > 2000) {
      setFieldErr('Описание: максимум 2000 символов');
      return;
    }
    setBusy(true);
    try {
      const payload = {
        title: title.trim(),
        description: desc.trim() ? desc.trim() : null,
        start_at: s.toISOString(),
        end_at: e.toISOString(),
      };
      if (initial) {
        await apiFetch(`/events/${initial.id}`, { method: 'PATCH', body: JSON.stringify(payload) });
        notify('ok', 'Событие обновлено');
      } else {
        await apiFetch('/events', { method: 'POST', body: JSON.stringify(payload) });
        notify('ok', 'Событие создано');
      }
      await onSaved();
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Не получилось сохранить';
      setFieldErr(msg);
      notify('err', msg);
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal title={initial ? 'Событие' : 'Новое событие'} onClose={onClose}>
      <label className="field">
        <span>Название</span>
        <input value={title} onChange={(e) => setTitle(e.target.value)} maxLength={200} placeholder="Встреча, тренировка…" />
      </label>
      <div className="field-row">
        <label className="field">
          <span>Начало</span>
          <input type="datetime-local" value={start} onChange={(e) => setStart(e.target.value)} />
        </label>
        <label className="field">
          <span>Окончание</span>
          <input type="datetime-local" value={end} onChange={(e) => setEnd(e.target.value)} />
        </label>
      </div>
      <label className="field">
        <span>Описание (необязательно)</span>
        <textarea value={desc} onChange={(e) => setDesc(e.target.value)} rows={3} maxLength={2000} placeholder="Место, заметки…" />
      </label>
      <FieldError text={fieldErr} />
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
