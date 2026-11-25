// Shop module (docs/CRYPTO_BRIDGE.md section 2). Runs inside the crypto Web
// Worker: generates keys, keeps them in IndexedDB, encrypts inputs, talks to
// the server and decrypts results. Only ciphertexts and public artifacts are
// ever sent; the secret key stays in this worker.
import { strFromU8, strToU8, unzipSync, zipSync } from "fflate";
import { b64BinaryLength, middleB64, round2, truncateB64 } from "../format";
import {
  ShopError,
  type ActivateResult,
  type BalanceView,
  type Catalog,
  type CipherView,
  type HistoryResult,
  type ImportResult,
  type LocalUser,
  type Meta,
  type RequestLog,
  type Shop,
  type ShopConfig,
  type TransactResult,
} from "../types";
import { checkBillBound, checkTopupBound, DEFAULT_MAX_VALUE, validateAmount, validateQuantity, validateUserId } from "../validate";
import { loadBridge, type Bridge } from "./bridge";
import { deleteKeys, getKeys, listUsers, putKeys, type KeyRecord } from "./keystore";
import { METADATA_TOLERANCE, recoverMetadata } from "./metadata";
import { ServerApi } from "./server";

export interface ShopInit {
  serverApi: string;
  maxValue?: number;
  defaultProfile?: string;
}

// Names inside the export/import zip (Table 2.3 of the assignment).
export const KEY_FILES = {
  params: "params.json",
  secret_key: "secret.key",
  public_key: "public.key",
  relin_key: "relin.key",
} as const;

const MAX_ZIP_BYTES = 64 << 20;
const MAX_FILE_BYTES = 16 << 20;
const CT_CACHE = 64;

function b64ToBytes(b64: string): Uint8Array {
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}

function bytesToB64(bytes: Uint8Array): string {
  let s = "";
  const chunk = 0x8000;
  for (let i = 0; i < bytes.length; i += chunk) {
    s += String.fromCharCode(...bytes.subarray(i, i + chunk));
  }
  return btoa(s);
}

export function createShop(): Shop & { configure(init: ShopInit): void } {
  let server = new ServerApi("http://localhost:8000");
  let serverBase = "http://localhost:8000";
  let maxValueCfg = DEFAULT_MAX_VALUE;
  let defaultProfile = "p1";
  let bridge: Bridge | null = null;
  let catalog: Catalog | null = null;
  const lastBalance = new Map<string, number>();
  const ctCache = new Map<string, string>();
  let refSeq = 0;

  async function br(): Promise<Bridge> {
    if (!bridge) bridge = await loadBridge();
    return bridge;
  }

  function remember(b64: string): string {
    const ref = `ct-${++refSeq}`;
    ctCache.set(ref, b64);
    while (ctCache.size > CT_CACHE) {
      const oldest = ctCache.keys().next().value;
      if (oldest === undefined) break;
      ctCache.delete(oldest);
    }
    return ref;
  }

  function view(b: Bridge, params: string, b64: string): CipherView {
    let level: number | undefined;
    let log2Scale: number | undefined;
    try {
      const info = b.ciphertextInfo(params, b64);
      level = info.level;
      log2Scale = info.log2Scale;
    } catch {
      // still show preview/size if info fails
    }
    return { preview: truncateB64(b64), preview_body: middleB64(b64), bytes: b64BinaryLength(b64), b64_length: b64.length, ref: remember(b64), level, log2_scale: log2Scale };
  }

  async function keys(userId: string): Promise<KeyRecord> {
    if (!validateUserId(userId).ok) throw new ShopError("invalid_input", "ID pengguna tidak valid");
    const rec = await getKeys(userId);
    if (!rec) throw new ShopError("keys_not_found", `no local keys for ${userId}`);
    return rec;
  }

  async function ensureCatalog(requests: RequestLog[]): Promise<Catalog> {
    if (catalog) return catalog;
    const r = await server.request<Catalog>("GET", "/catalog");
    requests.push(r.log);
    catalog = r.data;
    return catalog;
  }

  function maxValue(b: Bridge, profile?: string): number {
    const p = b.profiles().find((x) => x.name === profile);
    return p ? Math.min(maxValueCfg, p.maxValue) : maxValueCfg;
  }

  function balanceView(b: Bridge, rec: KeyRecord, encBalance: string, meta: Meta): BalanceView {
    const t = performance.now();
    const raw = b.decryptScalar(rec.params, rec.secret_key, encBalance);
    meta.timings.decrypt_ms = (meta.timings.decrypt_ms ?? 0) + (performance.now() - t);
    lastBalance.set(rec.user_id, raw);
    const balance = round2(raw);
    return { user_id: rec.user_id, balance, balance_raw: raw, negative: balance < 0, enc_balance: view(b, rec.params, encBalance), meta };
  }

  const newMeta = (): Meta => ({ requests: [], timings: {} });

  function withLog(err: unknown, meta: Meta): unknown {
    const log = (err as { log?: RequestLog }).log;
    if (log) meta.requests.push(log);
    return err;
  }

  async function fetchBalance(b: Bridge, rec: KeyRecord, meta: Meta): Promise<BalanceView> {
    try {
      const r = await server.request<{ user_id: string; enc_balance: string }>("GET", `/accounts/${rec.user_id}/balance`);
      meta.requests.push(r.log);
      return balanceView(b, rec, r.data.enc_balance, meta);
    } catch (err) {
      throw withLog(err, meta);
    }
  }

  return {
    configure(init: ShopInit) {
      serverBase = init.serverApi.replace(/\/+$/, "");
      server = new ServerApi(serverBase);
      if (init.maxValue && init.maxValue > 0) maxValueCfg = init.maxValue;
      if (init.defaultProfile) defaultProfile = init.defaultProfile;
    },

    async init() {
      await br();
    },

    async getConfig(): Promise<ShopConfig> {
      const b = await br();
      const profiles = b.profiles();
      const mv = Math.min(maxValueCfg, ...profiles.map((p) => p.maxValue));
      return {
        profiles,
        default_profile: profiles.some((p) => p.name === defaultProfile) ? defaultProfile : (profiles[0]?.name ?? "p1"),
        max_value: mv,
        tolerance: METADATA_TOLERANCE,
        server_api: serverBase,
        mock: false,
        bridge_version: b.version,
      };
    },

    async listUsers(): Promise<LocalUser[]> {
      return listUsers();
    },

    async getCatalog(): Promise<Catalog> {
      return ensureCatalog([]);
    },

    async activate(userId: string, profile: string): Promise<ActivateResult> {
      const v = validateUserId(userId);
      if (!v.ok) throw new ShopError("invalid_input", v.message);
      const b = await br();
      const meta = newMeta();
      const t0 = performance.now();

      let rec = await getKeys(userId);
      const generated = !rec;
      if (rec && rec.profile !== profile) {
        throw new ShopError("user_exists_locally", `local keys for ${userId} use profile ${rec.profile}`);
      }
      if (!rec) {
        const kb = b.genKeys(profile);
        b.validateKeys(kb.params, kb.secret_key, kb.public_key, kb.relin_key);
        rec = {
          user_id: userId,
          profile,
          params: kb.params,
          secret_key: kb.secret_key,
          public_key: kb.public_key,
          relin_key: kb.relin_key,
          created_at: new Date().toISOString(),
        };
        await putKeys(rec);
      }
      const te = performance.now();
      const encZero = b.encryptScalar(rec.params, rec.public_key, 0);
      meta.timings.encrypt_ms = performance.now() - te;
      try {
        // The payload carries params, pk, rlk and Enc(0) only - never the secret key.
        const r = await server.request<{ status: string; user_id: string; enc_balance: string }>("POST", "/accounts", {
          user_id: userId,
          params: rec.params,
          public_key: rec.public_key,
          relin_key: rec.relin_key,
          enc_balance: encZero,
        });
        meta.requests.push(r.log);
        const bv = balanceView(b, rec, r.data.enc_balance, meta);
        meta.timings.total_ms = performance.now() - t0;
        return {
          ...bv,
          status: "activated",
          profile: rec.profile,
          artifact_bytes: {
            params: b64BinaryLength(rec.params),
            public_key: b64BinaryLength(rec.public_key),
            relin_key: b64BinaryLength(rec.relin_key),
            secret_key_local: b64BinaryLength(rec.secret_key),
          },
        };
      } catch (err) {
        if (generated) await deleteKeys(userId); // server rejected: drop the new local keys
        throw withLog(err, meta);
      }
    },

    async getBalance(userId: string): Promise<BalanceView> {
      const rec = await keys(userId);
      const b = await br();
      const meta = newMeta();
      const t0 = performance.now();
      const bv = await fetchBalance(b, rec, meta);
      meta.timings.total_ms = performance.now() - t0;
      return bv;
    },

    async topup(userId: string, amount: number): Promise<BalanceView> {
      const rec = await keys(userId);
      const b = await br();
      const limit = maxValue(b, rec.profile);
      const v = validateAmount(amount, limit);
      if (!v.ok) throw new ShopError(amount > limit ? "value_out_of_range" : "invalid_input", v.message);
      const meta = newMeta();
      const t0 = performance.now();
      if (!lastBalance.has(userId)) await fetchBalance(b, rec, meta);
      const guard = checkTopupBound(lastBalance.get(userId), amount, limit);
      if (!guard.ok) throw new ShopError("value_out_of_range", guard.message);

      const te = performance.now();
      const encAmount = b.encryptScalar(rec.params, rec.public_key, amount);
      meta.timings.encrypt_ms = performance.now() - te;
      try {
        const r = await server.request<{ user_id: string; enc_balance: string }>("POST", `/accounts/${userId}/topup`, { enc_amount: encAmount });
        meta.requests.push(r.log);
        const bv = balanceView(b, rec, r.data.enc_balance, meta);
        meta.timings.total_ms = performance.now() - t0;
        return bv;
      } catch (err) {
        throw withLog(err, meta);
      }
    },

    async transact(userId: string, productId: string, quantity: number, discountCode?: string): Promise<TransactResult> {
      const rec = await keys(userId);
      const b = await br();
      const vq = validateQuantity(quantity);
      if (!vq.ok) throw new ShopError("invalid_input", vq.message);
      const meta = newMeta();
      const t0 = performance.now();
      const cat = await ensureCatalog(meta.requests);
      const product = cat.products.find((p) => p.product_id === productId);
      if (!product) throw new ShopError("product_not_found", `unknown product ${productId}`);
      const code = discountCode && discountCode !== "" ? discountCode : "NONE";
      const discount = cat.discounts.find((d) => d.code === code);
      if (!discount) throw new ShopError("discount_not_found", `unknown discount ${code}`);
      const limit = maxValue(b, rec.profile);
      if (!lastBalance.has(userId)) await fetchBalance(b, rec, meta);
      const guard = checkBillBound(product.base_price, quantity, lastBalance.get(userId), limit);
      if (!guard.ok) throw new ShopError("value_out_of_range", guard.message);

      const te = performance.now();
      const encQuantity = b.encryptScalar(rec.params, rec.public_key, quantity);
      const encMetadata = b.encryptVector(rec.params, rec.public_key, [product.numeric_id, quantity, discount.numeric_id]);
      meta.timings.encrypt_ms = performance.now() - te;
      try {
        // product_id and discount_code are public (the server needs them to look up the catalog).
        const r = await server.request<{ transaction_id: string; enc_total: string; enc_balance: string }>(
          "POST",
          `/accounts/${userId}/transactions`,
          { product_id: productId, discount_code: code, enc_quantity: encQuantity, enc_metadata: encMetadata },
        );
        meta.requests.push(r.log);
        const td = performance.now();
        const totalRaw = b.decryptScalar(rec.params, rec.secret_key, r.data.enc_total);
        meta.timings.decrypt_ms = performance.now() - td;
        const bv = balanceView(b, rec, r.data.enc_balance, meta);
        meta.timings.total_ms = performance.now() - t0;
        return {
          ...bv,
          transaction_id: r.data.transaction_id,
          product_id: productId,
          quantity,
          discount_code: code,
          total: round2(totalRaw),
          total_raw: totalRaw,
          enc_total: view(b, rec.params, r.data.enc_total),
          enc_quantity: view(b, rec.params, encQuantity),
          enc_metadata: view(b, rec.params, encMetadata),
          warnings: bv.negative ? ["insufficient_balance"] : [],
        };
      } catch (err) {
        throw withLog(err, meta);
      }
    },

    async getHistory(userId: string): Promise<HistoryResult> {
      const rec = await keys(userId);
      const b = await br();
      const meta = newMeta();
      const t0 = performance.now();
      const cat = await ensureCatalog(meta.requests);
      let data: { user_id: string; transactions: { transaction_id: string; enc_metadata: string; enc_total: string; created_at: string }[] };
      try {
        const r = await server.request<typeof data>("GET", `/accounts/${userId}/transactions`);
        meta.requests.push(r.log);
        data = r.data;
      } catch (err) {
        throw withLog(err, meta);
      }
      const td = performance.now();
      const items = data.transactions.map((t) => {
        const metaRaw = b.decryptVector(rec.params, rec.secret_key, t.enc_metadata, 3);
        const totalRaw = b.decryptScalar(rec.params, rec.secret_key, t.enc_total);
        const m = recoverMetadata(metaRaw);
        const product = cat.products.find((p) => p.numeric_id === m.productNum);
        const discount = cat.discounts.find((d) => d.numeric_id === m.discountNum);
        return {
          transaction_id: t.transaction_id,
          created_at: t.created_at,
          product_id: product?.product_id ?? `#${m.productNum}`,
          quantity: m.quantity,
          discount_code: discount?.code ?? `#${m.discountNum}`,
          metadata_raw: metaRaw,
          total: round2(totalRaw),
          total_raw: totalRaw,
          enc_total: view(b, rec.params, t.enc_total),
          enc_metadata: view(b, rec.params, t.enc_metadata),
          warning: m.uncertain || !product || !discount ? ("metadata_uncertain" as const) : null,
        };
      });
      meta.timings.decrypt_ms = performance.now() - td;
      meta.timings.total_ms = performance.now() - t0;
      return { user_id: userId, items, meta };
    },

    async exportKeys(userId: string): Promise<Blob> {
      const rec = await keys(userId);
      // The zip CONTAINS THE SECRET KEY: it is only offered as a local download.
      const zip = zipSync(
        {
          [KEY_FILES.params]: b64ToBytes(rec.params),
          [KEY_FILES.secret_key]: b64ToBytes(rec.secret_key),
          [KEY_FILES.public_key]: b64ToBytes(rec.public_key),
          [KEY_FILES.relin_key]: b64ToBytes(rec.relin_key),
        },
        { level: 6 },
      );
      return new Blob([zip as BlobPart], { type: "application/zip" });
    },

    async importKeys(userId: string, file: Blob): Promise<ImportResult> {
      const v = validateUserId(userId);
      if (!v.ok) throw new ShopError("invalid_input", v.message);
      if (await getKeys(userId)) throw new ShopError("user_exists_locally", `local keys for ${userId} already exist`);
      if (file.size === 0 || file.size > MAX_ZIP_BYTES) throw new ShopError("keys_corrupt", `zip size ${file.size} not accepted`);
      const b = await br();
      let entries: Record<string, Uint8Array>;
      try {
        const allowed = new Set<string>(Object.values(KEY_FILES));
        const unknown: string[] = [];
        entries = unzipSync(new Uint8Array(await file.arrayBuffer()), {
          // only the four expected names are extracted (prevents zip-slip and oversized entries)
          filter: (f) => {
            if (!allowed.has(f.name)) {
              if (!f.name.endsWith("/")) unknown.push(f.name);
              return false;
            }
            return f.originalSize <= MAX_FILE_BYTES;
          },
        });
        if (unknown.length > 0) throw new ShopError("keys_corrupt", `unexpected files in zip: ${unknown.join(", ")}`);
      } catch (err) {
        if (err instanceof ShopError) throw err;
        throw new ShopError("keys_corrupt", `cannot read zip: ${err instanceof Error ? err.message : String(err)}`);
      }
      for (const name of Object.values(KEY_FILES)) {
        if (!entries[name] || entries[name].length === 0) throw new ShopError("keys_corrupt", `missing ${name}`);
      }
      // params.json must be Lattigo JSON parameters; keep the exact bytes
      try {
        JSON.parse(strFromU8(entries[KEY_FILES.params]));
      } catch {
        throw new ShopError("keys_corrupt", "params.json is not valid JSON");
      }
      const rec: KeyRecord = {
        user_id: userId,
        profile: "",
        params: bytesToB64(entries[KEY_FILES.params]),
        secret_key: bytesToB64(entries[KEY_FILES.secret_key]),
        public_key: bytesToB64(entries[KEY_FILES.public_key]),
        relin_key: bytesToB64(entries[KEY_FILES.relin_key]),
        created_at: new Date().toISOString(),
      };
      try {
        rec.profile = b.paramsInfo(rec.params).profile;
        b.validateKeys(rec.params, rec.secret_key, rec.public_key, rec.relin_key);
      } catch (err) {
        if (err instanceof ShopError && err.code === "keys_mismatch") throw err;
        throw new ShopError("keys_corrupt", err instanceof Error ? err.message : String(err));
      }
      await putKeys(rec);
      lastBalance.delete(userId);
      return { status: "imported", user_id: userId, profile: rec.profile };
    },

    async deleteLocalKeys(userId: string): Promise<void> {
      if (!validateUserId(userId).ok) throw new ShopError("invalid_input", "ID pengguna tidak valid");
      await deleteKeys(userId);
      lastBalance.delete(userId);
    },

    async getCiphertext(ref: string): Promise<string> {
      const v = ctCache.get(ref);
      if (!v) throw new ShopError("invalid_input", "ciphertext no longer cached; refresh the view");
      return v;
    },
  };
}

// exported for tests
export const _internal = { b64ToBytes, bytesToB64, strToU8 };
