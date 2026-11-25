// Maps shop/server error codes to Indonesian messages for the GUI.

const MESSAGES: Record<string, string> = {
  invalid_input: "Masukan tidak valid.",
  value_out_of_range: "Nilai di luar batas aman CKKS (nilai sebesar ini dapat overflow tanpa pesan galat).",
  keys_not_found: "Kunci untuk pengguna ini tidak ada di perangkat. Impor berkas kunci terlebih dahulu.",
  keys_corrupt: "Berkas kunci rusak, tidak lengkap, atau berisi berkas yang tidak dikenal.",
  keys_mismatch: "Kunci tidak cocok satu sama lain (public/relin key bukan pasangan secret key ini).",
  user_exists_locally: "Kunci pengguna ini sudah ada di perangkat. Hapus kunci lokal dulu bila ingin menggantinya.",
  account_not_found: "Akun belum diaktifkan di server (HTTP 404).",
  account_exists: "Akun dengan ID ini sudah aktif di server (HTTP 409).",
  product_not_found: "Produk tidak ada di katalog (HTTP 404).",
  discount_not_found: "Kode diskon tidak dikenal (HTTP 404).",
  evaluation_failed: "Server gagal mengevaluasi ciphertext (HTTP 400).",
  invalid_payload: "Ciphertext atau kunci tidak dapat dimuat (HTTP 400).",
  invalid_json: "Format permintaan tidak valid (HTTP 400).",
  context_mismatch: "Parameter atau ciphertext tidak cocok dengan profil akun (HTTP 400).",
  secret_key_rejected: "Server menolak muatan yang berisi secret key (HTTP 400).",
  payload_too_large: "Muatan terlalu besar (HTTP 413).",
  server_unreachable: "Server tidak dapat dihubungi. Pastikan server berjalan.",
  wasm_load_failed: "Modul kriptografi (WebAssembly) gagal dimuat.",
  params_mismatch: "Parameter CKKS tidak dikenali.",
  internal_error: "Terjadi galat internal.",
};

export interface DisplayError {
  code: string;
  message: string;
  detail?: string;
}

/** Converts anything thrown by the shop into a code + Indonesian message. */
export function describeError(err: unknown): DisplayError {
  if (typeof err === "object" && err !== null && "code" in err && typeof (err as { code: unknown }).code === "string") {
    const e = err as { code: string; message?: string };
    const message = MESSAGES[e.code] ?? "Terjadi galat.";
    const detail = e.message && e.message !== message ? e.message : undefined;
    return { code: e.code, message, detail };
  }
  if (err instanceof Error) return { code: "internal_error", message: MESSAGES.internal_error, detail: err.message };
  return { code: "internal_error", message: MESSAGES.internal_error, detail: String(err) };
}

export function messageFor(code: string): string {
  return MESSAGES[code] ?? "Terjadi galat.";
}
