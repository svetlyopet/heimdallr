export function formatNumber(value) {
    return new Intl.NumberFormat(undefined, {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
    }).format(Number(value || 0));
}

export function getInitial(value) {
    return String(value || "?").charAt(0).toUpperCase();
}