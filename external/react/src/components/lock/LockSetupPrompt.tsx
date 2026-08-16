import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { isPlatformAuthenticatorAvailable } from "@/lib/secureVault";

export function TokenRememberChoice({
  onRemember,
  onDontRemember,
}: {
  onRemember: () => void;
  onDontRemember: () => void;
}) {
  return (
    <div className="flex flex-col gap-3">
      <span className="text-lg font-bold">face id isn't supported here</span>
      <p className="text-muted-foreground text-sm">
        This device doesn't support Face ID / Touch ID unlock. You can choose
        not to remember your token, so you'll enter it each time you open
        khayal (e.g. from your password manager).
      </p>
      <Button onClick={onDontRemember}>don't remember my token</Button>
      <Button variant="ghost" onClick={onRemember}>
        remember my token
      </Button>
    </div>
  );
}

interface LockSetupPromptProps {
  onSetupPrf: () => Promise<boolean>;
  onRemember: () => void;
  onDontRemember: () => void;
}

type Phase = "checking" | "prf" | "working" | "noauth";

export function LockSetupPrompt({
  onSetupPrf,
  onRemember,
  onDontRemember,
}: LockSetupPromptProps) {
  const [phase, setPhase] = useState<Phase>("checking");

  useEffect(() => {
    let cancelled = false;
    isPlatformAuthenticatorAvailable().then((available) => {
      if (cancelled) return;
      setPhase(available ? "prf" : "noauth");
    });
    return () => {
      cancelled = true;
    };
  }, []);

  const beginPrf = async () => {
    setPhase("working");
    const ok = await onSetupPrf();
    if (!ok) setPhase("noauth");
  };

  if (phase === "checking" || phase === "working") {
    return (
      <div className="flex flex-col gap-3 p-6 items-center text-center">
        <span className="text-lg font-bold">
          {phase === "checking"
            ? "checking your device..."
            : "waiting for biometrics..."}
        </span>
      </div>
    );
  }

  if (phase === "noauth") {
    return (
      <div className="p-6">
        <TokenRememberChoice
          onRemember={onRemember}
          onDontRemember={onDontRemember}
        />
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 p-6">
      <span className="text-lg font-bold">secure this device?</span>
      <p className="text-muted-foreground text-sm">
        Require Face ID / Touch ID to open khayal on this device. You can turn
        this on or off anytime in Settings.
      </p>
      <p className="text-muted-foreground text-xs opacity-70">
        Protects against casual access to an unlocked device — not a fully
        compromised device. If you lose this device's Face ID, you can always
        reconnect using your server token; nothing is deleted.
      </p>
      <Button onClick={beginPrf}>set up face id</Button>
      <Button variant="ghost" onClick={onRemember}>
        skip for now
      </Button>
    </div>
  );
}
