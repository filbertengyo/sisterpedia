"use client";

import { useState } from "react";
import { getShop } from "@/lib/api";
import { formatBytes } from "@/lib/format";
import type { ActivateResult, ProfileInfo } from "@/lib/types";
import { validateUserId } from "@/lib/validate";
import { CipherPreview } from "./CipherPreview";
import { DecryptedValue } from "./DecryptedValue";
import type { Runner } from "./runner";
import s from "./shop.module.css";

export function ActivateForm({ profiles, defaultProfile, run, onActivated }: { profiles: ProfileInfo[]; defaultProfile: string; run: Runner; onActivated: (userId: string) => void }) {
  const [userId, setUserId] = useState("");
  const [profile, setProfile] = useState(defaultProfile);
  const [touched, setTouched] = useState(false);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<ActivateResult | null>(null);
  const v = validateUserId(userId);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setTouched(true);
    if (!v.ok || busy) return;
    setBusy(true);
    const r = await run("Activate", () => getShop().activate(userId, profile));
    setBusy(false);
    if (r) {
      setResult(r);
      onActivated(userId);
    }
  }

  const info = profiles.find((p) => p.name === profile);

  return (
    <div className={s.card} data-testid="panel-activate">
      <h2>Activate — buat akun dan kunci CKKS</h2>
      <p className={s.hint}>
        Browser membangkitkan secret key, public key, dan relinearization key, mengenkripsi saldo awal 0, lalu mengirim parameter publik, pk, rlk, dan Enc(0) ke server.
      </p>
      <form className={s.form} onSubmit={submit}>
        <label className={s.field}>
          ID pengguna
          <input value={userId} onChange={(e) => setUserId(e.target.value.trim())} onBlur={() => setTouched(true)} placeholder="mis. vanessa" data-testid="activate-user" />
          {touched && !v.ok && <span className={s.fieldError}>{v.message}</span>}
        </label>
        <label className={s.field}>
          Profil parameter
          <select value={profile} onChange={(e) => setProfile(e.target.value)} data-testid="activate-profile">
            {profiles.map((p) => (
              <option key={p.name} value={p.name}>
                {p.name} (N=2^{p.logN}, level {p.maxLevel})
              </option>
            ))}
          </select>
        </label>
        <button className={s.button} disabled={busy} data-testid="activate-submit">
          {busy ? "Membangkitkan kunci..." : "Aktifkan akun"}
        </button>
      </form>
      {info && (
        <p className={s.small}>
          {info.name}: LogN {info.logN}, {info.slots} slot, skala 2^{info.logDefaultScale}, log2(QP) ≈ {info.logQP.toFixed(0)}, batas nilai aman ±{info.maxValue.toLocaleString("id-ID")}
        </p>
      )}
      {result && (
        <>
          <div className={s.success} data-testid="activate-status">
            Status: {result.status} — akun <b>{result.user_id}</b> ({result.profile})
          </div>
          <div className={s.sideBySide}>
            <CipherPreview label="Saldo awal" view={result.enc_balance} testId="activate-ct" />
            <DecryptedValue label="Saldo awal" value={result.balance} raw={result.balance_raw} testId="activate-balance" />
          </div>
          <dl className={s.kv}>
            <dt>Parameter (params.json)</dt>
            <dd>{formatBytes(result.artifact_bytes.params)}</dd>
            <dt>Public key (dikirim)</dt>
            <dd>{formatBytes(result.artifact_bytes.public_key)}</dd>
            <dt>Relinearization key (dikirim)</dt>
            <dd>{formatBytes(result.artifact_bytes.relin_key)}</dd>
            <dt>Secret key (hanya lokal)</dt>
            <dd>{formatBytes(result.artifact_bytes.secret_key_local)}</dd>
          </dl>
        </>
      )}
    </div>
  );
}
