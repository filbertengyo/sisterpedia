// Types shared by the GUI, the shop module (worker) and the mock.
// See docs/CRYPTO_BRIDGE.md section 2.

/** Display form of a ciphertext: Base64 preview, binary size and a handle to copy it in full. */
export interface CipherView {
  preview: string; // first 60 Base64 characters + "..." (starts with the library metadata header)
  preview_body: string; // 60 characters from the middle (polynomial coefficients), differs per ciphertext
  bytes: number; // binary (pre-Base64) size
  b64_length: number;
  ref: string; // handle for getCiphertext(ref)
  level?: number;
  log2_scale?: number;
}

/** One HTTP exchange with the server, for the GUI log panel (never values). */
export interface RequestLog {
  method: string;
  path: string;
  status: number;
  ms: number;
  request_bytes: number;
  response_bytes: number;
  server_eval_ms?: number;
}

export interface Timings {
  encrypt_ms?: number;
  decrypt_ms?: number;
  total_ms?: number;
}

export interface Meta {
  requests: RequestLog[];
  timings: Timings;
}

export interface BalanceView {
  user_id: string;
  balance: number; // rounded to 2 decimals for display
  balance_raw: number; // raw decrypted CKKS value
  negative: boolean;
  enc_balance: CipherView;
  meta?: Meta;
}

export interface ActivateResult extends BalanceView {
  status: "activated";
  profile: string;
  artifact_bytes: { params: number; public_key: number; relin_key: number; secret_key_local: number };
}

export type TransactWarning = "insufficient_balance";

export interface TransactResult extends BalanceView {
  transaction_id: string;
  product_id: string;
  quantity: number;
  discount_code: string;
  total: number;
  total_raw: number;
  enc_total: CipherView;
  enc_quantity: CipherView;
  enc_metadata: CipherView;
  warnings: TransactWarning[];
}

export interface HistoryItem {
  transaction_id: string;
  created_at: string;
  product_id: string;
  quantity: number;
  discount_code: string;
  metadata_raw: number[];
  total: number;
  total_raw: number;
  enc_total: CipherView;
  enc_metadata: CipherView;
  warning: "metadata_uncertain" | null;
}

export interface HistoryResult {
  user_id: string;
  items: HistoryItem[];
  meta?: Meta;
}

export interface Product {
  product_id: string;
  numeric_id: number;
  name: string;
  base_price: number;
}

export interface Discount {
  code: string;
  numeric_id: number;
  percent: number;
  fixed: number;
}

export interface Catalog {
  products: Product[];
  discounts: Discount[];
}

export interface ProfileInfo {
  name: string;
  logN: number;
  maxLevel: number;
  slots: number;
  logDefaultScale: number;
  logQP: number;
  maxValue: number;
}

export interface ShopConfig {
  profiles: ProfileInfo[];
  default_profile: string;
  max_value: number;
  tolerance: number;
  server_api: string;
  mock: boolean;
  bridge_version: string;
}

export interface LocalUser {
  user_id: string;
  profile: string;
  created_at: string;
}

export interface ImportResult {
  status: "imported";
  user_id: string;
  profile: string;
}

/** The API the GUI uses (real shop module in a worker, or the mock). */
export interface Shop {
  init(): Promise<void>;
  getConfig(): Promise<ShopConfig>;
  listUsers(): Promise<LocalUser[]>;
  getCatalog(): Promise<Catalog>;
  activate(userId: string, profile: string): Promise<ActivateResult>;
  getBalance(userId: string): Promise<BalanceView>;
  topup(userId: string, amount: number): Promise<BalanceView>;
  transact(userId: string, productId: string, quantity: number, discountCode?: string): Promise<TransactResult>;
  getHistory(userId: string): Promise<HistoryResult>;
  exportKeys(userId: string): Promise<Blob>;
  importKeys(userId: string, file: Blob): Promise<ImportResult>;
  deleteLocalKeys(userId: string): Promise<void>;
  getCiphertext(ref: string): Promise<string>;
}

export const SHOP_ERROR_CODES = [
  "invalid_input",
  "value_out_of_range",
  "keys_not_found",
  "keys_corrupt",
  "keys_mismatch",
  "user_exists_locally",
  "account_not_found",
  "account_exists",
  "product_not_found",
  "discount_not_found",
  "evaluation_failed",
  "invalid_payload",
  "invalid_json",
  "context_mismatch",
  "secret_key_rejected",
  "payload_too_large",
  "server_unreachable",
  "wasm_load_failed",
  "params_mismatch",
  "internal_error",
] as const;

export type ShopErrorCode = (typeof SHOP_ERROR_CODES)[number];

/** Error thrown by the shop module; crosses the worker boundary as {code, message}. */
export class ShopError extends Error {
  code: ShopErrorCode | string;
  status?: number;
  constructor(code: ShopErrorCode | string, message: string, status?: number) {
    super(message);
    this.name = "ShopError";
    this.code = code;
    this.status = status;
  }
}
