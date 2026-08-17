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
        this device doesn't support face id / touch id unlock. you can choose
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
        require face id / touch id to open khayal on this device. you can turn
        this on or off anytime in the security sheet.
      </p>
      <p className="text-muted-foreground text-xs opacity-70">
        protects against casual access to an unlocked device — not a fully
        compromised device. if you lose this device's face id, you can always
        reconnect using your server token; nothing is deleted.
      </p>
      <Button onClick={beginPrf}>set up face id</Button>
      <Button variant="ghost" onClick={onRemember}>
        skip for now
      </Button>
    </div>
  );
}
