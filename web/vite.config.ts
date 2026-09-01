import { defineConfig } from 'vite'
import preact from '@preact/preset-vite'

export default defineConfig({
  plugins: [preact()],
  base: './',
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    assetsDir: 'assets',
    sourcemap: false,
    chunkSizeWarningLimit: 500,
  },
  server: {
    port: 3000,
    proxy: {
      '/info': 'http://localhost:53217',
      '/devices': 'http://localhost:53217',
      '/settings': 'http://localhost:53217',
      '/history': 'http://localhost:53217',
      '/events': 'http://localhost:53217',
      '/send': 'http://localhost:53217',
      '/recv': 'http://localhost:53217',
      '/preview': 'http://localhost:53217',
      '/qr': 'http://localhost:53217',
      '/clipboard': 'http://localhost:53217',
      '/api': 'http://localhost:53217'
    }
  }
})
