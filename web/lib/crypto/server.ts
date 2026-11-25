// REST client used by the shop module (runs in the worker). Bodies carry
// only Base64 ciphertexts and public artifacts; the log records sizes and
// timings, never values.
import { ShopError, type RequestLog } from "../types";

export interface ServerResponse<T> {
  data: T;
  log: RequestLog;
}

function evalMs(header: string | null): number | undefined {
  if (!header) return undefined;
  const m = /eval;dur=([\d.]+)/.exec(header);
  return m ? Number(m[1]) : undefined;
}

export class ServerApi {
  constructor(private readonly base: string) {}

  async request<T>(method: "GET" | "POST", path: string, body?: unknown): Promise<ServerResponse<T>> {
    const payload = body === undefined ? undefined : JSON.stringify(body);
    const start = performance.now();
    let resp: Response;
    try {
      resp = await fetch(this.base + path, {
        method,
        headers: payload ? { "Content-Type": "application/json" } : undefined,
        body: payload,
        cache: "no-store",
      });
    } catch (err) {
      throw new ShopError("server_unreachable", `${method} ${path}: ${err instanceof Error ? err.message : String(err)}`);
    }
    const text = await resp.text();
    const log: RequestLog = {
      method,
      path,
      status: resp.status,
      ms: Math.round((performance.now() - start) * 10) / 10,
      request_bytes: payload?.length ?? 0,
      response_bytes: text.length,
      server_eval_ms: evalMs(resp.headers.get("Server-Timing")),
    };
    if (!resp.ok) {
      let code = "internal_error";
      let detail = `HTTP ${resp.status}`;
      try {
        const e = JSON.parse(text) as { error?: string; detail?: string };
        if (e.error) code = e.error;
        if (e.detail) detail = e.detail;
      } catch {
        // non-JSON error body
      }
      const err = new ShopError(code, detail, resp.status);
      (err as ShopError & { log?: RequestLog }).log = log;
      throw err;
    }
    return { data: JSON.parse(text) as T, log };
  }
}
