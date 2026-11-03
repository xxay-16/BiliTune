import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {port: 9245, strictPort: true, proxy: {'/api': 'http://127.0.0.1:9246'}},
  build: {target: 'es2022', chunkSizeWarningLimit: 800},
});
