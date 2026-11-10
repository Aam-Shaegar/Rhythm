import type { EventItem, ReminderSettings, Task } from './api/types';

export interface Notice {
  id: string;
  kind: 'overdue' | 'soon-event' | 'soon-task';
  title: string;
  detail: string;
  at: number; // timestamp для сортировки
  tab: 'tasks' | 'calendar';
}

// Сервер отдаёт только настройки: предстоящее выводим из загруженных сущностей.
export function deriveNotices(
  tasks: Task[],
  events: EventItem[],
  settings: ReminderSettings,
  now = new Date(),
): Notice[] {
  const out: Notice[] = [];
  const t = now.getTime();


  for (const task of tasks) {
    if (task.is_completed) continue;
    const due = new Date(task.due_at).getTime();
    if (Number.isNaN(due)) continue;
    if (due < t) {
      out.push({
        id: `overdue-${task.id}`,
        kind: 'overdue',
        title: `Просрочена: ${task.title}`,
        detail: `Срок был ${new Date(task.due_at).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' })}`,
        at: due,
        tab: 'tasks',
      });
    } else if (settings.task_offsets.length > 0 && due - t >= 0 && due - t <= Math.max(...settings.task_offsets) * 60000) {

      out.push({
        id: `soon-task-${task.id}`,
        kind: 'soon-task',
        title: `Скоро срок: ${task.title}`,
        detail: new Date(task.due_at).toLocaleString('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }),
        at: due,
        tab: 'tasks',
      });
    }
  }

  for (const ev of events) {
    const s = new Date(ev.start_at).getTime();
    if (Number.isNaN(s)) continue;
    if (s < t || !settings.event_offsets.length || s - t > Math.max(...settings.event_offsets) * 60000) continue;

    out.push({
      id: `soon-event-${ev.id}`,
      kind: 'soon-event',
      title: `Скоро событие: ${ev.title}`,
      detail: new Date(ev.start_at).toLocaleString('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' }),
      at: s,
      tab: 'calendar',
    });
  }

  return out.sort((a, b) => a.at - b.at);
}
