import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import type { ReactNode } from "react";
import { STORAGE_KEYS, VAULT_LOCK } from "@/lib/constants";
import type { LockMode } from "@/lib/constants";
import {
  isPlatformAuthenticatorAvailable,
  registerPrfCredential,
  getPrfOutput,
  deriveKeyFromPrf,
  encryptWithKey,
  decryptWithKey,
  generateSalt,
  fromBase64,
  toBase64,
} from "@/lib/secureVault";
import {
  getVaultRecord,
  saveVaultRecord,
  deleteVaultRecord,
  migrateOfflineQueue,
} from "@/lib/vaultStorage";
import {
  setupOfflineSync,
  encryptOfflineQueue,
  decryptOfflineQueue,
} from "@/lib/offline";
import type { VaultSession } from "@/lib/offline";

export interface VaultLockContextValue {
  lockMode: LockMode;
  locked: boolean;
  configured: boolean;
  token: string | null;
  vaultKey: CryptoKey | null;
  session: VaultSession;
  unlock: () => Promise<boolean>;
  setupPrf: (tokenArg?: string) => Promise<boolean>;
  completeOnboarding: (token: string, persist: boolean) => void;
  setTokenPersistence: (persist: boolean) => void;
  disable: () => Promise<boolean>;
  lock: () => void;
  resetToOnboarding: () => Promise<void>;
}

const VaultLockContext = createContext<VaultLockContextValue | null>(null);

export function VaultLockProvider({ children }: { children: ReactNode }) {
  const [lockMode, setLockMode] = useState<LockMode>("none");
  const [locked, setLocked] = useState(false);
  const [configured, setConfigured] = useState(false);
  const [token, setToken] = useState<string | null>(null);
  const [vaultKey, setVaultKey] = useState<CryptoKey | null>(null);
  const [loaded, setLoaded] = useState(false);

  const syncTokenRef = useRef<string | null>(null);

  // Initialize: migrate legacy queue, then determine lock state
  useEffect(() => {
    let cancelled = false;
    (async () => {
      await migrateOfflineQueue();
      const record = await getVaultRecord();
      if (cancelled) return;

      if (record && record.mode === "prf") {
        setLockMode("prf");
        setLocked(true);
        setConfigured(true);
      } else {
        const storedToken = localStorage.getItem(STORAGE_KEYS.TOKEN);
        const host = localStorage.getItem(STORAGE_KEYS.HOST);
        if (storedToken && host) {
          setToken(storedToken);
          setLockMode("none");
          setLocked(false);
          setConfigured(true);
        } else {
          setLockMode("none");
          setLocked(false);
          setConfigured(false);
        }
      }
      setLoaded(true);
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Drive offline sync whenever a decrypted token is available
  useEffect(() => {
    if (!token) return;
    const host =
      localStorage.getItem(STORAGE_KEYS.HOST) || window.location.origin;
    if (!host || syncTokenRef.current === token) return;
    syncTokenRef.current = token;
    setupOfflineSync(host, token, {
      mode: lockMode,
      key: vaultKey,
      token,
    });
  }, [token, lockMode, vaultKey]);

  const finalizeEnable = useCallback(
    async (
      key: CryptoKey,
      currentToken: string,
      credentialId: string,
      salt: Uint8Array,
    ) => {
      const encryptedToken = await encryptWithKey(key, currentToken);
      await saveVaultRecord({
        id: "vault",
        mode: "prf",
        credentialId,
        salt: toBase64(salt),
        encryptedToken,
      });
      await encryptOfflineQueue(key, currentToken);
      localStorage.removeItem(STORAGE_KEYS.TOKEN);
      localStorage.setItem(STORAGE_KEYS.LOCK_SETUP_DECIDED, "1");
      setToken(currentToken);
      setVaultKey(key);
      setLockMode("prf");
      setLocked(false);
      setConfigured(true);
    },
    [],
  );

  const setupPrf = useCallback(
    async (tokenArg?: string): Promise<boolean> => {
      const available = await isPlatformAuthenticatorAvailable();
      if (!available) return false;
      try {
        const { credentialId, prfEnabled } = await registerPrfCredential();
        if (!prfEnabled) return false;

        const currentToken =
          tokenArg ?? token ?? localStorage.getItem(STORAGE_KEYS.TOKEN) ?? "";
        if (!currentToken) return false;

        const salt = generateSalt(VAULT_LOCK.PRF_SALT_BYTES);
        const output = await getPrfOutput(credentialId, salt);
        const key = await deriveKeyFromPrf(output);
        await finalizeEnable(key, currentToken, credentialId, salt);
        return true;
      } catch {
        return false;
      }
    },
    [token, finalizeEnable],
  );

  const completeOnboarding = useCallback(
    (newToken: string, persist: boolean) => {
      localStorage.setItem(STORAGE_KEYS.LOCK_SETUP_DECIDED, "1");
      if (persist) {
        localStorage.setItem(STORAGE_KEYS.HOST, window.location.origin);
        localStorage.setItem(STORAGE_KEYS.TOKEN, newToken);
      } else {
        localStorage.removeItem(STORAGE_KEYS.TOKEN);
      }
      setToken(newToken);
      setVaultKey(null);
      setLockMode("none");
      setLocked(false);
      setConfigured(true);
    },
    [],
  );

  const setTokenPersistence = useCallback(
    (persist: boolean) => {
      localStorage.setItem(STORAGE_KEYS.LOCK_SETUP_DECIDED, "1");
      if (persist) {
        localStorage.setItem(STORAGE_KEYS.HOST, window.location.origin);
        if (token) localStorage.setItem(STORAGE_KEYS.TOKEN, token);
      } else {
        localStorage.removeItem(STORAGE_KEYS.TOKEN);
      }
    },
    [token],
  );

  const unlock = useCallback(async (): Promise<boolean> => {
    const record = await getVaultRecord();
    if (!record || record.mode !== "prf" || !record.credentialId) return false;
    try {
      const output = await getPrfOutput(
        record.credentialId,
        fromBase64(record.salt),
      );
      const key = await deriveKeyFromPrf(output);
      const plaintext = await decryptWithKey(key, record.encryptedToken);
      setToken(plaintext);
      setVaultKey(key);
      setLockMode("prf");
      setLocked(false);
      return true;
    } catch {
      return false;
    }
  }, []);

  const disable = useCallback(async (): Promise<boolean> => {
    const record = await getVaultRecord();
    if (!record || record.mode !== "prf" || !record.credentialId) return false;
    try {
      const output = await getPrfOutput(
        record.credentialId,
        fromBase64(record.salt),
      );
      const key = await deriveKeyFromPrf(output);
      const plaintext = await decryptWithKey(key, record.encryptedToken);
      localStorage.setItem(STORAGE_KEYS.TOKEN, plaintext);
      await decryptOfflineQueue(key);
      await deleteVaultRecord();
      setLockMode("none");
      setLocked(false);
      setVaultKey(null);
      setToken(plaintext);
      return true;
    } catch {
      return false;
    }
  }, []);

  const lock = useCallback(() => {
    if (lockMode === "none") return;
    setToken(null);
    setVaultKey(null);
    setLocked(true);
  }, [lockMode]);

  const resetToOnboarding = useCallback(async () => {
    await deleteVaultRecord();
    localStorage.removeItem(STORAGE_KEYS.TOKEN);
    localStorage.removeItem(STORAGE_KEYS.HOST);
    syncTokenRef.current = null;
    setLockMode("none");
    setLocked(false);
    setToken(null);
    setVaultKey(null);
    setConfigured(false);
  }, []);

  const session = useMemo<VaultSession>(
    () => ({ mode: lockMode, key: vaultKey, token: token ?? undefined }),
    [lockMode, vaultKey, token],
  );

  const value = useMemo<VaultLockContextValue>(
    () => ({
      lockMode,
      locked,
      configured,
      token,
      vaultKey,
      session,
      unlock,
      setupPrf,
      completeOnboarding,
      setTokenPersistence,
      disable,
      lock,
      resetToOnboarding,
    }),
    [
      lockMode,
      locked,
      configured,
      token,
      vaultKey,
      session,
      unlock,
      setupPrf,
      completeOnboarding,
      setTokenPersistence,
      disable,
      lock,
      resetToOnboarding,
    ],
  );

  if (!loaded) return null;

  return (
    <VaultLockContext.Provider value={value}>
      {children}
    </VaultLockContext.Provider>
  );
}

export function useVaultLock(): VaultLockContextValue {
  const ctx = useContext(VaultLockContext);
  if (!ctx) {
    throw new Error("useVaultLock must be used within a VaultLockProvider");
  }
  return ctx;
}
