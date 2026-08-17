import { VAULT_LOCK } from "./constants";
import { keys as idbKeys, get as idbGet, del as idbDel } from "idb-keyval";

export interface VaultRecord {
  id: "vault";
  mode: "prf";
  credentialId?: string;
  salt: string;
  encryptedToken: string;
}

export interface OfflineQueueItem {
  id: string;
  request?: unknown;
  token?: string;
  cipher?: string;
  timestamp: number;
}

export function openDB(): Promise<IDBDatabase> {
  return new Promise((resolve, reject) => {
    const request = indexedDB.open(VAULT_LOCK.DB_NAME, VAULT_LOCK.DB_VERSION);
    request.onupgradeneeded = () => {
      const db = request.result;
      if (!db.objectStoreNames.contains(VAULT_LOCK.STORE_OFFLINE)) {
        db.createObjectStore(VAULT_LOCK.STORE_OFFLINE, { keyPath: "id" });
      }
      if (!db.objectStoreNames.contains(VAULT_LOCK.STORE_VAULT)) {
        db.createObjectStore(VAULT_LOCK.STORE_VAULT, { keyPath: "id" });
      }
    };
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

function promisify<T>(request: IDBRequest<T>): Promise<T> {
  return new Promise((resolve, reject) => {
    request.onsuccess = () => resolve(request.result);
    request.onerror = () => reject(request.error);
  });
}

function txDone(tx: IDBTransaction): Promise<void> {
  return new Promise((resolve, reject) => {
    tx.oncomplete = () => resolve();
    tx.onerror = () => reject(tx.error);
    tx.onabort = () => reject(tx.error);
  });
}

// One-time migration from the legacy idb-keyval store

let migrated = false;

export async function migrateOfflineQueue(): Promise<void> {
  if (migrated) return;
  migrated = true;
  try {
    const allKeys = await idbKeys();
    const offlineKeys = allKeys.filter((k) =>
      String(k).startsWith("khayal-offline-"),
    );
    if (offlineKeys.length === 0) return;

    const db = await openDB();
    for (const key of offlineKeys) {
      const item = await idbGet<OfflineQueueItem>(key);
      if (!item || typeof item !== "object" || !item.id) continue;
      await promisify(
        db
          .transaction(VAULT_LOCK.STORE_OFFLINE, "readwrite")
          .objectStore(VAULT_LOCK.STORE_OFFLINE)
          .put(item),
      );
      await idbDel(key);
    }
  } catch {
    // Best-effort only — never block startup on a migration failure
  }
}

// Vault record

export async function getVaultRecord(): Promise<VaultRecord | null> {
  const db = await openDB();
  const tx = db.transaction(VAULT_LOCK.STORE_VAULT, "readonly");
  const record = await promisify<VaultRecord | undefined>(
    tx.objectStore(VAULT_LOCK.STORE_VAULT).get("vault"),
  );
  return record || null;
}

export async function saveVaultRecord(record: VaultRecord): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(VAULT_LOCK.STORE_VAULT, "readwrite");
  tx.objectStore(VAULT_LOCK.STORE_VAULT).put(record);
  await txDone(tx);
}

export async function deleteVaultRecord(): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(VAULT_LOCK.STORE_VAULT, "readwrite");
  tx.objectStore(VAULT_LOCK.STORE_VAULT).delete("vault");
  await txDone(tx);
}

// Offline queue (raw storage)

export async function getAllOfflineItems(): Promise<OfflineQueueItem[]> {
  const db = await openDB();
  const tx = db.transaction(VAULT_LOCK.STORE_OFFLINE, "readonly");
  const items = await promisify<OfflineQueueItem[]>(
    tx.objectStore(VAULT_LOCK.STORE_OFFLINE).getAll(),
  );
  return items || [];
}

export async function putOfflineItem(item: OfflineQueueItem): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(VAULT_LOCK.STORE_OFFLINE, "readwrite");
  tx.objectStore(VAULT_LOCK.STORE_OFFLINE).put(item);
  await txDone(tx);
}

export async function deleteOfflineItem(id: string): Promise<void> {
  const db = await openDB();
  const tx = db.transaction(VAULT_LOCK.STORE_OFFLINE, "readwrite");
  tx.objectStore(VAULT_LOCK.STORE_OFFLINE).delete(id);
  await txDone(tx);
}
