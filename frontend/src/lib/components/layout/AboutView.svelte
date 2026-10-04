<script lang="ts">
    import {openURL} from '$lib/utils/browser';
    import aetherLogo from '../../../assets/aether-logo.png';

    type Link = {label: string; url: string; description: string};

    const links: Link[] = [
        {
            label: 'GitHub',
            url: 'https://github.com/omacom/aether',
            description: 'Source code and issues',
        },
        {
            label: 'Documentation',
            url: 'https://github.com/omacom/aether/tree/master/docs',
            description: 'CLI, installation, custom apps, and templates',
        },
        {
            label: 'Releases',
            url: 'https://github.com/omacom/aether/releases/latest',
            description: 'Latest release notes and downloads',
        },
        {
            label: 'Omarchy',
            url: 'https://omarchy.org',
            description: 'The Linux desktop Aether is built for',
        },
    ];
    const AUTHOR_URL = 'https://x.com/iamdothash';
</script>

<div class="flex h-full items-center justify-center overflow-y-auto p-10">
    <div class="flex max-w-[440px] flex-col items-center gap-3.5 text-center">
        <img
            src={aetherLogo}
            alt="Aether"
            class="h-[72px] w-[72px] object-contain"
        />
        <div>
            <h1 class="text-fg-primary text-[24px] font-semibold">Aether</h1>
            <p class="text-fg-dimmed mt-1 font-mono text-[12px] font-medium">
                v{__APP_VERSION__}
            </p>
        </div>
        <p class="text-fg-secondary text-pretty text-[13px] leading-[1.55]">
            Aether extracts color palettes from wallpapers and applies cohesive
            themes across your Linux desktop. Built with Wails, Go, and Svelte
            5.
        </p>
        <nav
            class="flex flex-wrap justify-center gap-4 text-[12.5px]"
            aria-label="Links"
        >
            {#each links as link}
                <button
                    type="button"
                    class="text-accent hover:text-accent-hover transition-colors"
                    onclick={() => openURL(link.url)}
                    title={link.description}>{link.label}</button
                >
            {/each}
        </nav>
        <p class="text-fg-dimmed mt-3 text-[11px]">
            Built by <button
                type="button"
                class="text-fg-secondary hover:text-fg-primary underline underline-offset-2 transition-colors"
                onclick={() => openURL(AUTHOR_URL)}
                title="Author's X">Bjarne Øverli</button
            >
            · © {new Date().getFullYear()} · MIT License
        </p>
    </div>
</div>
