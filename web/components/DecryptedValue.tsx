import { formatRaw, formatRupiah } from "@/lib/format";
import s from "./shop.module.css";

/** Decrypted value shown next to its ciphertext: rounded for display, raw CKKS value below. */
export function DecryptedValue({ label, value, raw, testId, money = true }: { label: string; value: number; raw: number; testId?: string; money?: boolean }) {
  const negative = value < 0;
  return (
    <div className={s.pane} data-testid={testId}>
      <div className={s.paneLabel}>{label} (hasil dekripsi di browser)</div>
      <div className={`${s.value} ${negative ? s.negative : ""}`} data-testid={testId ? `${testId}-value` : undefined}>
        {money ? formatRupiah(value) : value}
      </div>
      <div className={s.valueRaw} data-testid={testId ? `${testId}-raw` : undefined}>
        nilai mentah CKKS: {formatRaw(raw)}
      </div>
    </div>
  );
}
