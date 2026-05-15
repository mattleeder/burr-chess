import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

const ReactCompilerConfig = { /* ... */ };

// https://vite.dev/config/
export default defineConfig(() => {
  return {
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
    },
  };
});
