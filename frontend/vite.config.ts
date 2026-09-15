import {defineConfig} from 'vite';
import {svelte} from '@sveltejs/vite-plugin-svelte';
import tailwindcss from '@tailwindcss/vite';
import {fileURLToPath} from 'node:url';
import {readFileSync} from 'node:fs';

const pkg = JSON.parse(
    readFileSync(new URL('./package.json', import.meta.url), 'utf-8')
);

export default defineConfig({
    plugins: [tailwindcss(), svelte()],
    define: {
        __APP_VERSION__: JSON.stringify(pkg.version),
    },
    resolve: {
        alias: {
            $lib: fileURLToPath(new URL('./src/lib', import.meta.url)),
        },
    },
});
