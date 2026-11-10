import forest from '../assets/themes/forest.webp';
import coffee from '../assets/themes/coffee.webp';
import lavender from '../assets/themes/lavender.webp';
import type { ThemeName } from '../api/types';

// Тёмные изображения со светлыми дудлами: screen растворяет подложку.
const SRC: Record<Exclude<ThemeName, 'custom'>, string> = {
  forest,
  coffee,
  lavender,
};

export default function ThemeArt({ theme, artOn }: { theme: ThemeName; artOn: boolean }) {
  if (!artOn || theme === 'custom') return null;
  return (
    <div className="bg-illustration" aria-hidden="true">
      <img src={SRC[theme]} alt="" draggable={false} decoding="async" fetchPriority="low" />
    </div>
  );
}
