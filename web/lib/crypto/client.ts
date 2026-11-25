// Main-thread proxy of the shop module running in the crypto worker. The GUI
// only sees results (decrypted values + ciphertext previews); it never
// touches keys or the raw WASM bridge.
import { ShopError, type RequestLog, type Shop } from "../types";
import type { ShopInit } from "./shop";

interface Pending {
  resolve: (v: unknown) => void;
  reject: (e: unknown) => void;
}

export function createWorkerShop(init: ShopInit): Shop {
  let worker: Worker | null = null;
  let seq = 0;
  const pending = new Map<number, Pending>();

  function ensureWorker(): Worker {
    if (worker) return worker;
    worker = new Worker(new URL("./worker.ts", import.meta.url), { type: "module", name: "sisterpedia-crypto" });
    worker.onmessage = (e: MessageEvent<{ id: number; ok: boolean; value?: unknown; error?: { code: string; message: string; status?: number; log?: RequestLog } }>) => {
      const p = pending.get(e.data.id);
      if (!p) return;
      pending.delete(e.data.id);
      if (e.data.ok) p.resolve(e.data.value);
      else {
        const err = new ShopError(e.data.error?.code ?? "internal_error", e.data.error?.message ?? "worker error", e.data.error?.status);
        (err as ShopError & { log?: RequestLog }).log = e.data.error?.log;
        p.reject(err);
      }
    };
    worker.onerror = (e) => {
      const err = new ShopError("wasm_load_failed", e.message || "crypto worker failed");
      for (const p of pending.values()) p.reject(err);
      pending.clear();
    };
    void call("configure", init);
    return worker;
  }

  function call<T>(method: string, ...args: unknown[]): Promise<T> {
    const w = ensureWorker();
    const id = ++seq;
    return new Promise<T>((resolve, reject) => {
      pending.set(id, { resolve: resolve as (v: unknown) => void, reject });
      w.postMessage({ id, method, args });
    });
  }

  return {
    init: () => call("init"),
    getConfig: () => call("getConfig"),
    listUsers: () => call("listUsers"),
    getCatalog: () => call("getCatalog"),
    activate: (u, p) => call("activate", u, p),
    getBalance: (u) => call("getBalance", u),
    topup: (u, a) => call("topup", u, a),
    transact: (u, p, q, d) => call("transact", u, p, q, d),
    getHistory: (u) => call("getHistory", u),
    exportKeys: (u) => call("exportKeys", u),
    importKeys: (u, f) => call("importKeys", u, f),
    deleteLocalKeys: (u) => call("deleteLocalKeys", u),
    getCiphertext: (r) => call("getCiphertext", r),
  };
}
