import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import { readFileSync, readdirSync, writeFileSync } from 'node:fs';
import { createHash } from 'node:crypto';

export default defineConfig({
  plugins: [react(), {
    name: 'rhythm-service-worker',
    apply: 'build',
    closeBundle() {
      const assets = readdirSync('dist/assets').filter(f => /\.(js|css)$/.test(f)).map(f => `/assets/${f}`);
      const version = createHash('sha256').update(assets.join(',')).digest('hex').slice(0,12);
      const sw = readFileSync('public/sw.js','utf8').replace("const CACHE = 'rhythm-dev';", `const CACHE = 'rhythm-${version}';`).replace('const PRECACHE = [];', `const PRECACHE = ${JSON.stringify(['/', '/index.html', '/manifest.json', '/icons/icon-192.png', ...assets])};`);
      writeFileSync('dist/sw.js',sw);
    },
  }],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: process.env.VITE_API_URL?.replace(/\/api\/v1\/?$/, '') ?? 'http://localhost:8080',
        changeOrigin: true,
      },
    },
  },
  preview: { port: 5173 },
});
