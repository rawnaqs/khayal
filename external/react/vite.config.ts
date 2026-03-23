import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

export default defineConfig({
  plugins: [react()],
  base: '/',
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    outDir: '../../internal/api/ui/static',
    emptyOutDir: true,
  },
  server: {
    port: 5173,
    proxy: {
      '/v1': {
        target: 'http://localhost:1133',
        changeOrigin: true,
      },
    },
  },
})
