export interface User {
  id: string;
  username: string;
  email: string;
  created_at: string;
  updated_at: string;
}

export interface AuthResponse {
  user: User;
  access_token: string;
  refresh_token: string;
}

// login returns a flat shape, register returns AuthResponse — normalize on client
export interface LoginResponse {
  access_token: string;
  user_id: string;
  username: string;
  email: string;
}

export type RecurrenceType = 'daily' | 'weekly' | 'monthly' | 'yearly';

export interface Task {
  id: string;
  title: string;
  description?: string | null;
  due_at: string;
  is_completed: boolean;
  completed_at?: string | null;
  recurrence_type?: RecurrenceType | null;
  recurrence_end?: string | null;
  parent_task_id?: string | null;
  created_at: string;
  updated_at: string;
}

export interface EventItem {
  id: string;
  title: string;
  description?: string | null;
  start_at: string;
  end_at: string;
  created_at: string;
  updated_at: string;
}

export interface DailyReport {
  date: string;
  total_tasks: number;
  completed_tasks: number;
  completion_pct: number;
  events_count: number;
}

export interface ReminderSettings {
  event_reminders: boolean;
  task_reminders: boolean;
  event_before_15m: boolean;
  event_before_1h: boolean;
  event_before_24h: boolean;
  task_before_1h: boolean;
  task_before_24h: boolean;
}

export interface ApiError {
  message: string;
  error: string;
  status: number;
}

export type ThemeName = 'forest' | 'coffee' | 'lavender' | 'custom';
