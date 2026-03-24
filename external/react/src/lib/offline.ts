import { get, set, del, keys } from "idb-keyval";
import { KhayalClient } from "./api";
import type { CaptureRequest } from "./api";
import { STORAGE_KEYS } from "./constants";

interface OfflineCapture {
  id: string;
  request: CaptureRequest;
  token?: string;
  timestamp: number;
}

const DB_PREFIX = "khayal-offline-";

export async function saveOffline(capture: CaptureRequest): Promise<string> {
  const id = `offline-${Date.now()}-${Math.random().toString(36).slice(2)}`;
  const token = localStorage.getItem(STORAGE_KEYS.TOKEN) || undefined;
  const item: OfflineCapture = {
    id,
    request: capture,
    token,
    timestamp: Date.now(),
  };
  await set(DB_PREFIX + id, item);

  // Trigger background sync if service worker is available
  triggerBackgroundSync();

  return id;
}

export async function getOfflineQueue(): Promise<OfflineCapture[]> {
  const allKeys = await keys();
  const offlineKeys = allKeys.filter((k) => String(k).startsWith(DB_PREFIX));

  const items: OfflineCapture[] = [];
  for (const key of offlineKeys) {
    const item = await get<OfflineCapture>(key);
    if (item) items.push(item);
  }

  return items.sort((a, b) => a.timestamp - b.timestamp);
}

export async function removeOfflineItem(id: string): Promise<void> {
  await del(DB_PREFIX + id);
}

export async function flushOfflineQueue(client: KhayalClient): Promise<void> {
  const queue = await getOfflineQueue();

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

export function setupOfflineSync(host: string, token: string): void {
  const client = new KhayalClient(host, token);

  const flush = () => flushOfflineQueue(client);

  window.addEventListener("focus", flush);
  window.addEventListener("online", flush);

  // Initial flush
  if (navigator.onLine) {
    flush();
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

export async function getOfflineCount(): Promise<number> {
  const queue = await getOfflineQueue();
  return queue.length;
}
