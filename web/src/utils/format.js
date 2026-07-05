export function formatNumber(value) {
    return new Intl.NumberFormat(undefined, {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
    }).format(Number(value || 0));
}

export function formatPercent(value, { scale = "percent" } = {}) {
    const number = Number(value || 0);
    const percent = scale === "ratio" ? number * 100 : number;
    return `${percent.toFixed(2)}%`;
}

export function formatDateTime(value) {
    if (!value) return "—";
    const date = new Date(value);
    if (Number.isNaN(date.getTime())) return "—";
    return date.toLocaleString();
}

export function getInitial(value) {
    return String(value || "?").charAt(0).toUpperCase();
}