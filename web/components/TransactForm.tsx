"use client";

import { useState } from "react";
import { getShop } from "@/lib/api";
import { formatRupiah } from "@/lib/format";
import type { Catalog, TransactResult } from "@/lib/types";
import { estimateBill, validateQuantity } from "@/lib/validate";
import { BalanceCard } from "./BalanceCard";
import { CipherPreview } from "./CipherPreview";
import { DecryptedValue } from "./DecryptedValue";
import type { Runner } from "./runner";
import s from "./shop.module.css";

function rule(percent: number, fixed: number): string {
  const parts: string[] = [];
  if (percent > 0) parts.push(`${Math.round(percent * 100)}%`);
  if (fixed > 0) parts.push(`potongan ${formatRupiah(fixed)}`);
  return parts.length ? parts.join(" + ") : "tanpa diskon";
}

export function TransactForm({ userId, catalog, run }: { userId: string; catalog: Catalog; run: Runner }) {
  const [productId, setProductId] = useState(catalog.products[0]?.product_id ?? "");
  const [qtyText, setQtyText] = useState("1");
  const [discount, setDiscount] = useState("NONE");
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<TransactResult | null>(null);

  const quantity = Number(qtyText);
  const vq = validateQuantity(quantity);
  const product = catalog.products.find((p) => p.product_id === productId);
  const disc = catalog.discounts.find((d) => d.code === discount);
  const estimate = product && disc && vq.ok ? estimateBill(product.base_price, quantity, disc.percent, disc.fixed) : null;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!vq.ok || busy) return;
    setBusy(true);
    const r = await run("Transact", () => getShop().transact(userId, productId, quantity, discount));
    setBusy(false);
    if (r) setResult(r);
  }

  return (
    <div className={s.card} data-testid="panel-transact">
      <h2>Transact — tagihan dihitung pada ciphertext</h2>
      <p className={s.hint}>
        Kuantitas dan metadata dikirim terenkripsi; ID produk dan kode diskon terbuka agar server dapat mencari harga. Server menghitung t = (p × q × (1 − d)) − k dengan dua perkalian ciphertext × ciphertext, lalu saldo ⊖ t.
      </p>
      <form className={s.form} onSubmit={submit}>
        <label className={s.field}>
          Produk
          <select value={productId} onChange={(e) => setProductId(e.target.value)} data-testid="transact-product">
            {catalog.products.map((p) => (
              <option key={p.product_id} value={p.product_id}>
                {p.product_id} — {p.name} ({formatRupiah(p.base_price)})
              </option>
            ))}
          </select>
        </label>
        <label className={s.field}>
          Kuantitas
          <input value={qtyText} onChange={(e) => setQtyText(e.target.value)} inputMode="numeric" data-testid="transact-qty" style={{ minWidth: 90 }} />
          {!vq.ok && <span className={s.fieldError}>{vq.message}</span>}
        </label>
        <label className={s.field}>
          Kode diskon
          <select value={discount} onChange={(e) => setDiscount(e.target.value)} data-testid="transact-discount">
            {catalog.discounts.map((d) => (
              <option key={d.code} value={d.code}>
                {d.code} — {rule(d.percent, d.fixed)}
              </option>
            ))}
          </select>
        </label>
        <button className={s.button} disabled={busy || !vq.ok} data-testid="transact-submit">
          {busy ? "Memproses..." : "Beli"}
        </button>
      </form>
      {estimate !== null && <p className={s.small}>Perkiraan lokal (plaintext, tidak dikirim): {formatRupiah(estimate)}</p>}
      {result && (
        <>
          <div className={s.success} data-testid="transact-status">
            Transaksi <span className={s.mono}>{result.transaction_id}</span>: {result.product_id} × {result.quantity}, diskon {result.discount_code}
          </div>
          <div className={s.sideBySide}>
            <CipherPreview label="Total tagihan" view={result.enc_total} testId="transact-total-ct" />
            <DecryptedValue label="Total tagihan" value={result.total} raw={result.total_raw} testId="transact-total" />
          </div>
          <BalanceCard view={result} label="Saldo baru" testId="transact-balance" />
          <div className={s.sideBySide}>
            <CipherPreview label="Kuantitas terkirim" view={result.enc_quantity} />
            <CipherPreview label="Metadata terkirim [produk, qty, diskon]" view={result.enc_metadata} />
          </div>
        </>
      )}
    </div>
  );
}
