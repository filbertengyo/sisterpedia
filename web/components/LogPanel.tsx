"use client";

import { formatBytes } from "@/lib/format";
import type { RequestLog } from "@/lib/types";
import s from "./shop.module.css";

export interface LogEntry extends RequestLog {
  at: string;
  action: string;
  user: string;
}

/** Request log: method, path, status, latency and sizes only (never values or keys). */
export function LogPanel({ entries }: { entries: LogEntry[] }) {
  return (
    <div className={s.card} data-testid="panel-log">
      <h2>Log permintaan ke server</h2>
      <p className={s.hint}>Hanya metadata permintaan yang dicatat: tidak ada nilai saldo, nominal, atau kunci. Badan permintaan berisi ciphertext Base64.</p>
      {entries.length === 0 ? (
        <p className={s.hint}>Belum ada permintaan.</p>
      ) : (
        <div className={s.log}>
          {entries.map((e, i) => (
            <div key={i} className={s.logRow}>
              <span>{e.at}</span>
              <span>{e.method}</span>
              <span title={e.action}>{e.path}</span>
              <span className={e.status < 400 ? s.status2 : s.status4}>{e.status}</span>
              <span>{e.ms} ms</span>
              <span>
                ↑{formatBytes(e.request_bytes)} ↓{formatBytes(e.response_bytes)}
                {e.server_eval_ms !== undefined ? ` · eval ${e.server_eval_ms} ms` : ""}
              </span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
