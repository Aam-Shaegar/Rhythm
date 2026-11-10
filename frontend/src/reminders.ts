import type { ReminderSettings } from './api/types';
export const DEFAULT_SETTINGS: ReminderSettings = { event_offsets: [60, 0], task_offsets: [60, 0] };
export function reminderLabel(minutes: number): string {
 if (minutes === 0) return 'В момент срока';
 if (minutes % 1440 === 0) return `За ${minutes / 1440} дн.`;
 if (minutes % 60 === 0) return `За ${minutes / 60} ч.`;
 return `За ${minutes} мин.`;
}
