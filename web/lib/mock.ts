// Mock shop (NEXT_PUBLIC_MOCK=1): same interface as the real shop module but
// with fake ciphertext previews and in-memory state, so the GUI can be
// developed and demoed without the server or the WASM bundle. No crypto.
import { round2 } from "./format";
import {
  ShopError,
  type ActivateResult,
  type BalanceView,
  type Catalog,
  type CipherView,
  type HistoryItem,
  type HistoryResult,
  type ImportResult,
  type LocalUser,
  type Meta,
  type Shop,
  type ShopConfig,
  type TransactResult,
} from "./types";
import { checkBillBound, checkTopupBound, DEFAULT_MAX_VALUE, validateAmount, validateQuantity, validateUserId } from "./validate";

const CATALOG: Catalog = {
  products: [
    { product_id: "P001", numeric_id: 1, name: "Chocolate Cake", base_price: 12500 },
    { product_id: "P002", numeric_id: 2, name: "Cheese Cake", base_price: 15000 },
    { product_id: "P003", numeric_id: 3, name: "Strawberry Cake", base_price: 17500 },
    { product_id: "P004", numeric_id: 4, name: "White Bread", base_price: 9000 },
    { product_id: "P005", numeric_id: 5, name: "Bottled Tea", base_price: 5000 },
  ],
  discounts: [
    { code: "NONE", numeric_id: 0, percent: 0, fixed: 0 },
    { code: "DISC10", numeric_id: 1, percent: 0.1, fixed: 0 },
    { code: "FLAT500", numeric_id: 2, percent: 0, fixed: 500 },
    { code: "COMBO", numeric_id: 3, percent: 0.1, fixed: 500 },
  ],
};

const PROFILES = [
  { name: "p1", logN: 13, maxLevel: 2, slots: 4096, logDefaultScale: 40, logQP: 201, maxValue: 524287 },
  { name: "p2", logN: 14, maxLevel: 3, slots: 8192, logDefaultScale: 40, logQP: 241, maxValue: 524287 },
];

// binary sizes measured for the real profiles (per level)
const CT_BYTES: Record<string, number[]> = { p1: [131_390, 262_478, 393_566], p2: [262_462, 524_622, 786_782, 1_048_942] };

const B64 = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/";

function fakeB64(n = 80): string {
  let s = "";
  for (let i = 0; i < n; i++) s += B64[Math.floor(Math.random() * 64)];
  return s;
}

interface MockAccount {
  profile: string;
  balance: number;
  history: { id: string; created_at: string; product_id: string; quantity: number; discount_code: string; total: number }[];
}

export function createMockShop(): Shop {
  const catalog = CATALOG;
  const local = new Map<string, LocalUser>();
  const server = new Map<string, MockAccount>();
  const cts = new Map<string, string>();
  let seq = 0;

  const delay = (ms = 120) => new Promise((r) => setTimeout(r, ms));
  const meta = (path: string, method = "GET", status = 200): Meta => ({
    requests: [{ method, path, status, ms: 42, request_bytes: method === "POST" ? 524_000 : 0, response_bytes: 524_000 }],
    timings: { encrypt_ms: 9, decrypt_ms: 3, total_ms: 60 },
  });
  const ct = (profile: string, level: number): CipherView => {
    const b64 = fakeB64(4000);
    const ref = `mock-${++seq}`;
    cts.set(ref, b64);
    return { preview: b64.slice(0, 60) + "...", preview_body: "..." + b64.slice(2000, 2060) + "...", bytes: CT_BYTES[profile][level], b64_length: Math.ceil(CT_BYTES[profile][level] / 3) * 4, ref, level, log2_scale: 40 };
  };
  const noise = () => (Math.random() - 0.5) * 2e-5;

  function need(userId: string): MockAccount {
    if (!local.has(userId)) throw new ShopError("keys_not_found", `no local keys for ${userId}`);
    const a = server.get(userId);
    if (!a) throw new ShopError("account_not_found", `account ${userId} is not activated`, 404);
    return a;
  }

  function bv(userId: string, a: MockAccount, path: string, method = "GET"): BalanceView {
    const maxL = a.profile === "p1" ? 2 : 3;
    const level = a.history.length > 0 ? maxL - 2 : maxL;
    const raw = a.balance + noise();
    return { user_id: userId, balance: round2(raw), balance_raw: raw, negative: round2(raw) < 0, enc_balance: ct(a.profile, level), meta: meta(path, method) };
  }

  return {
    async init() {},
    async getConfig(): Promise<ShopConfig> {
      return { profiles: PROFILES, default_profile: "p1", max_value: DEFAULT_MAX_VALUE, tolerance: 0.25, server_api: "mock", mock: true, bridge_version: "mock" };
    },
    async listUsers() {
      return [...local.values()];
    },
    async getCatalog() {
      return catalog;
    },
    async activate(userId, profile): Promise<ActivateResult> {
      const v = validateUserId(userId);
      if (!v.ok) throw new ShopError("invalid_input", v.message);
      await delay(400);
      if (server.has(userId)) throw new ShopError("account_exists", `account ${userId} exists`, 409);
      local.set(userId, { user_id: userId, profile, created_at: new Date().toISOString() });
      const a: MockAccount = { profile, balance: 0, history: [] };
      server.set(userId, a);
      const big = profile === "p1";
      return {
        ...bv(userId, a, "/accounts", "POST"),
        status: "activated",
        profile,
        artifact_bytes: { params: 520, public_key: big ? 524_350 : 1_310_830, relin_key: big ? 1_573_150 : 5_243_170, secret_key_local: big ? 262_210 : 655_440 },
      };
    },
    async getBalance(userId) {
      const a = need(userId);
      await delay();
      return bv(userId, a, `/accounts/${userId}/balance`);
    },
    async topup(userId, amount) {
      const a = need(userId);
      const v = validateAmount(amount);
      if (!v.ok) throw new ShopError("invalid_input", v.message);
      const g = checkTopupBound(a.balance, amount);
      if (!g.ok) throw new ShopError("value_out_of_range", g.message);
      await delay();
      a.balance += amount;
      return bv(userId, a, `/accounts/${userId}/topup`, "POST");
    },
    async transact(userId, productId, quantity, discountCode): Promise<TransactResult> {
      const a = need(userId);
      const vq = validateQuantity(quantity);
      if (!vq.ok) throw new ShopError("invalid_input", vq.message);
      const p = catalog.products.find((x) => x.product_id === productId);
      if (!p) throw new ShopError("product_not_found", productId, 404);
      const code = discountCode || "NONE";
      const d = catalog.discounts.find((x) => x.code === code);
      if (!d) throw new ShopError("discount_not_found", code, 404);
      const g = checkBillBound(p.base_price, quantity, a.balance);
      if (!g.ok) throw new ShopError("value_out_of_range", g.message);
      await delay(200);
      const total = p.base_price * quantity * (1 - d.percent) - d.fixed;
      a.balance -= total;
      const id = `MOCK${Date.now()}${++seq}`;
      a.history.push({ id, created_at: new Date().toISOString(), product_id: productId, quantity, discount_code: code, total });
      const b = bv(userId, a, `/accounts/${userId}/transactions`, "POST");
      const maxL = a.profile === "p1" ? 2 : 3;
      const raw = total + noise();
      return {
        ...b,
        transaction_id: id,
        product_id: productId,
        quantity,
        discount_code: code,
        total: round2(raw),
        total_raw: raw,
        enc_total: ct(a.profile, maxL - 2),
        enc_quantity: ct(a.profile, maxL),
        enc_metadata: ct(a.profile, maxL),
        warnings: b.negative ? ["insufficient_balance"] : [],
      };
    },
    async getHistory(userId): Promise<HistoryResult> {
      const a = need(userId);
      await delay();
      const maxL = a.profile === "p1" ? 2 : 3;
      const items: HistoryItem[] = a.history.map((h) => {
        const p = catalog.products.find((x) => x.product_id === h.product_id)!;
        const d = catalog.discounts.find((x) => x.code === h.discount_code)!;
        const raw = h.total + noise();
        return {
          transaction_id: h.id,
          created_at: h.created_at,
          product_id: h.product_id,
          quantity: h.quantity,
          discount_code: h.discount_code,
          metadata_raw: [p.numeric_id + noise(), h.quantity + noise(), d.numeric_id + noise()],
          total: round2(raw),
          total_raw: raw,
          enc_total: ct(a.profile, maxL - 2),
          enc_metadata: ct(a.profile, maxL),
          warning: null,
        };
      });
      return { user_id: userId, items, meta: meta(`/accounts/${userId}/transactions`) };
    },
    async exportKeys(userId) {
      if (!local.has(userId)) throw new ShopError("keys_not_found", userId);
      return new Blob([`mock key bundle for ${userId} (no real keys in mock mode)`], { type: "application/zip" });
    },
    async importKeys(userId): Promise<ImportResult> {
      if (local.has(userId)) throw new ShopError("user_exists_locally", userId);
      const profile = server.get(userId)?.profile ?? "p1";
      local.set(userId, { user_id: userId, profile, created_at: new Date().toISOString() });
      return { status: "imported", user_id: userId, profile };
    },
    async deleteLocalKeys(userId) {
      local.delete(userId);
    },
    async getCiphertext(ref) {
      const v = cts.get(ref);
      if (!v) throw new ShopError("invalid_input", "ciphertext no longer cached");
      return v;
    },
  };
}
