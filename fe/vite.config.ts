import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { execSync } from 'node:child_process'
import { fileURLToPath, URL } from 'node:url'

const appRoot = fileURLToPath(new URL('../', import.meta.url))

const resolveAppVersion = () => {
  if (process.env.VITE_APP_VERSION) return process.env.VITE_APP_VERSION
  try {
    return execSync('git describe --tags --exact-match', { cwd: appRoot, stdio: ['ignore', 'pipe', 'ignore'] })
      .toString()
      .trim()
  } catch {
    try {
      return execSync('git rev-parse --short HEAD', { cwd: appRoot, stdio: ['ignore', 'pipe', 'ignore'] })
        .toString()
        .trim()
    } catch {
      return 'unknown'
    }
  }
}

const appVersion = resolveAppVersion()

// https://vitejs.dev/config/
export default defineConfig({
  define: {
    'import.meta.env.VITE_APP_VERSION': JSON.stringify(appVersion),
  },
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    host: '0.0.0.0',
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:12395',
        changeOrigin: true,
      },
    },
  },
})
