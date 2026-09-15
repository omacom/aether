export function debounce<TArgs extends unknown[]>(
    fn: (...args: TArgs) => void,
    ms: number
): ((...args: TArgs) => void) & {cancel: () => void} {
    let timer: ReturnType<typeof setTimeout> | undefined;
    const debounced = (...args: TArgs) => {
        clearTimeout(timer);
        timer = setTimeout(() => {
            timer = undefined;
            fn(...args);
        }, ms);
    };
    debounced.cancel = () => {
        clearTimeout(timer);
        timer = undefined;
    };
    return debounced;
}
