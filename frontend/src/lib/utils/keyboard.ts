/**
 * Global keyboard shortcut handler.
 * Registers shortcuts once and dispatches to callbacks.
 */

type ShortcutHandler = () => void;

const shortcuts: Map<string, ShortcutHandler> = new Map();

function keyToString(e: KeyboardEvent): string {
    const parts: string[] = [];
    if (e.ctrlKey || e.metaKey) parts.push('ctrl');
    if (e.shiftKey) parts.push('shift');
    if (e.altKey) parts.push('alt');
    parts.push(e.key.toLowerCase());
    return parts.join('+');
}

// Input types that take no typed text. Every other type counts as text entry,
// which keeps unknown types on the safe side.
const NON_TEXT_INPUT_TYPES = new Set([
    'range',
    'checkbox',
    'radio',
    'button',
    'submit',
    'reset',
    'color',
]);

export function isTextEntryTarget(target: EventTarget | null): boolean {
    if (!(target instanceof HTMLElement)) return false;
    if (target instanceof HTMLTextAreaElement) return true;
    if (target instanceof HTMLInputElement) {
        return !NON_TEXT_INPUT_TYPES.has(target.type);
    }
    return (
        target.isContentEditable ||
        target.closest('[contenteditable]:not([contenteditable="false"])') !==
            null
    );
}

export function registerShortcut(combo: string, handler: ShortcutHandler) {
    shortcuts.set(combo.toLowerCase(), handler);
}

export function initKeyboardShortcuts() {
    document.addEventListener('keydown', (e: KeyboardEvent) => {
        // Don't intercept when typing in text fields. Allow Escape there.
        if (isTextEntryTarget(e.target) && e.key !== 'Escape') return;

        const key = keyToString(e);
        const handler = shortcuts.get(key);
        if (handler) {
            e.preventDefault();
            handler();
        }
    });
}

// Helper for role=button divs/spans: turn Enter and Space into a click,
// matching native button behaviour. Use on `onkeydown=`.
export function onActivate(handler: () => void) {
    return (e: KeyboardEvent) => {
        if (e.key === 'Enter' || e.key === ' ') {
            e.preventDefault();
            handler();
        }
    };
}
