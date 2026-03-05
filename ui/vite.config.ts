import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  server: {
    port: 3521,
    proxy: {
      '/api': {
        target: 'http://localhost:7127',
        changeOrigin: true,
      },
    },
  },
})
