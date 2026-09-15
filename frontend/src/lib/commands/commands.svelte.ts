import {
    setActiveTab,
    setKeymapOpen,
    setImageEditorOpen,
} from '$lib/stores/ui.svelte';
import {
    getWallpaperPath,
    getLightMode,
    setLightMode,
    getIsApplying,
    getBasePalette,
    getBaseExtendedColors,
    getHistorySnapshot,
    restoreHistorySnapshot,
} from '$lib/stores/theme.svelte';
import {getCanUndo, getCanRedo, pushState} from '$lib/stores/history.svelte';
import {DEFAULT_ADJUSTMENTS} from '$lib/types/theme';
import {zoomIn, zoomOut, resetZoom} from '$lib/utils/zoom';
import {
    applyTheme,
    undoAction,
    redoAction,
    changeWallpaper,
    extractColors,
    saveThemeAsNew,
} from '$lib/actions/themeActions';
import {getOmarchyAvailable} from '$lib/stores/omarchy.svelte';

export type Command = {
    id: string;
    label: string;
    category: string;
    shortcut?: string;
    keywords?: string;
    run: () => void | Promise<void>;
    visible?: () => boolean;
    disabled?: () => string | undefined;
    colors?: string[];
};

const hasWallpaper = () => !!getWallpaperPath();

export function buildCommands(): Command[] {
    return [
        {
            id: 'nav.editor',
            label: 'Go to Editor',
            category: 'Navigate',
            keywords: 'theme palette',
            run: () => setActiveTab('editor'),
        },
        {
            id: 'nav.wallhaven',
            label: 'Go to Wallhaven',
            category: 'Navigate',
            keywords: 'browse online wallpapers',
            run: () => setActiveTab('wallhaven'),
        },
        {
            id: 'nav.local',
            label: 'Go to Local',
            category: 'Navigate',
            keywords: 'browse files',
            run: () => setActiveTab('local'),
        },
        {
            id: 'nav.favorites',
            label: 'Go to Favorites',
            category: 'Navigate',
            run: () => setActiveTab('favorites'),
        },
        {
            id: 'nav.blueprints',
            label: 'Go to Blueprints',
            category: 'Navigate',
            keywords: 'saved themes presets',
            run: () => setActiveTab('blueprints'),
        },
        {
            id: 'nav.system',
            label: 'Go to System',
            category: 'Navigate',
            keywords: 'themes omarchy',
            run: () => setActiveTab('system'),
            visible: getOmarchyAvailable,
        },
        {
            id: 'nav.settings',
            label: 'Go to Settings',
            category: 'Navigate',
            keywords: 'preferences wallpaper folder',
            run: () => setActiveTab('settings'),
        },

        {
            id: 'theme.apply',
            label: 'Apply theme',
            category: 'Theme',
            shortcut: 'Ctrl+Enter',
            keywords: 'publish install',
            run: applyTheme,
            disabled: () =>
                getIsApplying() ? 'A theme is being applied' : undefined,
        },
        {
            id: 'theme.saveAs',
            label: 'Save and apply as new theme...',
            category: 'Theme',
            shortcut: 'Ctrl+J',
            keywords: 'name folder publish install',
            run: saveThemeAsNew,
            disabled: () =>
                getIsApplying() ? 'A theme is being applied' : undefined,
        },
        {
            id: 'theme.toggleLight',
            label: 'Toggle light / dark mode',
            category: 'Theme',
            keywords: 'invert appearance',
            run: () => setLightMode(!getLightMode()),
        },
        {
            id: 'theme.resetAdjustments',
            label: 'Reset palette adjustments',
            category: 'Theme',
            keywords: 'revert sliders',
            run: () => {
                const snapshot = getHistorySnapshot();
                pushState(snapshot);
                restoreHistorySnapshot({
                    ...snapshot,
                    palette: [...getBasePalette()],
                    extendedColors: {...getBaseExtendedColors()},
                    adjustments: {...DEFAULT_ADJUSTMENTS},
                    paletteCurvePoints: [],
                    pendingAdjustment: null,
                });
            },
        },

        {
            id: 'edit.undo',
            label: 'Undo',
            category: 'Edit',
            shortcut: 'Ctrl+Z',
            run: undoAction,
            disabled: () => (getCanUndo() ? undefined : 'Nothing to undo'),
        },
        {
            id: 'edit.redo',
            label: 'Redo',
            category: 'Edit',
            shortcut: 'Ctrl+Shift+Z',
            run: redoAction,
            disabled: () => (getCanRedo() ? undefined : 'Nothing to redo'),
        },

        {
            id: 'wallpaper.change',
            label: 'Change wallpaper…',
            category: 'Wallpaper',
            keywords: 'open file browse image',
            run: changeWallpaper,
        },
        {
            id: 'wallpaper.extract',
            label: 'Extract palette from wallpaper',
            category: 'Wallpaper',
            keywords: 'colors generate',
            visible: hasWallpaper,
            run: extractColors,
        },
        {
            id: 'wallpaper.edit',
            label: 'Open image editor',
            category: 'Wallpaper',
            keywords: 'crop adjust filters rotate',
            visible: hasWallpaper,
            run: () => setImageEditorOpen(true),
        },

        {
            id: 'view.zoomIn',
            label: 'Zoom in',
            category: 'View',
            shortcut: 'Ctrl++',
            run: zoomIn,
        },
        {
            id: 'view.zoomOut',
            label: 'Zoom out',
            category: 'View',
            shortcut: 'Ctrl+-',
            run: zoomOut,
        },
        {
            id: 'view.zoomReset',
            label: 'Reset zoom',
            category: 'View',
            shortcut: 'Ctrl+0',
            run: resetZoom,
        },
        {
            id: 'view.keymap',
            label: 'Show keyboard shortcuts',
            category: 'View',
            shortcut: 'Ctrl+K',
            keywords: 'help',
            run: () => setKeymapOpen(true),
        },
    ];
}
