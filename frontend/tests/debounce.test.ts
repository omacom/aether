import {expect, test, vi} from 'vitest';
import {debounce} from '../src/lib/utils/debounce';

test('debounce keeps the newest arguments and can be canceled and reused', () => {
    vi.useFakeTimers();
    const fn = vi.fn();
    const debounced = debounce(fn, 50);
    debounced('first');
    debounced('second');
    vi.advanceTimersByTime(50);
    expect(fn).toHaveBeenCalledExactlyOnceWith('second');
    debounced('canceled');
    debounced.cancel();
    debounced.cancel();
    vi.advanceTimersByTime(50);
    expect(fn).toHaveBeenCalledTimes(1);
    debounced('after cancel');
    vi.advanceTimersByTime(50);
    expect(fn).toHaveBeenLastCalledWith('after cancel');
});
