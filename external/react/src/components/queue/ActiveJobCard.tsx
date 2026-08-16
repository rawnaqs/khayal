import type { QueueJob } from "@/lib/api";
import { timeAgo } from "@/lib/time";
import { getStepsForType } from "../capture/CaptureResult";

interface ActiveJobCardProps {
  job: QueueJob;
}

export function ActiveJobCard({ job }: ActiveJobCardProps) {
  const steps = getStepsForType(job.type);

  return (
    <>
      <div className="sec">now processing</div>
      <div className="hero-card">
        <div className="hero-top">
          <div>
            <div className="hero-filename">{job.note_path || job.type}</div>
            <div className="hero-meta">
              {job.type} · {timeAgo(job.created_at)}
            </div>
          </div>
          <div className="hero-badge">
            <div className="badge-dot" />
            live
          </div>
        </div>
        <div className="prog-labels">
          {steps.map((step, i) => (
            <span key={step} className={`prog-step ${i === 0 ? "done" : ""}`}>
              {step}
            </span>
          ))}
        </div>
        <div className="prog-bar">
          <div
            className="prog-fill"
            style={{ animation: "indeterminate 2s linear infinite" }}
          />
        </div>
      </div>
    </>
  );
}
