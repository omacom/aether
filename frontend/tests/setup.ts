import {afterEach, vi} from 'vitest';
import {flushSync, mount, tick, unmount, type Component} from 'svelte';

const cleanups: (() => Promise<void>)[] = [];

export function render<Props extends Record<string, unknown>>(
    component: Component<Props>,
    props: Props
) {
    const target = document.createElement('div');
    document.body.append(target);
    const instance = mount(component, {target, props});
    let destroyed = false;
    const destroy = async () => {
        if (destroyed) return;
        destroyed = true;
        await unmount(instance);
        target.remove();
    };
    cleanups.push(destroy);
    flushSync();
    return {target, destroy};
}

export async function settle() {
    await vi.dynamicImportSettled();
    await tick();
    flushSync();
}

export function deferred<T>() {
    let resolve!: (value: T) => void;
    let reject!: (reason?: unknown) => void;
    const promise = new Promise<T>((res, rej) => {
        resolve = res;
        reject = rej;
    });
    return {promise, resolve, reject};
}

export function button(target: HTMLElement, label: string): HTMLButtonElement {
    const found = [...target.querySelectorAll('button')].find(
        el => el.textContent?.trim() === label
    );
    if (!found) throw new Error(`Missing button: ${label}`);
    return found;
}

afterEach(async () => {
    for (const cleanup of cleanups.splice(0)) await cleanup();
    vi.clearAllTimers();
    vi.useRealTimers();
    localStorage.clear();
});
