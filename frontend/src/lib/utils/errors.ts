// Wails rejects a failed Go call with the error string. Code that throws
// uses Error objects. Returns '' when the value has no text.
export function errorMessage(error: unknown): string {
    let text = '';
    if (typeof error === 'string') text = error;
    else if (error instanceof Error) text = error.message;
    return text
        .trim()
        .replace(/^Error:\s*/, '')
        .trim();
}

// Returns `${summary}: ${message}`, or `summary` alone when the error has no text.
export function failureMessage(summary: string, error?: unknown): string {
    const message = errorMessage(error);
    return message ? `${summary}: ${message}` : summary;
}
