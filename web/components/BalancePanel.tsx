"use client";

import { useState } from "react";
import { getShop } from "@/lib/api";
import type { BalanceView } from "@/lib/types";
import { BalanceCard } from "./BalanceCard";
import type { Runner } from "./runner";
import s from "./shop.module.css";

export function BalancePanel({ userId, run }: { userId: string; run: Runner }) {
  const [busy, setBusy] = useState(false);
  const [view, setView] = useState<BalanceView | null>(null);
  const [checks, setChecks] = useState(0);

  async function check() {
    if (busy) return;
    setBusy(true);
    const r = await run("CheckBalance", () => getShop().getBalance(userId));
    setBusy(false);
    if (r) setView(r);
    setChecks((n) => n + 1);
  }

  return (
    <div className={s.card} data-testid="panel-balance" data-checks={checks}>
      <h2>CheckBalance — saldo terenkripsi</h2>
      <p className={s.hint}>Server hanya mengembalikan ciphertext saldo; dekripsi dilakukan di browser dengan secret key lokal.</p>
      <div className={s.row}>
        <button className={s.button} onClick={check} disabled={busy} data-testid="balance-check">
          {busy ? "Memeriksa..." : `Periksa saldo ${userId}`}
        </button>
      </div>
      {view && view.user_id === userId && <BalanceCard view={view} />}
    </div>
  );
}
