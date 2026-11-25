"use client";

import { useRef, useState } from "react";
import { getShop } from "@/lib/api";
import { validateUserId } from "@/lib/validate";
import type { Runner } from "./runner";
import s from "./shop.module.css";

/** Export/import of params, secret key, public key and relinearization key (zip). */
export function KeysPanel({ userId, run, onChanged }: { userId: string; run: Runner; onChanged: (activeUser?: string) => void }) {
  const [confirmExport, setConfirmExport] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [importId, setImportId] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [message, setMessage] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const fileRef = useRef<HTMLInputElement>(null);
  const vImport = validateUserId(importId);

  async function doExport() {
    setConfirmExport(false);
    const blob = await run("Ekspor kunci", () => getShop().exportKeys(userId));
    if (!blob) return;
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = `sisterpedia-${userId}-keys.zip`;
    document.body.appendChild(a);
    a.click();
    a.remove();
    setTimeout(() => URL.revokeObjectURL(url), 1000);
    setMessage(`Berkas sisterpedia-${userId}-keys.zip diunduh. Simpan di tempat aman: berisi secret key.`);
  }

  async function doImport(e: React.FormEvent) {
    e.preventDefault();
    if (!file || !vImport.ok || busy) return;
    setBusy(true);
    const r = await run("Impor kunci", () => getShop().importKeys(importId, file));
    setBusy(false);
    if (r) {
      setMessage(`Kunci ${r.user_id} (${r.profile}) berhasil diimpor tanpa pembangkitan kunci baru.`);
      setFile(null);
      if (fileRef.current) fileRef.current.value = "";
      onChanged(r.user_id);
    }
  }

  async function doDelete() {
    setConfirmDelete(false);
    const r = await run("Hapus kunci lokal", async () => {
      await getShop().deleteLocalKeys(userId);
      return true;
    });
    if (r) {
      setMessage(`Kunci lokal ${userId} dihapus dari browser ini. Akun di server tetap ada; impor berkas kunci untuk memakainya lagi.`);
      onChanged();
    }
  }

  return (
    <div className={s.card} data-testid="panel-keys">
      <h2>Kunci — ekspor dan impor</h2>
      <p className={s.hint}>
        Berkas zip berisi <code>params.json</code>, <code>secret.key</code>, <code>public.key</code>, dan <code>relin.key</code> (format serialisasi biner Lattigo v6). Dengan berkas ini akun yang sama dapat dibuka kembali tanpa membangkitkan kunci baru.
      </p>
      {message && (
        <div className={s.success} data-testid="keys-message">
          {message}
        </div>
      )}
      <div className={s.row}>
        <button className={s.button} onClick={() => setConfirmExport(true)} disabled={!userId} data-testid="keys-export">
          Ekspor kunci {userId}
        </button>
        <button className={`${s.buttonGhost} ${s.buttonDanger}`} onClick={() => setConfirmDelete(true)} disabled={!userId} data-testid="keys-delete">
          Hapus kunci lokal {userId}
        </button>
      </div>

      <form className={s.form} onSubmit={doImport}>
        <label className={s.field}>
          ID pengguna untuk impor
          <input value={importId} onChange={(e) => setImportId(e.target.value.trim())} placeholder="mis. vanessa" data-testid="import-user" />
          {importId !== "" && !vImport.ok && <span className={s.fieldError}>{vImport.message}</span>}
        </label>
        <label className={s.field}>
          Berkas kunci (.zip)
          <input ref={fileRef} type="file" accept=".zip,application/zip" onChange={(e) => setFile(e.target.files?.[0] ?? null)} data-testid="import-file" />
        </label>
        <button className={s.button} disabled={!file || !vImport.ok || busy} data-testid="import-submit">
          {busy ? "Memvalidasi..." : "Impor"}
        </button>
      </form>

      {confirmExport && (
        <div className={s.modalBackdrop} role="dialog" aria-modal="true" data-testid="export-modal">
          <div className={s.modal}>
            <h2>Peringatan: berkas berisi secret key</h2>
            <p>
              Berkas ekspor memuat <b>secret key</b> akun <b>{userId}</b>. Siapa pun yang memilikinya dapat mendekripsi saldo dan riwayat Anda. Jangan bagikan, unggah, atau kirim berkas ini ke server mana pun.
            </p>
            <div className={s.row}>
              <button className={s.button} onClick={doExport} data-testid="export-confirm">
                Saya mengerti, unduh
              </button>
              <button className={s.buttonGhost} onClick={() => setConfirmExport(false)}>
                Batal
              </button>
            </div>
          </div>
        </div>
      )}
      {confirmDelete && (
        <div className={s.modalBackdrop} role="dialog" aria-modal="true" data-testid="delete-modal">
          <div className={s.modal}>
            <h2>Hapus kunci lokal {userId}?</h2>
            <p>Tanpa berkas ekspor, saldo dan riwayat akun ini tidak dapat didekripsi lagi. Pastikan Anda sudah mengekspor kunci.</p>
            <div className={s.row}>
              <button className={`${s.button}`} onClick={doDelete} data-testid="delete-confirm">
                Hapus
              </button>
              <button className={s.buttonGhost} onClick={() => setConfirmDelete(false)}>
                Batal
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
