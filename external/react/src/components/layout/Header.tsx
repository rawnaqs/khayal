import { useState } from "react";
import { ArrowUpCircle, Shield } from "lucide-react";
import { useServerStatus } from "@/hooks/useServerStatus";
import { SecuritySheet } from "@/components/settings/SecuritySheet";
import { APP_VERSION, GITHUB_RELEASES_URL } from "@/lib/constants";

export function Header() {
  const { status, health } = useServerStatus();
  const [securityOpen, setSecurityOpen] = useState(false);

  const version = health?.version || APP_VERSION;
  const hasUpdate = health?.update?.available;

  const onlineColor =
    status === "ok" ? "#3ddc84" : status === "degraded" ? "#ffb340" : "#ff4d4d";

  return (
    <header className="hdr">
      <div className="brand">
        <img src="/icon.svg" alt="khayal" className="mark" />
        <span className="bname">khayal</span>
        <span className="ver self-end">v{version}</span>
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
        <button
          onClick={() => setSecurityOpen(true)}
          className="flex items-center justify-center w-6 h-6 rounded-md text-[rgba(245,245,245,0.4)] hover:text-[rgba(245,245,245,0.8)] transition-colors"
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
