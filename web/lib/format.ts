// Display helpers. Rounding here is for display only; raw decrypted values
// are kept separately (balance_raw, total_raw) and shown alongside.

const rupiah = new Intl.NumberFormat("id-ID", {
  style: "currency",
  currency: "IDR",
  minimumFractionDigits: 2,
  maximumFractionDigits: 2,
});

/** Formats an amount as Indonesian Rupiah with 2 decimals, e.g. "Rp 78.000,00". */
export function formatRupiah(value: number): string {
  if (!Number.isFinite(value)) return "—";
  const rounded = Math.round(value * 100) / 100;
  // normalize -0 so it does not render as "-Rp 0,00"
  const v = rounded === 0 ? 0 : rounded;
  return rupiah.format(v).replace(/ /g, " ");
}

export const PREVIEW_CHARS = 60;

/** First `n` Base64 characters followed by "..." (unchanged if short). */
export function truncateB64(b64: string, n = PREVIEW_CHARS): string {
  if (b64.length <= n) return b64;
  return b64.slice(0, n) + "...";
}

/** 60 Base64 characters taken from the middle of the string (coefficient data), with ellipses. */
export function middleB64(b64: string, n = PREVIEW_CHARS): string {
  if (b64.length <= n * 2) return truncateB64(b64, n);
  const mid = Math.floor(b64.length / 8) * 4;
  return "..." + b64.slice(mid, mid + n) + "...";
}

/** Human-readable size: bytes, KiB or MiB (binary units). */
export function formatBytes(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "—";
  if (bytes < 1024) return `${bytes} B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KiB`;
  return `${(bytes / (1024 * 1024)).toFixed(2)} MiB`;
}

/** Raw CKKS value with enough digits to show the approximation error. */
export function formatRaw(value: number): string {
  if (!Number.isFinite(value)) return String(value);
  return value.toPrecision(15);
}

/** Rounds to 2 decimals (display/money). */
export function round2(value: number): number {
  const r = Math.round(value * 100) / 100;
  return r === 0 ? 0 : r;
}

/** Binary length of a standard padded Base64 string. */
export function b64BinaryLength(b64: string): number {
  const pad = b64.endsWith("==") ? 2 : b64.endsWith("=") ? 1 : 0;
  return (b64.length / 4) * 3 - pad;
}

export function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString("id-ID", { dateStyle: "short", timeStyle: "medium" });
}
