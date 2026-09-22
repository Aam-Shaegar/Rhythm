import type { ThemeName } from '../api/types';

// Тёмные PNG со светлыми дудлами: screen растворяет подложку.
const SRC: Record<Exclude<ThemeName, 'custom'>, string> = {
  forest: '/themes/forest.png',
  coffee: '/themes/coffee.png',
  lavender: '/themes/lavender.png',
};

export default function ThemeArt({ theme, artOn }: { theme: ThemeName; artOn: boolean }) {
  if (!artOn || theme === 'custom') return null;
  return (
    <div className="bg-illustration" aria-hidden="true">
      <img src={SRC[theme]} alt="" draggable={false} />
    </div>
  );
}
