export type Tab = 'home' | 'calendar' | 'tasks' | 'reports' | 'settings';

export interface TabMeta {
  id: Tab;
  label: string;
  icon: React.ReactNode;
}

const svgProps = {
  viewBox: '0 0 24 24',
  fill: 'none',
  strokeWidth: 2,
  strokeLinecap: 'round',
  strokeLinejoin: 'round',
} as const;

export const TABS: TabMeta[] = [
  {
    id: 'home',
    label: 'Главная',
    icon: (
      <svg {...svgProps}>
        <path d="M3 11.5 12 4l9 7.5" />
        <path d="M5.5 10.5V20h13v-9.5" />
      </svg>
    ),
  },
  {
    id: 'calendar',
    label: 'Расписание',
    icon: (
      <svg {...svgProps}>
        <rect x="3" y="5" width="18" height="16" rx="3" />
        <line x1="3" y1="10" x2="21" y2="10" />
        <line x1="8" y1="3" x2="8" y2="7" />
        <line x1="16" y1="3" x2="16" y2="7" />
      </svg>
    ),
  },
  {
    id: 'tasks',
    label: 'Задачи',
    icon: (
      <svg {...svgProps}>
        <path d="M4 6l1.5 1.5L8 5" />
        <path d="M7 6h13M4 12h16M4 18h10" />
      </svg>
    ),
  },
  {
    id: 'reports',
    label: 'Отчёты',
    icon: (
      <svg {...svgProps}>
        <line x1="6" y1="20" x2="6" y2="12" />
        <line x1="12" y1="20" x2="12" y2="6" />
        <line x1="18" y1="20" x2="18" y2="15" />
      </svg>
    ),
  },
  {
    id: 'settings',
    label: 'Настройки',
    icon: (
      <svg {...svgProps}>
        <circle cx="12" cy="12" r="3" />
        <path d="M19 12a7 7 0 0 0-.1-1.2l2-1.6-2-3.4-2.4 1a7 7 0 0 0-2-1.2L14 3h-4l-.5 2.6a7 7 0 0 0-2 1.2l-2.4-1-2 3.4 2 1.6A7 7 0 0 0 5 12c0 .4 0 .8.1 1.2l-2 1.6 2 3.4 2.4-1a7 7 0 0 0 2 1.2L10 21h4l.5-2.6a7 7 0 0 0 2-1.2l2.4 1 2-3.4-2-1.6c.07-.4.1-.8.1-1.2Z" />
      </svg>
    ),
  },
];
