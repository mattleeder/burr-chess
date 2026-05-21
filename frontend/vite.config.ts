import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

const ReactCompilerConfig = { /* ... */ };

// https://vite.dev/config/
export default defineConfig(() => {
  return {
    server: {
      proxy: {
        // Proxy /api/* to the Go backend in dev, stripping the /api prefix.
        // e.g. /api/validateSession → http://localhost:8080/validateSession
        '/api': {
          target: process.env.BACKEND_URL ?? 'http://localhost:8080',
          changeOrigin: true,
          rewrite: (path) => path.replace(/^\/api/, ''),
          ws: true,
        },
      },
    },
    plugins: [
      react({
        babel: {
          plugins: [
            ["babel-plugin-react-compiler", ReactCompilerConfig],
          ],
        },
      }),
    ],
    test: {
      globals: true,
      environment: 'happy-dom',
      setupFiles: './src/test/setup.ts',
      environmentOptions: {
        happyDom: { url: 'http://localhost' },
      },
      exclude: ['tests/**', 'node_modules/**'],
    },
  };
});
