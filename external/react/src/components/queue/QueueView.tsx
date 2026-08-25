import { useState, useEffect, useCallback } from "react";
import { motion } from "framer-motion";
import { RefreshCw, FileText, Link, Image, ChevronDown, ChevronUp } from "lucide-react";
import { QueueMetrics } from "./QueueMetrics";
import { ActiveJobCard } from "./ActiveJobCard";
import { FailedJobCard } from "./FailedJobCard";
import { FailedJobExpanded } from "./FailedJobExpanded";
import { RetryAllBanner } from "./RetryAllBanner";
import { DoneItem } from "./DoneItem";
import { OfflineSection } from "./OfflineSection";
import { LIMITS } from "@/lib/constants";
import { useQueue } from "@/hooks/useQueue";
import { useToast } from "@/hooks/use-toast";
import { useVaultLock } from "@/hooks/useVaultLock";
import { getOfflineQueue } from "@/lib/offline";
import { cn } from "@/lib/utils";

function timeAgo(dateStr: string) {
  try {
    const date = new Date(dateStr);
    const now = new Date();
    const diff = Math.floor((now.getTime() - date.getTime()) / 1000);
    if (diff < 60) return `${diff}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    return `${Math.floor(diff / 3600)}h ago`;
  } catch {
    return "";
  }
}

function getTypeIcon(type: string) {
  switch (type) {
    case "text":
      return <FileText className="w-4 h-4" />;
    case "url":
      return <Link className="w-4 h-4" />;
    case "image":
      return <Image className="w-4 h-4" />;
    default:
      return <FileText className="w-4 h-4" />;
  }
}

function getTypeIconClass(type: string) {
  switch (type) {
    case "text":
      return "t";
    case "url":
      return "u";
    case "image":
      return "i";
    default:
      return "t";
  }
}

function truncateContent(content: string, maxLen = 50) {
  if (!content) return "";
  if (content.length <= maxLen) return content;
  return content.slice(0, maxLen - 3) + "...";
}

interface QueueViewProps {
  onNoteSelect?: (notePath: string) => void;
}

// Internal pipeline job types — surfaced as flares on their ingest job,
// never as user-visible entries.
const INTERNAL_JOB_TYPES = new Set(["connections", "memory"]);

export function QueueView({ onNoteSelect }: QueueViewProps = {}) {
  const {
    loading,
    jobs,
    flares,
    doneExpanded,
    doneLoadingMore,
    loadMoreDone,
    setDoneExpanded,
    fetchQueue,
    retryJob,
    discardJob,
  } = useQueue();
  const { toast } = useToast();
  const { session } = useVaultLock();
  const [offlineItems, setOfflineItems] = useState<
    Array<{ id: string; content: string; timestamp: number }>
  >([]);
  const [firstLoadDone, setFirstLoadDone] = useState(false);

  const handleRefresh = useCallback(() => {
    fetchQueue();
    getOfflineQueue(session).then((items) => {
      setOfflineItems(
        items.map((i) => ({
          id: i.id,
          content: i.request.content,
          timestamp: i.timestamp,
        })),
      );
    });
  }, [fetchQueue, session]);

  useEffect(() => {
    handleRefresh();
  }, [handleRefresh]);

  useEffect(() => {
    if (!loading && !firstLoadDone) setFirstLoadDone(true);
  }, [loading, firstLoadDone]);

  const handleRetry = async (id: string) => {
    await retryJob(id);
    toast({ title: "Job retried" });
  };

  const handleDiscard = async (id: string) => {
    await discardJob(id);
    toast({ title: "Job discarded" });
  };

  const handleRetryAll = async () => {
    for (const job of failedJobs) {
      await retryJob(job.id);
    }
    toast({ title: `Retried ${failedJobs.length} jobs` });
  };

  // Derive job groups — internal pipeline types filtered out everywhere
  const userJobs = jobs.filter((j) => !INTERNAL_JOB_TYPES.has(j.type));
  const processingJob = userJobs.find((j) => j.status === "processing");
  const pendingJobs = userJobs.filter(
    (j) => j.status === "pending" || j.status === "queued",
  );
  const failedJobs = userJobs.filter((j) => j.status === "failed");
  const doneJobs = userJobs.filter((j) => j.status === "done");
  const visibleDone = doneExpanded
    ? doneJobs
    : doneJobs.slice(0, LIMITS.DONE_JOBS_SHOWN);

  return (
    <div className="q-body">
      {/* First-load skeleton */}
      {!firstLoadDone && (
        <div className="q-list" data-testid="queue-skeleton">
          {[1, 2, 3, 4].map((i) => (
            <div key={i} className="q-skel-row">
              <div className="animate-shimmer q-skel q-skel-icon" />
              <div className="q-skel-lines">
                <div className="animate-shimmer q-skel q-skel-w60" />
                <div className="animate-shimmer q-skel q-skel-w35" />
              </div>
            </div>
          ))}
        </div>
      )}

      {firstLoadDone && (
        <>
          {/* Hero processing card */}
          {processingJob && <ActiveJobCard job={processingJob} />}

          {/* Queue metrics */}
          <QueueMetrics
            pending={pendingJobs.length}
            processing={processingJob ? 1 : 0}
            failed={failedJobs.length}
          />

          {/* Pending list */}
          {pendingJobs.length > 0 && (
            <>
              <div className="sec">pending ({pendingJobs.length})</div>
              <div className="q-list">
                {pendingJobs.map((job, index) => (
                  <motion.div
                    key={job.id}
                    initial={{ opacity: 0, y: 4 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: index * 0.02 }}
                    className="qi"
                  >
                    <div className={`qi-icon ${getTypeIconClass(job.type)}`}>
                      {getTypeIcon(job.type)}
                    </div>
                    <div className="qi-body">
                      <div className="qi-title">
                        {truncateContent(job.note_path || job.type)}
                      </div>
                      <div className="qi-meta">
                        {job.type} · {job.status}
                      </div>
                    </div>
                    <div
                      className={`qi-dot ${job.status === "queued" ? "q" : "p"}`}
                    />
                    <span className="qi-ago">{timeAgo(job.created_at)}</span>
                  </motion.div>
                ))}
              </div>
            </>
          )}

          {/* Failed section */}
          {failedJobs.length > 0 && (
            <>
              <div className="sec">failed ({failedJobs.length})</div>
              {failedJobs.length > 1 && (
                <RetryAllBanner
                  count={failedJobs.length}
                  onRetryAll={handleRetryAll}
                />
              )}
              <div className="q-list">
                {failedJobs.map((job, index) => (
                  <motion.div
                    key={job.id}
                    initial={{ opacity: 0, y: 4 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: index * 0.02 }}
                  >
                    {index === 0 ? (
                      <FailedJobExpanded
                        job={job}
                        onRetry={handleRetry}
                        onDiscard={handleDiscard}
                      />
                    ) : (
                      <FailedJobCard
                        job={job}
                        onRetry={handleRetry}
                        onDiscard={handleDiscard}
                      />
                    )}
                  </motion.div>
                ))}
              </div>
            </>
          )}

          {/* Done history */}
          {doneJobs.length > 0 && (
            <>
              <div className="divider" />
              <div className="sec">done ({doneJobs.length})</div>
              <div className="q-list">
                {visibleDone.map((job, index) => (
                  <motion.div
                    key={job.id}
                    initial={{ opacity: 0, y: 4 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: doneExpanded ? 0 : index * 0.02 }}
                  >
                    <DoneItem
                      job={job}
                      flare={flares[job.id]}
                      onSelect={onNoteSelect}
                    />
                  </motion.div>
                ))}
              </div>

              {(doneJobs.length > LIMITS.DONE_JOBS_SHOWN || doneExpanded) && (
                <button
                  className="done-expand clickable"
                  onClick={() =>
                    doneExpanded ? setDoneExpanded(false) : loadMoreDone()
                  }
                  disabled={doneLoadingMore}
                  data-testid="queue-show-more"
                >
                  {doneLoadingMore ? (
                    <>
                      <RefreshCw className="w-3 h-3 animate-spin" />
                      loading history...
                    </>
                  ) : doneExpanded ? (
                    <>
                      <ChevronUp className="w-3 h-3" />
                      show less
                    </>
                  ) : (
                    <>
                      <ChevronDown className="w-3 h-3" />show all {doneJobs.length}
                    </>
                  )}
                </button>
              )}
            </>
          )}

          {/* Offline section */}
          <OfflineSection items={offlineItems} onSync={handleRefresh} />

          {/* Refresh button */}
          <div className="flex justify-center py-2">
            <button
              onClick={handleRefresh}
              disabled={loading}
              className="flex items-center gap-2 text-xs text-[rgba(245,245,245,0.3)] hover:text-[rgba(245,245,245,0.5)] transition-colors"
            >
              <RefreshCw className={cn("w-3 h-3", loading && "animate-spin")} />
              refresh
            </button>
          </div>
        </>
      )}
    </div>
  );
}
