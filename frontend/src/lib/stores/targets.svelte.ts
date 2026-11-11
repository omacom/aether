import type {main, targets as targetsNs} from '../../../wailsjs/go/models';
import {getSettings, updateSettings} from '$lib/stores/settings.svelte';
import {SPECIAL_APP_FLAGS} from '$lib/constants/apps';

type ApplyResultLike = {targets?: targetsNs.Status[] | null} | null | undefined;

let targets = $state<main.TargetInfo[]>([]);
let statuses = $state<Record<string, targetsNs.Status>>({});

export function getTargets(): main.TargetInfo[] {
    return targets;
}

export async function loadTargets(): Promise<void> {
    try {
        const {ListTargets} = await import('../../../wailsjs/go/main/App');
        targets = (await ListTargets()) ?? [];
    } catch {
        targets = [];
    }
}

/** Status reported by the most recent Apply, if any. */
export function getTargetStatus(id: string): targetsNs.Status | undefined {
    return statuses[id];
}

export function setLastApply(result: ApplyResultLike): void {
    statuses = Object.fromEntries(
        (result?.targets ?? []).map(st => [st.id, st])
    );
}

/** Targets are opt-out: on unless includedApps[id] is explicitly false. */
export function isTargetEnabled(id: string): boolean {
    return getSettings().includedApps?.[id] !== false;
}

/** Persists an explicit true/false so the apply path can honour opt-out. */
export function setTargetEnabled(id: string, on: boolean): void {
    const flag = SPECIAL_APP_FLAGS[id];
    updateSettings({
        includedApps: {...(getSettings().includedApps ?? {}), [id]: on},
        ...(flag ? {[flag]: on} : {}),
    });
}
