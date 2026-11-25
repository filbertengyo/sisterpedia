// Single GUI-facing API. All CKKS work happens in the crypto worker
// (lib/crypto/*); this module only picks the real shop or the mock and
// offers a tiny public-data client (catalog, health) for the main thread.
import { createMockShop } from "./mock";
import { createWorkerShop } from "./crypto/client";
import type { Catalog, Shop } from "./types";

export const SERVER_API = (process.env.NEXT_PUBLIC_SERVER_API ?? "http://localhost:8000").replace(/\/+$/, "");
export const MOCK = process.env.NEXT_PUBLIC_MOCK === "1" || process.env.NEXT_PUBLIC_MOCK === "true";
export const MAX_VALUE = Number(process.env.NEXT_PUBLIC_MAX_VALUE ?? "500000") || 500000;
export const DEFAULT_PROFILE = process.env.NEXT_PUBLIC_DEFAULT_PROFILE ?? "p1";

let shop: Shop | null = null;

/** Returns the shop (created lazily, browser only). */
export function getShop(): Shop {
  if (!shop) {
    shop = MOCK ? createMockShop() : createWorkerShop({ serverApi: SERVER_API, maxValue: MAX_VALUE, defaultProfile: DEFAULT_PROFILE });
  }
  return shop;
}

/** Public, non-sensitive server endpoints callable from the main thread. */
export const serverApi = {
  async health(): Promise<boolean> {
    if (MOCK) return true;
    try {
      const r = await fetch(`${SERVER_API}/healthz`, { cache: "no-store" });
      return r.ok;
    } catch {
      return false;
    }
  },
  async getCatalog(): Promise<Catalog> {
    if (MOCK) return getShop().getCatalog();
    const r = await fetch(`${SERVER_API}/catalog`, { cache: "no-store" });
    if (!r.ok) throw new Error(`catalog: HTTP ${r.status}`);
    return (await r.json()) as Catalog;
  },
};
