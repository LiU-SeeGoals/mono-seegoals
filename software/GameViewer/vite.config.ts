import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react-swc';

// https://vitejs.dev/config/
export default defineConfig({
  cacheDir: '.vite',
  plugins: [react()],
  server: {
    host: process.env.GAME_VIEWER_ADDR,
    port: parseInt(process.env.GAME_VIEWER_PORT),
  },
});
