"use client";

import { useState } from "react";
import { getShop } from "@/lib/api";
import { formatRupiah } from "@/lib/format";
import type { BalanceView } from "@/lib/types";
import { parseAmount, validateAmount } from "@/lib/validate";
import { BalanceCard } from "./BalanceCard";
import type { Runner } from "./runner";
import s from "./shop.module.css";

export function TopupForm({ userId, maxValue, run }: { userId: string; maxValue: number; run: Runner }) {
  const [text, setText] = useState("");
  const [touched, setTouched] = useState(false);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<{ amount: number; view: BalanceView } | null>(null);
  const amount = parseAmount(text);
  const v = validateAmount(amount, maxValue);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setTouched(true);
    if (!v.ok || busy) return; // anti double-submit
    setBusy(true);
    const r = await run("Topup", () => getShop().topup(userId, amount));
    setBusy(false);
    if (r) {
      setResult({ amount, view: r });
      setText("");
      setTouched(false);
    }
  }

  return (
    <div className={s.card} data-testid="panel-topup">
      <h2>Topup — tambah saldo secara homomorfik</h2>
      <p className={s.hint}>Nominal dienkripsi di browser; server menghitung c_saldo ⊕ c_nominal tanpa melihat nilainya.</p>
      <form className={s.form} onSubmit={submit}>
        <label className={s.field}>
          Nominal (bulat atau desimal, maks. 2 desimal)
          <input value={text} onChange={(e) => setText(e.target.value)} onBlur={() => setTouched(true)} inputMode="decimal" placeholder="mis. 100000 atau 2500.75" data-testid="topup-amount" />
          {touched && text !== "" && !v.ok && <span className={s.fieldError}>{v.message}</span>}
        </label>
        <button className={s.button} disabled={busy || !v.ok} data-testid="topup-submit">
          {busy ? "Mengirim..." : "Top up"}
        </button>
      </form>
      {result && (
        <>
          <div className={s.success}>Top up {formatRupiah(result.amount)} berhasil. Saldo baru:</div>
          <BalanceCard view={result.view} label="Saldo baru" testId="topup-balance" />
        </>
      )}
    </div>
  );
}
