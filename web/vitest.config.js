import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
  plugins: [svelte({ hot: !process.env.VITEST })],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/setupTests.js'],
    include: ['src/**/*.{test,spec}.{js,ts}'],
    testTimeout: 30000,
    coverage: {
      reporter: ['text', 'json', 'html'],
      exclude: [
        'coverage/**',
        'dist/**',
        '**/[.]**',
        'packages/*/test?(s)/**',
        '**/*.d.ts',
        '**/vite.config.ts',
        '**/{karma,rollup,webpack,vite,vitest,jest,ava,babel,nyc,cypress}.*',
        '**/.{eslint,mocha,prettier}rc.{js,cjs,mjs,yml,yaml,json}'
      ]
    }
  }
}); 