"use client";

import { useState } from "react";
import { getShop } from "@/lib/api";
import { formatBytes } from "@/lib/format";
import type { CipherView } from "@/lib/types";
import s from "./shop.module.css";

/** Ciphertext preview: first 60 Base64 characters, binary size, level/scale and copy-full. Never shows keys. */
export function CipherPreview({ label, view, testId }: { label: string; view: CipherView; testId?: string }) {
  const [copied, setCopied] = useState<"idle" | "ok" | "fail">("idle");

  async function copy() {
    try {
      const full = await getShop().getCiphertext(view.ref);
      await navigator.clipboard.writeText(full);
      setCopied("ok");
    } catch {
      setCopied("fail");
    }
    setTimeout(() => setCopied("idle"), 1500);
  }

  return (
    <div className={s.pane} data-testid={testId}>
      <div className={s.paneLabel}>{label} (ciphertext, Base64)</div>
      <div className={s.cipher} data-testid={testId ? `${testId}-preview` : undefined}>
        {view.preview}
      </div>
      <div className={s.cipher} style={{ marginTop: 6 }} title="Potongan dari tengah ciphertext (koefisien polinomial)">
        {view.preview_body}
      </div>
      <div className={s.cipherMeta}>
        <span data-testid={testId ? `${testId}-bytes` : undefined}>{formatBytes(view.bytes)} biner</span>
        <span>{formatBytes(view.b64_length)} Base64</span>
        {view.level !== undefined && <span>level {view.level}</span>}
        {view.log2_scale !== undefined && <span>skala 2^{Number(view.log2_scale.toFixed(4))}</span>}
        <button type="button" className={s.buttonGhost} onClick={copy}>
          {copied === "ok" ? "Tersalin" : copied === "fail" ? "Gagal menyalin" : "Salin penuh"}
        </button>
      </div>
    </div>
  );
}
