import type { Task } from './api/types';
export function compareTasks(a: Task, b: Task): number {
 return Number(a.is_completed) - Number(b.is_completed) || Date.parse(a.due_at) - Date.parse(b.due_at) || a.id.localeCompare(b.id);
}
export function taskStatus(t: Task, now = Date.now()): string | null {
 if (t.is_completed) return t.completed_late || (t.completed_at && Date.parse(t.completed_at) > Date.parse(t.due_at)) ? 'Выполнена с опозданием · просрочена' : null;
 return Date.parse(t.due_at) < now ? 'Просрочена' : null;
}
