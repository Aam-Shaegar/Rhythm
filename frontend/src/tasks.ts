import type { Task, TaskPriority } from './api/types';
export function compareTasks(a: Task, b: Task): number {
 return Number(a.is_completed) - Number(b.is_completed) || Date.parse(a.due_at) - Date.parse(b.due_at) || a.id.localeCompare(b.id);
}
export function taskStatus(t: Task, now = Date.now()): string | null {
 if (t.is_completed) return t.completed_late || (t.completed_at && Date.parse(t.completed_at) > Date.parse(t.due_at)) ? 'Выполнена с опозданием · просрочена' : null;
 return Date.parse(t.due_at) < now ? 'Просрочена' : null;
}

export const CATEGORY_LABELS: Record<string,string> = { work: 'Работа', study: 'Учёба', finance: 'Финансы', home: 'Быт', health: 'Здоровье', personal: 'Личное', other: 'Прочее' };
export const PRIORITY_LABELS = { high: 'Высокий', medium: 'Средний', low: 'Низкий' };
export function comparePriorityTasks(a: Task, b: Task, priorities: Map<string, TaskPriority>): number {
 const completed = Number(a.is_completed) - Number(b.is_completed);
 if (completed) return completed;
 if (a.is_completed) return compareTasks(a,b);
 return (priorities.get(b.id)?.score ?? -1) - (priorities.get(a.id)?.score ?? -1) || compareTasks(a,b);
}
