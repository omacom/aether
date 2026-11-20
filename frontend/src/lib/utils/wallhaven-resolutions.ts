export interface ResolutionOption {
    value: string;
    label: string;
}

export interface ResolutionOptionGroups {
    minimum: ResolutionOption[];
    exact: ResolutionOption[];
}

interface BaseResolution {
    res: string;
    tag?: string;
}

const BASE_RESOLUTIONS: readonly BaseResolution[] = [
    {res: '1920x1080'},
    {res: '2560x1440'},
    {res: '3440x1440', tag: '21:9'},
    {res: '3840x2160'},
    {res: '3840x1080', tag: '32:9'},
    {res: '5120x1440', tag: '32:9'},
];

const RESOLUTION_RE = /^\d+x\d+$/;

function formatDimensions(res: string): string {
    return res.replace('x', '×');
}

function formatLabel(
    res: string,
    mode: 'minimum' | 'exact',
    suffix?: string
): string {
    const base =
        mode === 'minimum'
            ? `${formatDimensions(res)}+`
            : formatDimensions(res);
    return suffix ? `${base} (${suffix})` : base;
}

export function buildResolutionOptions(
    detectedResolution: string,
    currentSelection: string
): ResolutionOptionGroups {
    const detected = RESOLUTION_RE.test(detectedResolution)
        ? detectedResolution
        : '';
    const baseSet = new Set(BASE_RESOLUTIONS.map(item => item.res));

    const minimum: ResolutionOption[] = [];
    const exact: ResolutionOption[] = [];

    if (detected && !baseSet.has(detected)) {
        minimum.push({
            value: detected,
            label: formatLabel(detected, 'minimum', 'Display'),
        });
        exact.push({
            value: `exact:${detected}`,
            label: formatLabel(detected, 'exact', 'Display'),
        });
    }

    for (const item of BASE_RESOLUTIONS) {
        const suffix = item.res === detected ? 'Display' : item.tag;
        minimum.push({
            value: item.res,
            label: formatLabel(item.res, 'minimum', suffix),
        });
        exact.push({
            value: `exact:${item.res}`,
            label: formatLabel(item.res, 'exact', suffix),
        });
    }

    // Keep any custom saved resolution visible so the <select> never goes blank.
    if (currentSelection) {
        const isExact = currentSelection.startsWith('exact:');
        const raw = isExact
            ? currentSelection.slice('exact:'.length)
            : currentSelection;
        if (RESOLUTION_RE.test(raw) && raw !== detected && !baseSet.has(raw)) {
            if (isExact) {
                exact.push({
                    value: currentSelection,
                    label: formatLabel(raw, 'exact'),
                });
            } else {
                minimum.push({
                    value: currentSelection,
                    label: formatLabel(raw, 'minimum'),
                });
            }
        }
    }

    return {minimum, exact};
}
