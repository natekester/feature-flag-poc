import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',
    port: 3000,
    strictPort: true,
  },
  test: {
    globals: true,
    environment: 'happy-dom',
    includeSource: ['src/**/*.{js,ts,jsx,tsx}'],
  },
});
