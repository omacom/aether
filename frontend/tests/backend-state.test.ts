import {expect, test} from 'vitest';
import * as theme from '../src/lib/stores/theme.svelte';

test('applyBackendState applies glass, clock and roles from the backend', () => {
    theme.setGlass(null);
    theme.setClock(null);
    theme.applyBackendState({
        yozakuraRoles: {accent: '#ff0000'},
        glass: 0.8,
        clock: {digits: 'thin', colon: '·', glow: 0.5, ornament: 'none'},
    });
    expect(theme.getGlass()).toBe(0.8);
    expect(theme.getClock()).toEqual({
        digits: 'thin',
        colon: '·',
        glow: 0.5,
        ornament: 'none',
    });
    expect(theme.getYozakuraRoles()).toEqual({accent: '#ff0000'});

    theme.applyBackendState({glass: null, clock: null});
    expect(theme.getGlass()).toBeNull();
    expect(theme.getClock()).toBeNull();
});
