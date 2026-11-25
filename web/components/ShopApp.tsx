"use client";

import { useCallback, useEffect, useState } from "react";
import { getShop, MOCK, SERVER_API, serverApi } from "@/lib/api";
import { describeError, type DisplayError } from "@/lib/errors";
import type { Catalog, LocalUser, Meta, RequestLog, ShopConfig } from "@/lib/types";
import { ActivateForm } from "./ActivateForm";
import { BalancePanel } from "./BalancePanel";
import { ErrorBanner } from "./ErrorBanner";
import { HistoryTable } from "./HistoryTable";
import { KeysPanel } from "./KeysPanel";
import { LogPanel, type LogEntry } from "./LogPanel";
import type { Runner } from "./runner";
import s from "./shop.module.css";
import { TopupForm } from "./TopupForm";
import { TransactForm } from "./TransactForm";
import { UserSwitcher } from "./UserSwitcher";

const TABS = [
  { id: "activate", label: "Activate" },
  { id: "balance", label: "CheckBalance" },
  { id: "topup", label: "Topup" },
  { id: "transact", label: "Transact" },
  { id: "history", label: "GetTransact" },
  { id: "keys", label: "Kunci" },
] as const;

type Tab = (typeof TABS)[number]["id"];

const ACTIVE_KEY = "sisterpedia.activeUser";
const EMPTY_CATALOG: Catalog = { products: [], discounts: [] };

export function ShopApp() {
  const [config, setConfig] = useState<ShopConfig | null>(null);
  const [catalog, setCatalog] = useState<Catalog>(EMPTY_CATALOG);
  const [users, setUsers] = useState<LocalUser[]>([]);
  const [active, setActive] = useState("");
  const [tab, setTab] = useState<Tab>("activate");
  const [error, setError] = useState<(DisplayError & { action?: string }) | null>(null);
  const [logs, setLogs] = useState<LogEntry[]>([]);
  const [online, setOnline] = useState<boolean | null>(null);
  const [initError, setInitError] = useState<DisplayError | null>(null);

  const addLogs = useCallback((action: string, user: string, requests: RequestLog[] | undefined) => {
    if (!requests?.length) return;
    const at = new Date().toLocaleTimeString("id-ID");
    setLogs((prev) => [...requests.map((r) => ({ ...r, at, action, user })), ...prev].slice(0, 200));
  }, []);

  const refreshUsers = useCallback(async (select?: string) => {
    const list = await getShop().listUsers();
    setUsers(list);
    setActive((cur) => {
      const want = select ?? cur;
      const next = list.some((u) => u.user_id === want) ? want : (list[0]?.user_id ?? "");
      if (next) localStorage.setItem(ACTIVE_KEY, next);
      else localStorage.removeItem(ACTIVE_KEY);
      return next;
    });
  }, []);

  const loadCatalog = useCallback(async () => {
    try {
      setCatalog(await serverApi.getCatalog());
      setOnline(true);
    } catch {
      setOnline(false);
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const shop = getShop();
        await shop.init();
        const cfg = await shop.getConfig();
        if (cancelled) return;
        setConfig(cfg);
        const saved = localStorage.getItem(ACTIVE_KEY) ?? undefined;
        await refreshUsers(saved);
      } catch (err) {
        if (!cancelled) setInitError(describeError(err));
      }
      await loadCatalog();
    })();
    const timer = setInterval(async () => setOnline(await serverApi.health()), 10000);
    return () => {
      cancelled = true;
      clearInterval(timer);
    };
  }, [refreshUsers, loadCatalog]);

  const run: Runner = useCallback(
    async (action, f) => {
      setError(null);
      try {
        const r = await f();
        const meta = (r as { meta?: Meta } | undefined)?.meta;
        addLogs(action, active, meta?.requests);
        setOnline(true);
        return r;
      } catch (err) {
        const log = (err as { log?: RequestLog }).log;
        addLogs(action, active, log ? [log] : undefined);
        const d = describeError(err);
        if (d.code === "server_unreachable") setOnline(false);
        setError({ ...d, action });
        return undefined;
      }
    },
    [active, addLogs],
  );

  function selectUser(id: string) {
    setActive(id);
    localStorage.setItem(ACTIVE_KEY, id);
    setError(null);
  }

  const needsUser = tab !== "activate" && tab !== "keys";

  return (
    <div className={s.app}>
      <header className={s.header}>
        <div className={s.title}>
          <h1>Sisterpedia — Toko Daring FHE (CKKS)</h1>
          <span>Saldo, nominal, kuantitas, dan tagihan selalu terenkripsi; dekripsi hanya di browser Anda.</span>
        </div>
        <div className={s.badges}>
          <span className={`${s.badge} ${online === false ? s.badgeBad : online ? s.badgeOk : ""}`} data-testid="server-status">
            server {MOCK ? "mock" : SERVER_API}: {online === null ? "…" : online ? "online" : "offline"}
          </span>
          <span className={`${s.badge} ${config ? s.badgeOk : initError ? s.badgeBad : ""}`} data-testid="wasm-status">
            {config ? `kripto WASM siap (${config.bridge_version.split(" ")[0]})` : initError ? "kripto WASM gagal" : "memuat kripto WASM…"}
          </span>
          {active && (
            <span className={s.badge} data-testid="active-user">
              pengguna aktif: <b>{active}</b>
            </span>
          )}
        </div>
      </header>

      <div className={s.bannerArea}>
        {initError && <ErrorBanner error={{ ...initError, action: "Inisialisasi" }} onClose={() => setInitError(null)} />}
        {online === false && !error && (
          <ErrorBanner error={{ code: "server_unreachable", message: "Server tidak dapat dihubungi. Pastikan server berjalan.", action: "Koneksi" }} onClose={() => setOnline(null)} />
        )}
        <ErrorBanner error={error} onClose={() => setError(null)} />
      </div>

      <aside className={s.side}>
        <UserSwitcher users={users} active={active} onSelect={selectUser} />
        {config && (
          <section>
            <h3 className={s.sectionTitle}>Konfigurasi</h3>
            <dl className={s.kv}>
              <dt>Pustaka</dt>
              <dd>Lattigo v6 (WASM)</dd>
              <dt>Profil</dt>
              <dd>{config.profiles.map((p) => p.name).join(", ")}</dd>
              <dt>Batas nilai</dt>
              <dd>±{config.max_value.toLocaleString("id-ID")}</dd>
            </dl>
          </section>
        )}
      </aside>

      <main className={s.main}>
        <nav className={s.tabs} role="tablist">
          {TABS.map((t) => (
            <button key={t.id} role="tab" aria-selected={tab === t.id} className={`${s.tab} ${tab === t.id ? s.tabActive : ""}`} onClick={() => setTab(t.id)} data-testid={`tab-${t.id}`}>
              {t.label}
            </button>
          ))}
        </nav>

        {!config && !initError && <p className={s.hint}>Memuat modul kriptografi...</p>}

        {config && (
          <>
            <div hidden={tab !== "activate"}>
              <ActivateForm profiles={config.profiles} defaultProfile={config.default_profile} run={run} onActivated={(id) => void refreshUsers(id)} />
            </div>
            {needsUser && !active && <div className={s.notice}>Pilih atau aktifkan pengguna terlebih dahulu.</div>}
            {active && (
              <>
                <div hidden={tab !== "balance"}>
                  <BalancePanel key={`b-${active}`} userId={active} run={run} />
                </div>
                <div hidden={tab !== "topup"}>
                  <TopupForm key={`t-${active}`} userId={active} maxValue={config.max_value} run={run} />
                </div>
                <div hidden={tab !== "transact"}>
                  {catalog.products.length > 0 ? (
                    <TransactForm key={`x-${active}`} userId={active} catalog={catalog} run={run} />
                  ) : (
                    <div className={s.notice}>Katalog belum dimuat dari server.</div>
                  )}
                </div>
                <div hidden={tab !== "history"}>
                  <HistoryTable key={`h-${active}`} userId={active} catalog={catalog} run={run} />
                </div>
              </>
            )}
            <div hidden={tab !== "keys"}>
              <KeysPanel userId={active} run={run} onChanged={(id) => void refreshUsers(id)} />
            </div>
          </>
        )}

        <LogPanel entries={logs} />
      </main>
    </div>
  );
}
