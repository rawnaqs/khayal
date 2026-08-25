import { useState } from "react";
import { ArrowUpCircle, LogOut, Shield, X } from "lucide-react";
import { useServerStatus } from "@/hooks/useServerStatus";
import { useVaultLock } from "@/hooks/useVaultLock";
import { SecuritySheet } from "@/components/settings/SecuritySheet";
import { APP_VERSION, GITHUB_RELEASES_URL, STORAGE_KEYS } from "@/lib/constants";

export function Header() {
  const { status, health } = useServerStatus();
  const { lockMode, lock } = useVaultLock();
  const [securityOpen, setSecurityOpen] = useState(false);
  const [confirmLogout, setConfirmLogout] = useState(false);

  const handleLogout = () => {
    localStorage.removeItem(STORAGE_KEYS.TOKEN);
    if (lockMode !== "none") {
      lock();
      setConfirmLogout(false);
    } else {
      // no vault lock configured: restart into a clean unauthenticated state
      window.location.reload();
    }
  };

  const version = health?.version || APP_VERSION;
  const hasUpdate = health?.update?.available;

  const onlineColor =
    status === "ok" ? "#3ddc84" : status === "degraded" ? "#ffb340" : "#ff4d4d";

  return (
    <header className="hdr">
      <div className="brand">
        <img src="/icon.svg" alt="khayal" className="mark" />
        <span className="bname">
          khayal
          <span className="ver">v{version}</span>
        </span>
        {hasUpdate && (
          <a
            href={GITHUB_RELEASES_URL}
            target="_blank"
            rel="noopener noreferrer"
            title={`update to v${health?.update?.latest}`}
          >
            <ArrowUpCircle size={14} className="update-icon" />
          </a>
        )}
      </div>
      <div className="flex items-center gap-2">
        {confirmLogout ? (
          <div className="flex items-center gap-1.5" data-testid="logout-confirm">
            <span className="text-[10px] font-mono whitespace-nowrap" style={{ color: "rgba(245,169,169,0.8)" }}>
              lock out?
            </span>
            <button
              onClick={handleLogout}
              className="flex items-center justify-center w-6 h-6 min-h-0 rounded-md"
              style={{ color: "#f5a9a9", background: "rgba(255,99,99,0.08)", border: "1px solid rgba(255,99,99,0.25)" }}
              title="confirm logout"
              aria-label="confirm logout"
              data-testid="logout-go"
            >
              <LogOut size={12} />
            </button>
            <button
              onClick={() => setConfirmLogout(false)}
              className="flex items-center justify-center w-6 h-6 min-h-0 rounded-md text-[rgba(245,245,245,0.3)] hover:text-[rgba(245,245,245,0.6)] transition-colors"
              title="cancel"
              aria-label="cancel logout"
              data-testid="logout-cancel"
            >
              <X size={12} />
            </button>
          </div>
        ) : (
          <button
            onClick={() => setConfirmLogout(true)}
            className="flex items-center justify-center w-6 h-6 min-h-0 rounded-md text-[rgba(245,245,245,0.4)] hover:text-[rgba(245,169,169,0.9)] transition-colors"
            title="logout"
            aria-label="logout"
            data-testid="logout-trigger"
          >
            <LogOut size={14} />
          </button>
        )}
        <button
          onClick={() => setSecurityOpen(true)}
          className="flex items-center justify-center w-6 h-6 min-h-0 rounded-md text-[rgba(245,245,245,0.4)] hover:text-[rgba(245,245,245,0.8)] transition-colors"
          title="security"
          aria-label="security"
        >
          <Shield size={14} />
        </button>
        <div
          className="online"
          style={{
            background: onlineColor,
            boxShadow: status === "ok" ? `0 0 8px ${onlineColor}` : "none",
          }}
        />
      </div>
      <SecuritySheet open={securityOpen} onOpenChange={setSecurityOpen} />
    </header>
  );
}
