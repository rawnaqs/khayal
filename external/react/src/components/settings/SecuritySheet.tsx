import { useEffect, useState } from "react";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from "@/components/ui/sheet";
import { useVaultLock } from "@/hooks/useVaultLock";
import { useToast } from "@/hooks/use-toast";
import { TokenRememberChoice } from "@/components/lock/LockSetupPrompt";

interface SecuritySheetProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function SecuritySheet({ open, onOpenChange }: SecuritySheetProps) {
  const { lockMode, setupPrf, disable, setTokenPersistence } = useVaultLock();
  const { toast } = useToast();
  const [setupState, setSetupState] = useState<
    "idle" | "working" | "choice"
  >("idle");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (!open) {
      setSetupState("idle");
      setError("");
      setBusy(false);
    }
  }, [open]);

  const trySetupPrf = async () => {
    setSetupState("working");
    setError("");
    const ok = await setupPrf();
    if (ok) {
      setSetupState("idle");
      toast({ title: "Lock enabled" });
    } else {
      setSetupState("choice");
    }
  };

  const handleRemember = () => {
    setTokenPersistence(true);
    setSetupState("idle");
    toast({ title: "Token will be remembered" });
  };

  const handleDontRemember = () => {
    setTokenPersistence(false);
    setSetupState("idle");
    toast({ title: "Token will not be remembered" });
  };

  const handleDisable = async () => {
    setBusy(true);
    setError("");
    const ok = await disable();
    setBusy(false);
    if (ok) {
      toast({ title: "Lock disabled" });
    } else {
      setError("couldn't verify. lock stays on.");
    }
  };

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent side="bottom">
        <SheetHeader>
          <SheetTitle>security</SheetTitle>
        </SheetHeader>

        {lockMode === "none" && (
          <>
            {setupState === "idle" && (
              <div className="flex flex-col gap-3 pt-2">
                <p className="text-muted-foreground text-sm">
                  khayal is not locked on this device.
                </p>
                <Button onClick={trySetupPrf}>set up face id</Button>
              </div>
            )}
            {setupState === "working" && (
              <p className="text-sm text-muted-foreground text-center pt-2">
                waiting for biometrics...
              </p>
            )}
            {setupState === "choice" && (
              <div className="pt-2">
                <TokenRememberChoice
                  onRemember={handleRemember}
                  onDontRemember={handleDontRemember}
                />
              </div>
            )}
          </>
        )}

        {lockMode === "prf" && (
          <div className="flex flex-col gap-3 pt-2">
            <div className="flex items-center justify-between">
              <span className="text-sm">require face id to open</span>
              <Switch
                checked={true}
                onCheckedChange={(checked) => {
                  if (!checked) handleDisable();
                }}
              />
            </div>
            {busy && (
              <p className="text-sm text-muted-foreground text-center">
                verifying...
              </p>
            )}
            {error && (
              <p className="text-sm text-destructive text-center">{error}</p>
            )}
          </div>
        )}
      </SheetContent>
    </Sheet>
  );
}
