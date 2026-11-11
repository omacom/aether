import {debounce} from '$lib/utils/debounce';
import {clamp} from '$lib/utils/math';
import {GetYozakuraRoles} from '../../../wailsjs/go/main/App';
import type {main} from '../../../wailsjs/go/models';
import {
    getClock as getStoredClock,
    getExtendedColors,
    getGlass as getStoredGlass,
    getLightMode,
    getPalette,
    getYozakuraRoles,
    setClock as setStoredClock,
    setGlass as setStoredGlass,
    setYozakuraRoles,
    type YozakuraClock,
} from './theme.svelte';

export const YOZAKURA_ROLE_KEYS = [
    'ground',
    'surface',
    'surface2',
    'border',
    'accent',
    'accent2',
    'info',
    'danger',
    'ok',
    'warn',
    'text',
    'muted',
    'accentInk',
] as const;

export const GLASS_MIN = 0.8;
export const GLASS_MAX = 1;
export const DEFAULT_GLASS = 0.92;
export const DEFAULT_CLOCK: YozakuraClock = {
    digits: 'bold',
    colon: ':',
    glow: 0.35,
    ornament: 'arc',
};
export const CLOCK_DIGITS = ['thin', 'medium', 'bold', 'heavy'] as const;
export const CLOCK_ORNAMENTS = [
    'arc',
    'arcButterfly',
    'stars',
    'wave',
    'seal',
] as const;

export interface YozakuraContrast {
    pair: string;
    ratio: number;
    min: number;
    state: string;
}

export interface YozakuraResult {
    roles: Record<string, string>;
    explicit: Record<string, boolean>;
    glass: number;
    clock: YozakuraClock;
    contrast: YozakuraContrast[];
    accentCandidates: string[];
}

let result = $state.raw<YozakuraResult | null>(null);
let requestId = 0;

export function getYozakuraResult(): YozakuraResult | null {
    return result;
}

export function getYozakuraOverrides(): Record<string, string> {
    return getYozakuraRoles();
}

export function setYozakuraRole(key: string, hex: string): void {
    setYozakuraRoles({...getYozakuraRoles(), [key]: hex});
}

export function resetYozakuraRole(key: string): void {
    const {[key]: _removed, ...rest} = getYozakuraRoles();
    setYozakuraRoles(rest);
}

// The role as shown: the pinned override, else the backend-derived value.
export function getEffectiveRole(key: string): string {
    return getYozakuraRoles()[key] ?? result?.roles[key] ?? '';
}

// Pins the next chromatic accent candidate, wrapping past the last one.
export function cycleAccent(): void {
    const candidates = result?.accentCandidates ?? [];
    if (candidates.length === 0) return;
    const at = candidates.indexOf(getEffectiveRole('accent').toLowerCase());
    setYozakuraRole('accent', candidates[(at + 1) % candidates.length]);
}

export function getGlass(): number {
    return getStoredGlass() ?? result?.glass ?? DEFAULT_GLASS;
}

export function setGlass(value: number): void {
    if (!Number.isFinite(value)) return;
    setStoredGlass(clamp(value, GLASS_MIN, GLASS_MAX));
}

export function getClock(): YozakuraClock {
    return getStoredClock() ?? result?.clock ?? DEFAULT_CLOCK;
}

export function setClock(value: YozakuraClock): void {
    setStoredClock({...value, glow: clamp(value.glow, 0, 1)});
}

async function fetchYozakura(): Promise<void> {
    const id = ++requestId;
    const request = {
        palette: [...getPalette()],
        extendedColors: {...getExtendedColors()},
        lightMode: getLightMode(),
        overrides: {...getYozakuraRoles()},
        glass: getStoredGlass(),
        clock: getStoredClock(),
    } as unknown as main.YozakuraRolesRequest;
    try {
        const next = await GetYozakuraRoles(request);
        if (id === requestId) result = next as YozakuraResult;
    } catch (e) {
        if (id === requestId) console.error('GetYozakuraRoles failed:', e);
    }
}

export const refreshYozakura = debounce(() => void fetchYozakura(), 120);

export function resetYozakuraResult(): void {
    refreshYozakura.cancel();
    requestId++;
    result = null;
}
