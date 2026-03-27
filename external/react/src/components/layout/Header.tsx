import { ArrowUpCircle } from "lucide-react";
import { useServerStatus } from "@/hooks/useServerStatus";
import { APP_VERSION, GITHUB_RELEASES_URL } from "@/lib/constants";

export function Header() {
  const { status, health } = useServerStatus();

  const version = health?.version || APP_VERSION;
  const hasUpdate = health?.update?.available;

  const onlineColor =
    status === "ok" ? "#3ddc84" : status === "degraded" ? "#ffb340" : "#ff4d4d";

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
            title={`update to v${health?.update?.latest}`}
          >
            <ArrowUpCircle size={14} className="update-icon" />
          </a>
        )}
        {version && <span className="ver">v{version}</span>}
        <div
          className="online"
          style={{
            background: onlineColor,
            boxShadow: status === "ok" ? `0 0 8px ${onlineColor}` : "none",
          }}
        />
      </div>
    </header>
  );
}
