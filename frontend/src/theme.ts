import type { ThemeName } from './api/types';

// Ключи localStorage (исторические имена сохранены для совместимости).
const THEME_KEY = 'rhytm.theme';
const THEME_COLOR_KEY = 'rhytm.theme.color';
const THEME_PHOTO_KEY = 'rhytm.theme.photo';
const THEME_CUSTOM_KEY = 'rhytm.theme.custom';
const THEME_ART_KEY = 'rhytm.theme.art';

export interface CustomPalette {
  accent: string;
  bgA: string;
  bgB: string;
  text: string;
}

export const DEFAULT_CUSTOM: CustomPalette = {
  accent: '#8a8f6e',
  bgA: '#e9e6d4',
  bgB: '#c8caa8',
  text: '#33321f',
};

export function loadThemeName(): ThemeName {
  return (localStorage.getItem(THEME_KEY) as ThemeName) || 'forest';
}

export function saveThemeName(t: ThemeName): void {
  localStorage.setItem(THEME_KEY, t);
}

export function loadArt(): boolean {
  return localStorage.getItem(THEME_ART_KEY) !== '0';
}

export function saveArt(on: boolean): void {
  localStorage.setItem(THEME_ART_KEY, on ? '1' : '0');
}

export function loadCustom(): CustomPalette {
  try {
    const raw = localStorage.getItem(THEME_CUSTOM_KEY);
    if (raw) {
      const p = JSON.parse(raw) as Partial<CustomPalette>;
      if (p.accent && p.bgA && p.bgB && p.text) return p as CustomPalette;
    }
  } catch {
    // Битый JSON → дефолт и миграция со старого ключа ниже.
  }
  const legacy = localStorage.getItem(THEME_COLOR_KEY);
  if (legacy && /^#[0-9a-fA-F]{6}$/.test(legacy)) {
    return { ...DEFAULT_CUSTOM, accent: legacy };
  }
  return { ...DEFAULT_CUSTOM };
}

export function saveCustom(p: CustomPalette): void {
  localStorage.setItem(THEME_CUSTOM_KEY, JSON.stringify(p));
  // Дублируем акцент в исторический ключ для совместимости.
  localStorage.setItem(THEME_COLOR_KEY, p.accent);
}

export function loadPhoto(): string | null {
  return localStorage.getItem(THEME_PHOTO_KEY);
}

function clampHex(h: string): string | null {
  return /^#[0-9a-fA-F]{6}$/.test(h) ? h : null;
}

function toRgb(h: string): [number, number, number] {
  return [parseInt(h.slice(1, 3), 16), parseInt(h.slice(3, 5), 16), parseInt(h.slice(5, 7), 16)];
}

function toHex(r: number, g: number, b: number): string {
  const c = (n: number) => Math.max(0, Math.min(255, Math.round(n))).toString(16).padStart(2, '0');
  return `#${c(r)}${c(g)}${c(b)}`;
}

// Смешивание hex: ratio=0→a, 1→b.
export function mixHex(a: string, b: string, ratio: number): string {
  const pa = clampHex(a) ?? '#808080';
  const pb = clampHex(b) ?? '#808080';
  const [r1, g1, b1] = toRgb(pa);
  const [r2, g2, b2] = toRgb(pb);
  const t = Math.max(0, Math.min(1, ratio));
  return toHex(r1 + (r2 - r1) * t, g1 + (g2 - g1) * t, b1 + (b2 - b1) * t);
}

// Относительная яркость 0..1 для выбора поверхностей.
export function luminance(h: string): number {
  const [r, g, b] = toRgb(clampHex(h) ?? '#808080').map((v) => {
    const s = v / 255;
    return s <= 0.03928 ? s / 12.92 : Math.pow((s + 0.055) / 1.055, 2.4);
  });
  return 0.2126 * r + 0.7152 * g + 0.0722 * b;
}

export function applyThemeToDom(): void {
  const screen = document.getElementById('phoneScreen');
  if (!screen) return;
  const theme = loadThemeName();
  const art = loadArt();
  screen.setAttribute('data-theme', theme);
  screen.classList.toggle('no-art', !art);

  for (const v of ['--accent', '--accent-deep', '--bg-a', '--bg-b', '--text', '--text-soft', '--surface', '--surface-strong']) {
    screen.style.removeProperty(v);
  }
  screen.classList.remove('has-photo');

  if (theme === 'custom') {
    const p = loadCustom();
    const photo = loadPhoto();
    screen.style.setProperty('--accent', p.accent);
    screen.style.setProperty('--accent-deep', p.accent);
    if (photo) {
      // Поверх фото текст светлый (.has-photo): палитру не трогаем.
      screen.classList.add('has-photo');
      const el = document.getElementById('customPhoto');
      if (el) el.style.backgroundImage = `url(${photo})`;
      return;
    }
    const dark = luminance(p.bgA) < 0.4;
    const surface = dark ? 'rgba(20,20,18,0.42)' : 'rgba(255,255,255,0.55)';
    const surfaceStrong = dark ? 'rgba(20,20,18,0.6)' : 'rgba(255,255,255,0.78)';
    screen.style.setProperty('--bg-a', p.bgA);
    screen.style.setProperty('--bg-b', p.bgB);
    screen.style.setProperty('--text', p.text);
    screen.style.setProperty('--text-soft', mixHex(p.text, p.bgA, 0.35));
    screen.style.setProperty('--surface', surface);
    screen.style.setProperty('--surface-strong', surfaceStrong);
  }
}
