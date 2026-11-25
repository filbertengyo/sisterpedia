// IndexedDB keystore. Only the crypto worker reads or writes it: the secret
// key never reaches the main thread (except inside the export zip Blob) and
// never leaves the browser.
import type { LocalUser } from "../types";

const DB_NAME = "sisterpedia-keys";
const STORE = "users";
const VERSION = 1;

export interface KeyRecord {
  user_id: string;
  profile: string;
  params: string; // Base64 of Lattigo params JSON
  secret_key: string; // Base64 - never sent anywhere
  public_key: string;
  relin_key: string;
  created_at: string;
}

let dbPromise: Promise<IDBDatabase> | null = null;

function open(): Promise<IDBDatabase> {
  if (dbPromise) return dbPromise;
  dbPromise = new Promise((resolve, reject) => {
    const req = indexedDB.open(DB_NAME, VERSION);
    req.onupgradeneeded = () => {
      const db = req.result;
      if (!db.objectStoreNames.contains(STORE)) db.createObjectStore(STORE, { keyPath: "user_id" });
    };
    req.onsuccess = () => resolve(req.result);
    req.onerror = () => {
      dbPromise = null;
      reject(req.error);
    };
  });
  return dbPromise;
}

function run<T>(mode: IDBTransactionMode, f: (s: IDBObjectStore) => IDBRequest<T>): Promise<T> {
  return open().then(
    (db) =>
      new Promise<T>((resolve, reject) => {
        const tx = db.transaction(STORE, mode);
        const req = f(tx.objectStore(STORE));
        let result: T;
        req.onsuccess = () => {
          result = req.result;
        };
        tx.oncomplete = () => resolve(result);
        tx.onerror = () => reject(tx.error ?? req.error);
        tx.onabort = () => reject(tx.error ?? new Error("transaction aborted"));
      }),
  );
}

export async function getKeys(userId: string): Promise<KeyRecord | undefined> {
  return run<KeyRecord | undefined>("readonly", (s) => s.get(userId) as IDBRequest<KeyRecord | undefined>);
}

export async function putKeys(rec: KeyRecord): Promise<void> {
  await run("readwrite", (s) => s.put(rec));
}

export async function deleteKeys(userId: string): Promise<void> {
  await run("readwrite", (s) => s.delete(userId));
}

/** Lists local users without any key material. */
export async function listUsers(): Promise<LocalUser[]> {
  const all = await run<KeyRecord[]>("readonly", (s) => s.getAll() as IDBRequest<KeyRecord[]>);
  return all
    .map((r) => ({ user_id: r.user_id, profile: r.profile, created_at: r.created_at }))
    .sort((a, b) => a.created_at.localeCompare(b.created_at));
}
