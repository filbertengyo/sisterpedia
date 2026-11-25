// Loads the Go/Lattigo WebAssembly bridge (cmd/wasm) inside the worker and
// exposes typed wrappers. Every bridge function returns a JSON envelope
// {"ok":true,"value":...} | {"ok":false,"error":{code,message}}.
import { ShopError, type ProfileInfo } from "../types";

export interface KeyBundle {
  profile: string;
  params: string;
  secret_key: string;
  public_key: string;
  relin_key: string;
}

export interface CiphertextInfo {
  level: number;
  log2Scale: number;
  bytes: number;
}

export interface Bridge {
  version: string;
  profiles(): ProfileInfo[];
  paramsInfo(params: string): ProfileInfo & { profile: string };
  genKeys(profile: string): KeyBundle;
  validateKeys(params: string, sk: string, pk: string, rlk?: string): boolean;
  encryptScalar(params: string, pk: string, x: number): string;
  encryptVector(params: string, pk: string, values: number[]): string;
  decryptScalar(params: string, sk: string, ct: string): number;
  decryptVector(params: string, sk: string, ct: string, n: number): number[];
  ciphertextInfo(params: string, ct: string): CiphertextInfo;
}

type RawFn = (...args: unknown[]) => string;
interface RawApi {
  version: string;
  initError?: string;
  [name: string]: unknown;
}

interface GoInstance {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
}

type GoCtor = new () => GoInstance;

interface WasmGlobal {
  Go?: GoCtor;
  sisterpedia?: RawApi;
  importScripts?: (...urls: string[]) => void;
  location?: { origin: string };
}

let loading: Promise<Bridge> | null = null;

function wrap(raw: RawApi): Bridge {
  const call = <T>(name: string, ...args: unknown[]): T => {
    const fn = raw[name] as RawFn | undefined;
    if (typeof fn !== "function") throw new ShopError("wasm_load_failed", `bridge function ${name} missing`);
    const res = JSON.parse(fn(...args)) as { ok: boolean; value?: T; error?: { code: string; message: string } };
    if (!res.ok) {
      const code = res.error?.code === "unknown_profile" || res.error?.code === "invalid_arguments" ? "invalid_input" : (res.error?.code ?? "internal_error");
      throw new ShopError(code, res.error?.message ?? "bridge error");
    }
    return res.value as T;
  };
  return {
    version: raw.version,
    profiles: () => call("profiles"),
    paramsInfo: (p) => call("paramsInfo", p),
    genKeys: (profile) => call("genKeys", profile),
    validateKeys: (p, sk, pk, rlk) => call("validateKeys", p, sk, pk, rlk ?? ""),
    encryptScalar: (p, pk, x) => call("encryptScalar", p, pk, x),
    encryptVector: (p, pk, values) => call("encryptVector", p, pk, values),
    decryptScalar: (p, sk, ct) => call("decryptScalar", p, sk, ct),
    decryptVector: (p, sk, ct, n) => call("decryptVector", p, sk, ct, n),
    ciphertextInfo: (p, ct) => call("ciphertextInfo", p, ct),
  };
}

async function loadGoRuntime(g: WasmGlobal, origin: string): Promise<void> {
  if (g.Go) return;
  const url = `${origin}/wasm_exec.js`;
  if (typeof g.importScripts === "function") {
    try {
      g.importScripts(url); // classic worker
    } catch {
      // module worker: importScripts is not allowed, fall through to import()
    }
  }
  if (!g.Go) {
    await import(/* webpackIgnore: true */ /* turbopackIgnore: true */ url);
  }
  if (!g.Go) throw new ShopError("wasm_load_failed", "wasm_exec.js did not define Go");
}

/** Loads crypto.wasm once and returns the typed bridge. */
export function loadBridge(): Promise<Bridge> {
  if (loading) return loading;
  loading = (async () => {
    const g = globalThis as unknown as WasmGlobal;
    const origin = g.location?.origin ?? "";
    try {
      await loadGoRuntime(g, origin);
      const go = new g.Go!();
      let instance: WebAssembly.Instance;
      try {
        ({ instance } = await WebAssembly.instantiateStreaming(fetch(`${origin}/crypto.wasm`), go.importObject));
      } catch {
        const buf = await (await fetch(`${origin}/crypto.wasm`)).arrayBuffer();
        ({ instance } = await WebAssembly.instantiate(buf, go.importObject));
      }
      void go.run(instance);
      for (let i = 0; i < 200 && !g.sisterpedia; i++) {
        await new Promise((r) => setTimeout(r, 10));
      }
      const raw = g.sisterpedia;
      if (!raw) throw new ShopError("wasm_load_failed", "bridge did not register");
      if (raw.initError) throw new ShopError("wasm_load_failed", raw.initError);
      return wrap(raw);
    } catch (err) {
      loading = null;
      if (err instanceof ShopError) throw err;
      throw new ShopError("wasm_load_failed", err instanceof Error ? err.message : String(err));
    }
  })();
  return loading;
}
