// Input validation shared by the GUI forms and the shop module.

export const USER_ID_PATTERN = /^[a-z0-9_-]{3,32}$/;
export const DEFAULT_MAX_VALUE = 500000;
export const MAX_QUANTITY = 1000;

export type Validation = { ok: true } | { ok: false; message: string };

const ok: Validation = { ok: true };
const bad = (message: string): Validation => ({ ok: false, message });
const id = (n: number) => n.toLocaleString("id-ID");

export function validateUserId(userId: string): Validation {
  if (!USER_ID_PATTERN.test(userId)) {
    return bad("ID pengguna harus 3-32 karakter: huruf kecil, angka, garis bawah atau tanda hubung.");
  }
  return ok;
}

/** Parses an amount typed by the user; accepts "12345.67" and "12345,67" (max 2 decimals). */
export function parseAmount(text: string): number {
  const t = text.trim().replace(/\s/g, "");
  if (!/^\d+([.,]\d{1,2})?$/.test(t)) return Number.NaN;
  return Number(t.replace(",", "."));
}

export function validateAmount(amount: number, maxValue = DEFAULT_MAX_VALUE): Validation {
  if (!Number.isFinite(amount)) return bad("Nominal tidak valid (gunakan angka, maksimal 2 desimal).");
  if (amount <= 0) return bad("Nominal harus lebih dari 0.");
  if (amount > maxValue) return bad(`Nominal melebihi batas ${id(maxValue)}.`);
  return ok;
}

export function validateQuantity(q: number, maxQuantity = MAX_QUANTITY): Validation {
  if (!Number.isInteger(q)) return bad("Kuantitas harus bilangan bulat.");
  if (q < 1) return bad("Kuantitas minimal 1.");
  if (q > maxQuantity) return bad(`Kuantitas maksimal ${maxQuantity}.`);
  return ok;
}

/**
 * Overflow guard: CKKS silently returns wrong values once |x| * scale exceeds
 * q0/2 at the lowest level (see docs/PARAMS.md). The client therefore keeps
 * the balance within +-maxValue, using the last balance it decrypted.
 */
export function checkTopupBound(lastBalance: number | undefined, amount: number, maxValue = DEFAULT_MAX_VALUE): Validation {
  if (lastBalance !== undefined && Math.abs(lastBalance + amount) > maxValue) {
    return bad(`Saldo setelah top up (${id(lastBalance + amount)}) akan melebihi batas aman ${id(maxValue)}.`);
  }
  return ok;
}

export function checkBillBound(price: number, quantity: number, lastBalance: number | undefined, maxValue = DEFAULT_MAX_VALUE): Validation {
  const subtotal = price * quantity;
  if (subtotal > maxValue) {
    return bad(`Subtotal ${id(subtotal)} melebihi batas aman ${id(maxValue)}.`);
  }
  if (lastBalance !== undefined && Math.abs(lastBalance - subtotal) > maxValue) {
    return bad(`Saldo setelah transaksi akan melewati batas aman -${id(maxValue)}.`);
  }
  return ok;
}

/** Plaintext estimate of a bill, computed locally on the client only. */
export function estimateBill(price: number, quantity: number, percent: number, fixed: number): number {
  return price * quantity * (1 - percent) - fixed;
}
