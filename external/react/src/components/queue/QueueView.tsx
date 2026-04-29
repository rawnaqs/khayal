import { useState, useEffect, useCallback } from "react";
import { motion } from "framer-motion";
import { RefreshCw, FileText, Link, Image } from "lucide-react";
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

export function QueueView() {
  const { loading, jobs, fetchQueue, retryJob, discardJob } = useQueue();
  const { toast } = useToast();
  const [offlineItems, setOfflineItems] = useState<
    Array<{ id: string; content: string; timestamp: number }>
  >([]);

  const handleRefresh = useCallback(() => {
    fetchQueue();
    getOfflineQueue().then((items) => {
      setOfflineItems(
        items.map((i) => ({
          id: i.id,
          content: i.request.content,
          timestamp: i.timestamp,
        })),
      );
    });
  }, [fetchQueue]);

  useEffect(() => {
    handleRefresh();
  }, [handleRefresh]);

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

  // Derive job groups
  const processingJob = jobs.find((j) => j.status === "processing");
  const pendingJobs = jobs.filter(
    (j) => j.status === "pending" || j.status === "queued",
  );
  const failedJobs = jobs.filter((j) => j.status === "failed");
  const doneJobs = jobs.filter((j) => j.status === "done");

  return (
    <div className="q-body">
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
            {doneJobs.slice(0, LIMITS.DONE_JOBS_SHOWN).map((job, index) => (
              <motion.div
                key={job.id}
                initial={{ opacity: 0, y: 4 }}
                animate={{ opacity: 1, y: 0 }}
                transition={{ delay: index * 0.02 }}
              >
                <DoneItem job={job} />
              </motion.div>
            ))}
          </div>
          {doneJobs.length > 5 && (
            <div className="done-expand">
              {doneJobs.length - 5} more · scroll to expand
            </div>
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
    </div>
  );
}
