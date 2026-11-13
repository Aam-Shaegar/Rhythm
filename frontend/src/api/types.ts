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
  importance: number;
  estimated_minutes: number;
  id: string;
  title: string;
  description?: string | null;
  due_at: string;
  is_completed: boolean;
  is_overdue: boolean;
  completed_late: boolean;
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
  completed_on_time: number;
  completed_late: number;
  overdue_tasks: number;
  on_time_pct: number;
  date: string;
  total_tasks: number;
  completed_tasks: number;
  completion_pct: number;
  events_count: number;
}

export interface ReminderSettings {
  event_offsets: number[];
  task_offsets: number[];
}

export interface ApiError {
  message: string;
  error: string;
  status: number;
}

export type ThemeName = 'forest' | 'coffee' | 'lavender' | 'custom';

export interface TaskPriority {
  task_id: string;
  category: string;
  category_confidence: number;
  risk: number;
  score: number;
  level: 'high' | 'medium' | 'low';
  reasons: string[];
  model_version: string;
}
export interface PriorityCategoryStat {
  category: string;
  known_outcomes: number;
  late_outcomes: number;
  completed_late: number;
  open_late: number;
  late_rate: number;
}
export interface PriorityResponse {
  items: TaskPriority[];
  recommendations: string[];
  categories: PriorityCategoryStat[];
  model_version: string;
  model_source: string;
  personal_samples: number;
  personal_state: 'baseline' | 'learning' | 'personalized';
  trained_at: string;
}
