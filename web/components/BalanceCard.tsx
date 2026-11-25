"use client";

import type { BalanceView } from "@/lib/types";
import { CipherPreview } from "./CipherPreview";
import { DecryptedValue } from "./DecryptedValue";
import s from "./shop.module.css";

/** Encrypted balance side by side with its decrypted value; negative balances are flagged. */
export function BalanceCard({ view, label = "Saldo", testId = "balance" }: { view: BalanceView; label?: string; testId?: string }) {
  return (
    <>
      <div className={s.sideBySide}>
        <CipherPreview label={label} view={view.enc_balance} testId={`${testId}-ct`} />
        <DecryptedValue label={label} value={view.balance} raw={view.balance_raw} testId={testId} />
      </div>
      {view.negative && (
        <div className={s.warning} data-testid={`${testId}-negative`}>
          Saldo tidak mencukupi: saldo hasil dekripsi negatif. Server tetap mengurangi saldo karena tidak dapat membaca nilainya.
        </div>
      )}
    </>
  );
}
