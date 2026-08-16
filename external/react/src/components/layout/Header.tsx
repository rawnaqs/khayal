import { ArrowUpCircle, LogOut } from "lucide-react";
import { useServerStatus } from "@/hooks/useServerStatus";
import { APP_VERSION, GITHUB_RELEASES_URL } from "@/lib/constants";

interface HeaderProps {
  onLogout: () => void;
}

export function Header({ onLogout }: HeaderProps) {
  const { status, health } = useServerStatus();

  const version = health?.version || APP_VERSION;
  const hasUpdate = health?.update?.available;

  const onlineColor =
    status === "ok"
      ? "var(--ok)"
      : status === "degraded"
        ? "var(--warn)"
        : "var(--bad)";

  const statusLabel =
    status === "ok"
      ? "connected"
      : status === "degraded"
        ? "degraded"
        : "offline";

  return (
    <header className="hdr">
      <div className="brand">
        <img src="/icon.svg" alt="khayal" className="mark" />
        <span className="bname">khayal</span>
      </div>
      <div className="flex items-center gap-2">
        {hasUpdate && (
          <a
            href={GITHUB_RELEASES_URL}
            target="_blank"
            rel="noopener noreferrer"
            aria-label={`update to version ${health?.update?.latest}`}
          >
            <ArrowUpCircle
              size={14}
              className="update-icon"
              aria-hidden="true"
            />
          </a>
        )}
        {version && <span className="ver">v{version}</span>}
        <button
          onClick={onLogout}
          aria-label="log out"
          className="flex items-center justify-center w-6 h-6 min-h-0 rounded-[7px] transition-colors duration-150 bg-[var(--s2)] border border-[var(--border2)] text-[var(--t3)] hover:border-[var(--gold)] hover:text-[var(--gold)]"
        >
          <LogOut size={11} />
        </button>
        <div
          className="online"
          role="status"
          aria-label={statusLabel}
          style={{
            background: onlineColor,
            boxShadow: status === "ok" ? `0 0 8px var(--ok)` : "none",
          }}
        />
      </div>
    </header>
  );
}
