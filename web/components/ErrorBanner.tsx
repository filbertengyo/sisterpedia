"use client";

import type { DisplayError } from "@/lib/errors";
import s from "./shop.module.css";

export function ErrorBanner({ error, onClose }: { error: (DisplayError & { action?: string }) | null; onClose: () => void }) {
  if (!error) return null;
  return (
    <div className={s.banner} role="alert" data-testid="error-banner" data-code={error.code}>
      <div>
        <b>{error.action ? `${error.action} gagal: ` : ""}</b>
        {error.message}
        <small>
          kode: {error.code}
          {error.detail ? ` — ${error.detail}` : ""}
        </small>
      </div>
      <button type="button" className={s.buttonGhost} onClick={onClose} aria-label="Tutup">
        ✕
      </button>
    </div>
  );
}
