import '@fontsource-variable/inter';
import '@fontsource-variable/jetbrains-mono';
import './app.css';
import App from './App.svelte';
import {mount} from 'svelte';

const app = mount(App, {
    target: document.getElementById('app')!,
});

export default app;
