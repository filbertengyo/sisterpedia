"use client";

import type { LocalUser } from "@/lib/types";
import s from "./shop.module.css";

/** Lists users whose keys are stored in this browser; switching does not reload the page. */
export function UserSwitcher({ users, active, onSelect }: { users: LocalUser[]; active: string; onSelect: (id: string) => void }) {
  return (
    <section>
      <h3 className={s.sectionTitle}>Pengguna di perangkat ini</h3>
      {users.length === 0 ? (
        <p className={s.hint}>Belum ada kunci tersimpan. Aktifkan akun baru atau impor kunci.</p>
      ) : (
        <ul className={s.userList} data-testid="user-list">
          {users.map((u) => (
            <li key={u.user_id}>
              <button
                type="button"
                className={`${s.userItem} ${u.user_id === active ? s.userActive : ""}`}
                onClick={() => onSelect(u.user_id)}
                data-testid={`user-${u.user_id}`}
              >
                <span>{u.user_id}</span>
                <span className={s.small}>{u.profile}</span>
              </button>
            </li>
          ))}
        </ul>
      )}
      <p className={s.hint} style={{ marginTop: 10 }}>
        Secret key setiap pengguna disimpan di IndexedDB browser ini dan tidak pernah dikirim ke server.
      </p>
    </section>
  );
}
