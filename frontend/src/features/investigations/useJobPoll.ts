import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "../../lib/api";
import type { JobResponse } from "../../types";

const POLL_INTERVAL_MS = 3000;

/**
 * Shared job poller: GET /api/jobs/:id, refetching every 3s until the job
 * reaches a terminal state (done/failed). Query key ["job", jobId] is shared
 * so the detail page, brief modal, and status banners observe one cache entry.
 */
export function useJobPoll(jobId: string | null) {
  return useQuery({
    queryKey: ["job", jobId],
    queryFn: () => apiFetch<JobResponse>(`/api/jobs/${jobId}`),
    enabled: Boolean(jobId),
    refetchInterval: (query) => {
      const status = query.state.data?.job?.status;
      if (status === "done" || status === "failed") return false;
      return POLL_INTERVAL_MS;
    },
    retry: false,
  });
}
