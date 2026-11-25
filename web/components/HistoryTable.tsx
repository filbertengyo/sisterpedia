"use client";

import { useState } from "react";
import { getShop } from "@/lib/api";
import { formatRaw, formatRupiah, formatTime } from "@/lib/format";
import type { Catalog, HistoryResult } from "@/lib/types";
import { CipherPreview } from "./CipherPreview";
import type { Runner } from "./runner";
import s from "./shop.module.css";

export function HistoryTable({ userId, catalog, run }: { userId: string; catalog: Catalog; run: Runner }) {
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<HistoryResult | null>(null);
  const [open, setOpen] = useState<string | null>(null);
  const [loads, setLoads] = useState(0);

  async function load() {
    if (busy) return;
    setBusy(true);
    const r = await run("GetTransact", () => getShop().getHistory(userId));
    setBusy(false);
    if (r) setResult(r);
    setLoads((n) => n + 1);
  }

  const name = (id: string) => catalog.products.find((p) => p.product_id === id)?.name ?? id;

  return (
    <div className={s.card} data-testid="panel-history" data-loads={loads}>
      <h2>GetTransact — riwayat transaksi</h2>
      <p className={s.hint}>Server mengirim ciphertext metadata [ID produk, kuantitas, ID diskon] dan total; browser mendekripsi dan membulatkan metadata ke bilangan bulat terdekat.</p>
      <div className={s.row}>
        <button className={s.button} onClick={load} disabled={busy} data-testid="history-load">
          {busy ? "Memuat..." : `Muat riwayat ${userId}`}
        </button>
      </div>
      {result && result.user_id === userId && (
        <div className={s.tableWrap}>
          {result.items.length === 0 ? (
            <p className={s.hint} data-testid="history-empty">
              Belum ada transaksi.
            </p>
          ) : (
            <table className={s.table} data-testid="history-table">
              <thead>
                <tr>
                  <th>Waktu</th>
                  <th>ID transaksi</th>
                  <th>Produk</th>
                  <th>Qty</th>
                  <th>Diskon</th>
                  <th>Total</th>
                  <th>Ciphertext</th>
                </tr>
              </thead>
              <tbody>
                {result.items.map((it) => (
                  <tr key={it.transaction_id} data-testid="history-row">
                    <td>{formatTime(it.created_at)}</td>
                    <td className={s.mono}>{it.transaction_id}</td>
                    <td>
                      {it.product_id} <span className={s.small}>{name(it.product_id)}</span>
                    </td>
                    <td>{it.quantity}</td>
                    <td>{it.discount_code}</td>
                    <td>
                      <div className={it.total < 0 ? s.negative : undefined}>{formatRupiah(it.total)}</div>
                      <div className={s.valueRaw}>{formatRaw(it.total_raw)}</div>
                      {it.warning && (
                        <div className={s.fieldError} title="Nilai metadata hasil dekripsi lebih dari 0,25 dari bilangan bulat terdekat">
                          ⚠ metadata tidak pasti
                        </div>
                      )}
                    </td>
                    <td>
                      <button type="button" className={s.buttonGhost} onClick={() => setOpen(open === it.transaction_id ? null : it.transaction_id)}>
                        {open === it.transaction_id ? "Tutup" : "Lihat"}
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
          {result.items
            .filter((it) => it.transaction_id === open)
            .map((it) => (
              <div key={it.transaction_id} className={s.sideBySide} style={{ marginTop: 12 }}>
                <CipherPreview label="Total" view={it.enc_total} />
                <div>
                  <CipherPreview label="Metadata" view={it.enc_metadata} />
                  <p className={s.valueRaw}>dekripsi mentah: [{it.metadata_raw.map((v) => formatRaw(v)).join(", ")}]</p>
                </div>
              </div>
            ))}
        </div>
      )}
    </div>
  );
}
