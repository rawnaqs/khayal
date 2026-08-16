import { Check, Loader2, Zap, X, AlertTriangle } from "lucide-react";
import type { CaptureResponse } from "@/lib/api";
import { PROCESSING_STEPS } from "@/lib/constants";

interface CaptureResultProps {
  result: CaptureResponse | null;
  error: string | null;
  errorCode?: string;
  isOffline: boolean;
  processingTime?: number;
  onDismiss: () => void;
  onRetry: () => void;
}

export function getStepsForType(type: string): string[] {
  return PROCESSING_STEPS[type] || PROCESSING_STEPS.text;
}

function parseError(error: string): { code: string; message: string } {
  const parts = error.split("·").map((s) => s.trim());
  if (parts.length >= 2)
    return { code: parts[0], message: parts.slice(1).join(" · ") };
  return { code: "ERR", message: error };
}

function SuccessTile({
  result,
  processingTime,
  onDismiss,
}: {
  result: CaptureResponse;
  processingTime?: number;
  onDismiss: () => void;
}) {
  const subtitle = processingTime
    ? `${result.type} · ${processingTime}ms`
    : result.type;

  return (
    <div className="tile tile-ok" role="status" aria-live="polite">
      <div className="icon-ok" aria-hidden="true">
        <Check className="w-4 h-4" style={{ color: "var(--ok)" }} />
      </div>
      <div className="tile-inner">
        <div className="tile-top">
          <span className="tile-title">saved</span>
          <button
            className="tile-dismiss"
            onClick={onDismiss}
            aria-label="dismiss"
          >
            <X className="w-2 h-2" aria-hidden="true" />
          </button>
        </div>
        <div className="tile-sub">{subtitle}</div>
        {result.note_path && (
          <div className="tile-sub" style={{ opacity: 0.5 }}>
            {result.note_path}
          </div>
        )}
        <div className="tile-bar">
          <div className="tile-bar-fill" />
        </div>
      </div>
    </div>
  );
}

function QueuedTile({
  result,
  onDismiss,
}: {
  result: CaptureResponse;
  onDismiss: () => void;
}) {
  const steps = getStepsForType(result.type);
  // Simulate: saved=done, others=waiting
  const activeStep = 1;

  return (
    <div className="tile tile-q" role="status" aria-live="polite">
      <div className="icon-q" aria-hidden="true">
        <Loader2 className="w-4 h-4" style={{ color: "var(--warn)" }} />
      </div>
      <div className="tile-inner">
        <div className="tile-top">
          <span className="tile-title">queued</span>
          <button
            className="tile-dismiss"
            onClick={onDismiss}
            aria-label="dismiss"
          >
            <X className="w-2 h-2" aria-hidden="true" />
          </button>
        </div>
        <div className="tile-sub">
          {result.note_path || result.type} · {result.id.slice(0, 8)}
        </div>
        <div className="steps">
          {steps.map((step, i) => (
            <span key={step}>
              <div
                className={`sd ${i < activeStep ? "done" : i === activeStep ? "act" : "wait"}`}
              />
              <span
                className={`sl ${i < activeStep ? "done" : i === activeStep ? "act" : ""}`}
              >
                {step}
              </span>
              {i < steps.length - 1 && <span className="sep">·</span>}
            </span>
          ))}
        </div>
        <div className="tile-bar">
          <div className="tile-bar-fill" />
        </div>
      </div>
    </div>
  );
}

function OfflineTile({ onDismiss }: { onDismiss: () => void }) {
  return (
    <div className="tile tile-off" role="status" aria-live="polite">
      <div className="icon-off" aria-hidden="true">
        <Zap className="w-4 h-4" style={{ color: "var(--gold)" }} />
      </div>
      <div className="tile-inner">
        <div className="tile-top">
          <span className="tile-title">saved offline</span>
          <button
            className="tile-dismiss"
            onClick={onDismiss}
            aria-label="dismiss"
          >
            <X className="w-2 h-2" aria-hidden="true" />
          </button>
        </div>
        <div className="tile-sub">will sync when connected</div>
        <div className="tile-bar">
          <div className="tile-bar-fill" />
        </div>
      </div>
    </div>
  );
}

function ErrorTile({
  error,
  onRetry,
  onDismiss,
}: {
  error: string;
  onRetry: () => void;
  onDismiss: () => void;
}) {
  const { code, message } = parseError(error);

  return (
    <div className="tile tile-err" role="alert">
      <div className="icon-err" aria-hidden="true">
        <AlertTriangle className="w-4 h-4" style={{ color: "var(--bad)" }} />
      </div>
      <div className="tile-inner">
        <div className="tile-top">
          <span className="tile-title">capture failed</span>
        </div>
        <div className="tile-sub">
          {code} · {message}
        </div>
        <div className="err-box">
          <div className="err-code">{code}</div>
          <div className="err-hint">{message}</div>
        </div>
        <div className="err-actions">
          <button className="ea" onClick={onDismiss}>
            discard
          </button>
          <button className="ea p" onClick={onRetry}>
            retry
          </button>
        </div>
      </div>
    </div>
  );
}

export function CaptureResult({
  result,
  error,
  errorCode,
  isOffline,
  processingTime,
  onDismiss,
  onRetry,
}: CaptureResultProps) {
  if (error)
    return (
      <ErrorTile
        error={errorCode ? `${errorCode} · ${error}` : error}
        onRetry={onRetry}
        onDismiss={onDismiss}
      />
    );
  if (isOffline) return <OfflineTile onDismiss={onDismiss} />;
  if (result) {
    const isQueued = result.status === "pending" || result.status === "queued";
    return isQueued ? (
      <QueuedTile result={result} onDismiss={onDismiss} />
    ) : (
      <SuccessTile
        result={result}
        processingTime={processingTime}
        onDismiss={onDismiss}
      />
    );
  }
  return null;
}
