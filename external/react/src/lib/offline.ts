import { KhayalClient } from "./api";
import type { CaptureRequest } from "./api";
import {
  getAllOfflineItems,
  putOfflineItem,
  deleteOfflineItem,
} from "./vaultStorage";
import { encryptWithKey, decryptWithKey } from "./secureVault";
import type { LockMode } from "./constants";

export interface VaultSession {
  mode: LockMode;
  key?: CryptoKey | null;
  token?: string;
}

export interface OfflineCapture {
  id: string;
  request: CaptureRequest;
  token?: string;
  timestamp: number;
}

function isLocked(
  session?: VaultSession,
): session is VaultSession & { key: CryptoKey } {
  return !!session && session.mode !== "none" && !!session.key;
}

export async function saveOffline(
  capture: CaptureRequest,
  session?: VaultSession,
): Promise<string> {
  const id = `offline-${Date.now()}-${Math.random().toString(36).slice(2)}`;
  const token = session?.token;
  const timestamp = Date.now();

  if (isLocked(session)) {
    const plaintext = JSON.stringify({ request: capture, token: token ?? null });
    const cipher = await encryptWithKey(session.key, plaintext);
    await putOfflineItem({ id, cipher, timestamp });
  } else {
    await putOfflineItem({ id, request: capture, token, timestamp });
  }

  triggerBackgroundSync();

  return id;
}

export async function getOfflineQueue(
  session?: VaultSession,
): Promise<OfflineCapture[]> {
  const items = await getAllOfflineItems();
  const result: OfflineCapture[] = [];

  for (const item of items) {
    if (item.cipher) {
      if (!isLocked(session)) continue;
      try {
        const plaintext = await decryptWithKey(session.key, item.cipher);
        const parsed = JSON.parse(plaintext) as {
          request: CaptureRequest;
          token?: string | null;
        };
        result.push({
          id: item.id,
          request: parsed.request,
          token: parsed.token ?? undefined,
          timestamp: item.timestamp,
        });
      } catch {
        // Skip items that fail to decrypt (wrong key / corrupted)
      }
    } else if (item.request) {
      result.push({
        id: item.id,
        request: item.request as CaptureRequest,
        token: item.token,
        timestamp: item.timestamp,
      });
    }
  }

  return result.sort((a, b) => a.timestamp - b.timestamp);
}

export async function removeOfflineItem(id: string): Promise<void> {
  await deleteOfflineItem(id);
}

export async function flushOfflineQueue(
  client: KhayalClient,
  session?: VaultSession,
): Promise<void> {
  const queue = await getOfflineQueue(session);

  for (const item of queue) {
    try {
      await client.capture(item.request);
      await removeOfflineItem(item.id);
    } catch {
      // If one fails, stop the flush (will retry on next sync)
      break;
    }
  }
}

export function setupOfflineSync(
  host: string,
  token: string,
  session?: VaultSession,
): void {
  const client = new KhayalClient(host, token);

  const flush = () => flushOfflineQueue(client, session);

  window.addEventListener("focus", flush);
  window.addEventListener("online", flush);

  // Initial flush
  if (navigator.onLine) {
    flush();
  }
}

export async function encryptOfflineQueue(
  key: CryptoKey,
  token?: string,
): Promise<void> {
  const items = await getAllOfflineItems();
  for (const item of items) {
    if (item.cipher || !item.request) continue;
    const plaintext = JSON.stringify({
      request: item.request,
      token: token ?? item.token ?? null,
    });
    const cipher = await encryptWithKey(key, plaintext);
    await putOfflineItem({ id: item.id, cipher, timestamp: item.timestamp });
  }
}

export async function decryptOfflineQueue(key: CryptoKey): Promise<void> {
  const items = await getAllOfflineItems();
  for (const item of items) {
    if (!item.cipher) continue;
    try {
      const plaintext = await decryptWithKey(key, item.cipher);
      const parsed = JSON.parse(plaintext) as {
        request: CaptureRequest;
        token?: string | null;
      };
      await putOfflineItem({
        id: item.id,
        request: parsed.request,
        token: parsed.token ?? undefined,
        timestamp: item.timestamp,
      });
    } catch {
      // Skip items that fail to decrypt
    }
  }
}

async function triggerBackgroundSync(): Promise<void> {
  if (
    "serviceWorker" in navigator &&
    "sync" in ServiceWorkerRegistration.prototype
  ) {
    try {
      const registration = await navigator.serviceWorker.ready;
      await (registration as any).sync.register("sync-offline-captures");
    } catch {
      // Background sync not supported or permission denied
      // Fallback: send message to SW
      if (navigator.serviceWorker.controller) {
        navigator.serviceWorker.controller.postMessage({
          type: "SYNC_OFFLINE",
        });
      }
    }
  }
}

export async function getOfflineCount(session?: VaultSession): Promise<number> {
  const queue = await getOfflineQueue(session);
  return queue.length;
}
