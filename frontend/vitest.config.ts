import {defineConfig} from 'vitest/config';
import {svelte} from '@sveltejs/vite-plugin-svelte';
import {fileURLToPath} from 'node:url';

export default defineConfig({
    plugins: [svelte()],
    define: {__APP_VERSION__: JSON.stringify('test')},
    resolve: {
        conditions: ['browser'],
        alias: {$lib: fileURLToPath(new URL('./src/lib', import.meta.url))},
    },
    test: {
        environment: 'jsdom',
        include: ['tests/**/*.test.ts'],
        setupFiles: ['./tests/setup.ts'],
        clearMocks: true,
        restoreMocks: true,
    },
});
