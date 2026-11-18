import {afterEach, beforeAll, expect, test, vi} from 'vitest';
import {
    initKeyboardShortcuts,
    registerShortcut,
} from '../src/lib/utils/keyboard';

const undo = vi.fn();
const apply = vi.fn();
const escape = vi.fn();

beforeAll(() => {
    initKeyboardShortcuts();
    registerShortcut('ctrl+z', () => undo());
    registerShortcut('ctrl+enter', () => apply());
    registerShortcut('escape', () => escape());
});

afterEach(() => {
    document.body.innerHTML = '';
    undo.mockClear();
    apply.mockClear();
    escape.mockClear();
});

function press(el: HTMLElement, key: string, ctrl = true) {
    el.focus();
    el.dispatchEvent(
        new KeyboardEvent('keydown', {key, ctrlKey: ctrl, bubbles: true})
    );
}

function input(type: string): HTMLInputElement {
    const el = document.createElement('input');
    el.type = type;
    document.body.append(el);
    return el;
}

test.each(['range', 'checkbox', 'radio'])(
    'ctrl+z and ctrl+enter run with the focus on a %s input',
    type => {
        const el = input(type);
        press(el, 'z');
        press(el, 'Enter');
        expect(undo).toHaveBeenCalledTimes(1);
        expect(apply).toHaveBeenCalledTimes(1);
    }
);

const textTargets: [string, () => HTMLElement][] = [
    ...['text', 'search', 'number', 'email', 'url', 'password', 'tel'].map(
        type =>
            [`input ${type}`, () => input(type)] as [string, () => HTMLElement]
    ),
    [
        'textarea',
        () => {
            const el = document.createElement('textarea');
            document.body.append(el);
            return el;
        },
    ],
    [
        'contenteditable',
        () => {
            const el = document.createElement('div');
            el.setAttribute('contenteditable', 'true');
            el.tabIndex = 0;
            document.body.append(el);
            return el;
        },
    ],
];

test.each(textTargets)('text entry %s keeps ctrl+z for itself', (_, make) => {
    const el = make();
    press(el, 'z');
    expect(undo).not.toHaveBeenCalled();
});

test.each(textTargets)('escape still runs from %s', (_, make) => {
    const el = make();
    press(el, 'Escape', false);
    expect(escape).toHaveBeenCalledTimes(1);
});
