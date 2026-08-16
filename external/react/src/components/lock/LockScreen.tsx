import { useCallback, useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { useVaultLock } from "@/hooks/useVaultLock";

export function LockScreen() {
  const { unlock, resetToOnboarding } = useVaultLock();
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);
  const autoStarted = useRef(false);

  const doUnlock = useCallback(async () => {
    setBusy(true);
    setError("");
    const ok = await unlock();
    setBusy(false);
    if (!ok) setError("Could not unlock. Try again.");
  }, [unlock]);

  useEffect(() => {
    if (autoStarted.current) return;
    autoStarted.current = true;
    doUnlock();
  }, [doUnlock]);

  return (
    <div className="flex flex-col items-center justify-center h-screen p-6 bg-background">
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.4, ease: "easeOut" }}
      >
        <Card className="w-full max-w-sm glass border-primary/20 shadow-[0_0_40px_hsl(var(--primary)/0.1)]">
          <CardHeader className="text-center pb-2">
            <div className="w-16 h-16 mx-auto mb-4 rounded-2xl flex items-center justify-center">
              <img src="/icon.svg" alt="khayal" className="w-16 h-16" />
            </div>
            <CardTitle className="text-2xl font-bold tracking-tight">
              khayal
            </CardTitle>
            <p className="text-caption mt-1">locked</p>
          </CardHeader>
          <CardContent className="space-y-4 pt-4">
            <Button
              className="w-full h-12 btn-gradient font-semibold tracking-wide"
              onClick={doUnlock}
              disabled={busy}
            >
              {busy ? (
                <span className="animate-pulse">unlocking...</span>
              ) : (
                "unlock with face id"
              )}
            </Button>

            {error && (
              <motion.p
                initial={{ opacity: 0, y: -4 }}
                animate={{ opacity: 1, y: 0 }}
                className="text-sm text-destructive text-center"
              >
                {error}
              </motion.p>
            )}

            <Button
              variant="ghost"
              className="w-full text-muted-foreground text-xs"
              onClick={resetToOnboarding}
            >
              can't unlock? reconnect instead
            </Button>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  );
}
