// Crypto Web Worker entry point: dispatches messages from the main thread to
// the shop module. Heavy CKKS work (keygen, relin key, p2) runs here so the
// UI never freezes, and the keystore (secret keys) is only reachable here.
import { createShop, type ShopInit } from "./shop";

interface Request {
  id: number;
  method: string;
  args: unknown[];
}

const shop = createShop();
const scope = self as unknown as {
  onmessage: ((e: MessageEvent<Request>) => void) | null;
  postMessage: (msg: unknown) => void;
};

const METHODS = new Set([
  "init",
  "getConfig",
  "listUsers",
  "getCatalog",
  "activate",
  "getBalance",
  "topup",
  "transact",
  "getHistory",
  "exportKeys",
  "importKeys",
  "deleteLocalKeys",
  "getCiphertext",
]);

scope.onmessage = async (e: MessageEvent<Request>) => {
  const { id, method, args } = e.data;
  try {
    let value: unknown;
    if (method === "configure") {
      shop.configure(args[0] as ShopInit);
    } else if (METHODS.has(method)) {
      const fn = (shop as unknown as Record<string, (...a: unknown[]) => Promise<unknown>>)[method];
      value = await fn(...args);
    } else {
      throw Object.assign(new Error(`unknown method ${method}`), { code: "invalid_input" });
    }
    scope.postMessage({ id, ok: true, value });
  } catch (err) {
    const e2 = err as { code?: string; message?: string; status?: number; log?: unknown };
    scope.postMessage({
      id,
      ok: false,
      error: { code: e2.code ?? "internal_error", message: e2.message ?? String(err), status: e2.status, log: e2.log },
    });
  }
};
