import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { ApiException, apiFetch } from "../../lib/api";
import type { Job, ProvidersResponse } from "../../types";
import { CloseIcon } from "../../components/icons";
import { useJobPoll } from "./useJobPoll";
import {
  evidenceLabel,
  parseResultString,
  type InvestigationResult,
  type ResultHypothesis,
} from "./result";
import {
  formatRelativeTime,
  ProviderPicker,
} from "../issues/RepositoryIssues";

// ─── small presentational pieces ─────────────────────────────────────────────

function ReadinessBadge({ readiness }: { readiness: "READY" | "NEEDS_INFORMATION" }) {
  if (readiness === "READY") {
    return (
      <span className="inline-flex items-center gap-1.5 rounded border border-[#1f3a2c] bg-[#4edea3]/10 px-2 py-0.5 font-mono text-[11px] text-[#4edea3]">
        <span className="size-1.5 rounded-full bg-[#4edea3]" />
        Ready
      </span>
    );
  }
  return (
    <span className="inline-flex items-center gap-1.5 rounded border border-[#e0a84e]/20 bg-[#e0a84e]/10 px-2 py-0.5 font-mono text-[11px] text-[#e0a84e]">
      <span className="size-1.5 rounded-full bg-[#e0a84e]" />
      Needs Information
    </span>
  );
}

function SectionTitle({ children, aside }: { children: React.ReactNode; aside?: string }) {
  return (
    <div className="flex items-center justify-between">
      <h2 className="text-sm font-semibold tracking-tight text-foreground">{children}</h2>
      {aside && <span className="font-mono text-xs text-muted">{aside}</span>}
    </div>
  );
}

function EvidencePill({
  evidenceRef,
  rank,
  onSelect,
}: {
  evidenceRef: string;
  rank: number;
  onSelect: (ref: string, rank: number) => void;
}) {
  const info = evidenceLabel(evidenceRef);
  const label =
    info.kind === "issue"
      ? "Issue body"
      : info.kind === "comment"
        ? `Comment #${info.id}`
        : evidenceRef;
  return (
    <button
      type="button"
      onClick={() => onSelect(evidenceRef, rank)}
      className="flex items-center gap-1 rounded border border-accent/20 bg-accent/10 px-2 py-1 font-mono text-xs text-accent transition-colors hover:border-accent/40 hover:text-foreground"
    >
      <span>{label}</span>
    </button>
  );
}

// ─── history ─────────────────────────────────────────────────────────────────

interface HistoryRun {
  id: string;
  status: string;
  model?: string | null;
  provider_type?: string | null;
  result?: string | null;
  created_at: string;
  updated_at: string;
}

function runBadge(run: HistoryRun): React.ReactNode {
  if (run.status === "failed") {
    return (
      <span className="inline-flex items-center rounded border border-[#e06c75]/20 bg-[#e06c75]/10 px-1.5 py-0.5 font-mono text-[10px] leading-none text-[#e06c75]">
        JOB_FAILED
      </span>
    );
  }
  if (run.status === "done") {
    const parsed = parseResultString(run.result ?? null);
    if (parsed.ok) {
      const color =
        parsed.result.readiness === "READY"
          ? "border-[#4edea3]/20 bg-[#4edea3]/10 text-[#4edea3]"
          : "border-[#e0a84e]/20 bg-[#e0a84e]/10 text-[#e0a84e]";
      return (
        <span className={`inline-flex items-center rounded border px-1.5 py-0.5 font-mono text-[10px] leading-none ${color}`}>
          {parsed.result.readiness}
        </span>
      );
    }
    return <span className="font-mono text-[11px] text-muted">Done</span>;
  }
  return <span className="font-mono text-[11px] text-muted">{run.status}</span>;
}

function HistorySection({
  repoId,
  issueNumber,
  currentId,
}: {
  repoId: string;
  issueNumber: number;
  currentId: string;
}) {
  const navigate = useNavigate();
  const histQuery = useQuery({
    queryKey: ["issue-investigations", repoId, issueNumber],
    queryFn: () =>
      apiFetch<{ investigations: HistoryRun[] }>(
        `/api/repos/${repoId}/issues/${issueNumber}/investigations`
      ),
    retry: false,
  });
  const runs = histQuery.data?.investigations ?? [];

  return (
    <section className="space-y-3 border-t border-[#222222] pt-6">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-2">
          <h2 className="text-sm font-semibold tracking-tight text-foreground">
            Investigation history
          </h2>
          {histQuery.isSuccess && (
            <span className="font-mono text-xs text-muted">
              {runs.length} run{runs.length === 1 ? "" : "s"} for #{issueNumber}
            </span>
          )}
        </div>
        <span className="font-mono text-[11px] text-muted">Newest first</span>
      </div>
      {histQuery.isLoading && <p className="text-[12px] text-muted">Loading history…</p>}
      {histQuery.isError && (
        <p className="text-[12px] text-muted">Could not load run history.</p>
      )}
      {histQuery.isSuccess && runs.length === 0 && (
        <p className="text-[12px] text-muted">No other runs recorded for this issue.</p>
      )}
      {runs.length > 0 && (
        <div className="divide-y divide-[#222222] rounded border border-[#222222] bg-surface font-mono text-xs">
          {runs.map((run) => {
            const isCurrent = run.id === currentId;
            return (
              <div
                key={run.id}
                className="flex items-center justify-between gap-3 p-3 transition-colors hover:bg-raised"
              >
                <div className="flex min-w-0 items-center gap-2.5">
                  <span className="shrink-0 text-muted">{formatRelativeTime(run.updated_at)}</span>
                  {runBadge(run)}
                  {(run.provider_type || run.model) && (
                    <span className="truncate text-[11px] text-muted">
                      {[run.provider_type, run.model].filter(Boolean).join(" · ")}
                    </span>
                  )}
                </div>
                {isCurrent ? (
                  <span className="shrink-0 rounded border border-accent/20 bg-accent/10 px-2 py-0.5 font-mono text-[11px] text-accent">
                    Current run
                  </span>
                ) : (
                  <button
                    type="button"
                    onClick={() => navigate(`/investigations/${run.id}`)}
                    className="shrink-0 rounded border border-transparent px-2 py-0.5 text-[11px] text-muted transition-colors hover:border-[#222222] hover:bg-surface hover:text-foreground"
                  >
                    View result →
                  </button>
                )}
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}

// ─── request-information dialog ──────────────────────────────────────────────

function RequestInfoDialog({
  repoId,
  issueNumber,
  investigationId,
  onClose,
}: {
  repoId: string;
  issueNumber: number;
  investigationId: string;
  onClose: () => void;
}) {
  const [comment, setComment] = useState<string | null>(null);
  const [postedUrl, setPostedUrl] = useState<string | null>(null);

  const draftQuery = useQuery({
    queryKey: ["request-draft", investigationId],
    queryFn: () =>
      apiFetch<{ comment: string }>(
        `/api/repos/${repoId}/issues/${issueNumber}/investigations/${investigationId}/request-draft`
      ),
    retry: false,
  });

  const postMutation = useMutation({
    mutationFn: (body: string) =>
      apiFetch<{ id: number; html_url: string; body: string }>(
        `/api/repos/${repoId}/issues/${issueNumber}/investigations/${investigationId}/request-information`,
        { method: "POST", body: JSON.stringify({ comment: body }) }
      ),
    onSuccess: (data) => setPostedUrl(data.html_url),
  });

  const draft = comment ?? draftQuery.data?.comment ?? "";

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Request missing information"
      onClick={onClose}
    >
      <div
        className="w-full max-w-lg rounded border border-[#222222] bg-surface p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-[14px] font-medium text-foreground">Request missing information</h2>
        <p className="mt-0.5 font-mono text-[11px] text-muted">
          Posts as you on issue #{issueNumber}. Nothing is posted until you confirm.
        </p>
        {draftQuery.isLoading && <p className="mt-4 text-[12px] text-muted">Preparing draft…</p>}
        {draftQuery.isError && (
          <p role="alert" className="mt-4 text-[12px] text-[#e06c75]">
            {draftQuery.error instanceof ApiException
              ? draftQuery.error.message
              : "Could not prepare the draft."}
          </p>
        )}
        {draftQuery.isSuccess && !postedUrl && (
          <div className="mt-4 flex flex-col gap-4">
            <textarea
              value={draft}
              onChange={(e) => setComment(e.target.value)}
              rows={6}
              aria-label="Information request comment"
              className="w-full rounded border border-[#222222] bg-background p-3 font-mono text-[12px] leading-relaxed text-foreground outline-none transition-colors focus:border-accent"
            />
            {postMutation.isError && (
              <p role="alert" className="text-[12px] text-[#e06c75]">
                {postMutation.error instanceof ApiException
                  ? postMutation.error.message
                  : "Posting failed."}
              </p>
            )}
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={onClose}
                className="h-8 rounded border border-[#222222] bg-surface px-3 text-[12px] text-foreground transition-colors hover:border-subtle hover:bg-raised"
              >
                Cancel
              </button>
              <button
                type="button"
                disabled={!draft.trim() || postMutation.isPending}
                onClick={() => postMutation.mutate(draft)}
                className="h-8 rounded bg-accent px-3 text-[12px] font-medium text-white transition-colors hover:bg-accent-hover disabled:opacity-50"
              >
                {postMutation.isPending ? "Posting…" : "Post comment"}
              </button>
            </div>
          </div>
        )}
        {postedUrl && (
          <div className="mt-4 flex flex-col gap-4">
            <p className="text-[12px] text-[#4edea3]">Comment posted to the issue.</p>
            <div className="flex justify-end gap-2">
              <a
                href={postedUrl}
                target="_blank"
                rel="noreferrer"
                className="h-8 rounded border border-[#222222] bg-surface px-3 py-1.5 text-[12px] text-foreground transition-colors hover:border-subtle"
              >
                View on GitHub
              </a>
              <button
                type="button"
                onClick={onClose}
                className="h-8 rounded bg-accent px-3 text-[12px] font-medium text-white hover:bg-accent-hover"
              >
                Done
              </button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── re-investigate dialog ───────────────────────────────────────────────────

function ReinvestigateDialog({
  repoId,
  issueNumber,
  investigationId,
  onClose,
}: {
  repoId: string;
  issueNumber: number;
  investigationId: string;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const [configId, setConfigId] = useState("");
  const [model, setModel] = useState("");

  const providersQuery = useQuery({
    queryKey: ["providers"],
    queryFn: () => apiFetch<ProvidersResponse>("/api/providers"),
    retry: false,
  });
  const providers = providersQuery.data?.providers ?? [];
  const hasEligible = providers.some((p) => p.enabled && p.models.some((m) => m.enabled));

  const mutation = useMutation({
    mutationFn: () =>
      apiFetch<{ job: { id: string } }>(
        `/api/repos/${repoId}/issues/${issueNumber}/investigations/${investigationId}/reinvestigate`,
        {
          method: "POST",
          body: JSON.stringify({ provider_config_id: configId, model, issue_number: issueNumber }),
        }
      ),
    onSuccess: (data) => {
      onClose();
      navigate(`/investigations/${data.job.id}`);
    },
  });

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Re-investigate issue"
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm rounded border border-[#222222] bg-surface p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-[14px] font-medium text-foreground">Re-investigate</h2>
        <p className="mt-0.5 font-mono text-[11px] text-muted">
          Runs a new investigation on fresh evidence. The current run is kept.
        </p>
        {providersQuery.isLoading && <p className="mt-4 text-[12px] text-muted">Loading providers…</p>}
        {providersQuery.isError && (
          <p role="alert" className="mt-4 text-[12px] text-[#e06c75]">Failed to load providers.</p>
        )}
        {providersQuery.isSuccess && !hasEligible && (
          <p role="alert" className="mt-4 text-[12px] text-muted">
            No enabled providers configured. Contact your org admin.
          </p>
        )}
        {providersQuery.isSuccess && hasEligible && (
          <form
            onSubmit={(e) => {
              e.preventDefault();
              if (configId && model && !mutation.isPending) mutation.mutate();
            }}
            className="mt-4 flex flex-col gap-4"
          >
            <ProviderPicker
              providers={providers}
              selectedConfigId={configId}
              selectedModel={model}
              onConfigChange={setConfigId}
              onModelChange={setModel}
            />
            {mutation.isError && (
              <p role="alert" className="text-[12px] text-[#e06c75]">
                {mutation.error instanceof ApiException
                  ? mutation.error.message
                  : "Re-investigation failed."}
              </p>
            )}
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={onClose}
                className="h-8 rounded border border-[#222222] bg-surface px-3 text-[12px] text-foreground transition-colors hover:border-subtle hover:bg-raised"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={!configId || !model || mutation.isPending}
                className="h-8 rounded bg-accent px-3 text-[12px] font-medium text-white transition-colors hover:bg-accent-hover disabled:opacity-50"
              >
                {mutation.isPending ? "Starting…" : "Start"}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

// ─── hypothesis card ─────────────────────────────────────────────────────────

function HypothesisCard({
  hypothesis,
  rank,
  onEvidence,
}: {
  hypothesis: ResultHypothesis;
  rank: number;
  onEvidence: (ref: string, rank: number) => void;
}) {
  return (
    <div className="space-y-3 rounded border border-[#222222] bg-surface p-4">
      <span className="font-mono text-[11px] font-medium uppercase tracking-wider text-muted">
        Rank {rank}
      </span>
      <h3 className="text-xs font-medium leading-relaxed text-foreground">
        {hypothesis.statement}
      </h3>
      <div className="flex flex-wrap items-center gap-2">
        <span className="rounded border border-[#222222] bg-raised px-1.5 py-0.5 font-mono text-[11px] text-muted">
          {hypothesis.confidence}
        </span>
        {hypothesis.evidence.map((ref) => (
          <EvidencePill key={ref} evidenceRef={ref} rank={rank} onSelect={onEvidence} />
        ))}
      </div>
      <details className="pt-1 text-xs">
        <summary className="flex cursor-pointer select-none items-center gap-1 font-mono text-[11px] text-muted transition-colors hover:text-foreground">
          <span>Reasoning summary</span>
        </summary>
        <p className="mt-2 border-l border-[#222222] py-1 pl-4 text-xs leading-relaxed text-muted">
          {hypothesis.reasoning}
        </p>
      </details>
    </div>
  );
}

// ─── main page ───────────────────────────────────────────────────────────────

export default function InvestigationDetail() {
  const { investigationId } = useParams<{ investigationId: string }>();
  const jobQuery = useJobPoll(investigationId ?? null);
  const job = jobQuery.data?.job as Job | undefined;
  const repo = jobQuery.data?.repository ?? null;

  const [evidenceOverride, setEvidenceOverride] = useState<
    { ref: string; rank: number } | "closed" | null
  >(null);
  const [requestOpen, setRequestOpen] = useState(false);
  const [reinvestigateOpen, setReinvestigateOpen] = useState(false);

  const parsed = job && job.status === "done" ? parseResultString(job.result) : null;
  const result: InvestigationResult | null = parsed && parsed.ok ? parsed.result : null;

  const canAct =
    job?.status === "done" &&
    result?.readiness === "NEEDS_INFORMATION" &&
    (result?.missing_information.length ?? 0) > 0;
  const canReinvestigate =
    job?.status === "done" && result?.readiness === "NEEDS_INFORMATION";

  const repoId = job?.repository_id ?? null;
  const issueNumber = job?.issue_number ?? 0;
  const githubIssueUrl =
    repo?.html_url && issueNumber > 0 ? `${repo.html_url}/issues/${issueNumber}` : null;
  const backTarget = repoId ? `/repositories/${repoId}` : "/repositories";

  const evidenceRefs: { ref: string; rank: number }[] = [];
  (result?.hypotheses ?? []).forEach((h, i) => {
    h.evidence.forEach((ref) => evidenceRefs.push({ ref, rank: i + 1 }));
  });
  // Desktop drawer auto-opens on the first evidence (Stitch default); the
  // small-screen overlay only opens on explicit pill selection, so narrow
  // screens are never covered on load. Either can be closed.
  const autoEvidence =
    evidenceRefs.length > 0 && evidenceOverride !== "closed"
      ? evidenceRefs[0]
      : null;
  const pickedEvidence =
    evidenceOverride !== null && evidenceOverride !== "closed" ? evidenceOverride : null;
  const asideEvidence = pickedEvidence ?? autoEvidence;

  return (
    <div className="flex min-w-0 flex-1 items-start gap-8">
      <main className="min-w-0 flex-1">
        <div className="flex max-w-3xl flex-col gap-8">
          <Link
            to={backTarget}
            className="inline-flex w-fit items-center gap-1.5 font-mono text-xs text-muted transition-colors hover:text-foreground"
          >
            <span aria-hidden="true">←</span>
            <span>{repo ? repo.full_name : "Repositories"}</span>
          </Link>

          {/* Loading / error states */}
          {jobQuery.isLoading && (
            <div className="flex flex-col gap-3" aria-busy="true" aria-label="Loading investigation">
              <div className="h-7 w-2/3 animate-pulse rounded bg-[#222222]" />
              <div className="h-4 w-full animate-pulse rounded bg-[#222222]" />
              <div className="h-40 animate-pulse rounded border border-[#222222] bg-surface" />
            </div>
          )}
          {jobQuery.isError && (
            <div role="alert" className="rounded border border-[#222222] bg-surface p-8 text-center">
              <p className="text-sm font-medium text-[#e06c75]">
                {jobQuery.error instanceof ApiException
                  ? jobQuery.error.message
                  : "Failed to load the investigation."}
              </p>
              <div className="mt-4 flex justify-center gap-2">
                <button
                  type="button"
                  onClick={() => void jobQuery.refetch()}
                  className="rounded border border-[#333333] bg-[#222222] px-3.5 py-1.5 font-mono text-xs font-medium text-[#ededed] transition-colors hover:bg-[#2a2a2a]"
                >
                  Retry
                </button>
                <Link
                  to={backTarget}
                  className="rounded border border-[#222222] px-3.5 py-1.5 font-mono text-xs text-muted transition-colors hover:text-foreground"
                >
                  Back to issues
                </Link>
              </div>
            </div>
          )}

          {jobQuery.isSuccess && job && (
            <>
              {/* Header */}
              <div className="space-y-4 border-b border-[#222222] pb-8">
                <div className="flex flex-wrap items-center gap-2.5">
                  <span className="rounded border border-[#222222] bg-surface px-1.5 py-0.5 font-mono text-xs font-medium text-muted">
                    #{issueNumber}
                  </span>
                  {job.status === "done" && result && (
                    <ReadinessBadge readiness={result.readiness} />
                  )}
                  {job.status === "failed" && (
                    <span className="inline-flex items-center gap-1.5 rounded border border-[#e06c75]/20 bg-[#e06c75]/10 px-2 py-0.5 font-mono text-[11px] text-[#e06c75]">
                      Failed
                    </span>
                  )}
                  {(job.status === "queued" || job.status === "running") && (
                    <span className="inline-flex items-center gap-1.5 rounded border border-[#222222] bg-surface px-2 py-0.5 font-mono text-[11px] text-muted">
                      <span className="size-1.5 animate-pulse rounded-full bg-accent" />
                      {job.status === "queued" ? "Queued" : "Running"}
                    </span>
                  )}
                  {job.provider_type === "mock" && (
                    <span
                      className="rounded border border-dashed border-[#555555] px-1.5 py-0.5 font-mono text-[10px] text-muted"
                      title="This result came from the mock provider and is development data, not a trustworthy analysis."
                    >
                      mock provider · dev data
                    </span>
                  )}
                </div>
                <h1 className="break-words text-[26px] font-semibold leading-[34px] tracking-tight text-foreground">
                  {job.input_title || `Issue #${issueNumber}`}
                </h1>
                <div className="flex flex-wrap items-center gap-2 font-mono text-xs text-muted">
                  <span>Investigated {formatRelativeTime(job.updated_at) || "recently"}</span>
                  {(job.provider_type || job.model) && (
                    <>
                      <span className="text-subtle">·</span>
                      <span>{[job.provider_type, job.model].filter(Boolean).join(" ")}</span>
                    </>
                  )}
                  {job.input_tokens != null && job.output_tokens != null && (
                    <>
                      <span className="text-subtle">·</span>
                      <span>
                        {job.input_tokens} input · {job.output_tokens} output tokens
                      </span>
                    </>
                  )}
                </div>
                <div className="flex flex-wrap items-center gap-2.5 pt-2">
                  {canAct && repoId && (
                    <button
                      type="button"
                      onClick={() => setRequestOpen(true)}
                      className="rounded bg-accent px-3.5 py-1.5 text-xs font-medium text-white transition-colors hover:bg-accent-hover"
                    >
                      Request missing information
                    </button>
                  )}
                  {canReinvestigate && repoId && (
                    <button
                      type="button"
                      onClick={() => setReinvestigateOpen(true)}
                      className="rounded border border-[#222222] bg-surface px-3 py-1.5 text-xs font-medium text-foreground transition-colors hover:border-subtle hover:bg-raised"
                    >
                      Re-investigate
                    </button>
                  )}
                  {githubIssueUrl && (
                    <a
                      href={githubIssueUrl}
                      target="_blank"
                      rel="noreferrer"
                      className="rounded px-3 py-1.5 text-xs font-medium text-muted transition-colors hover:text-foreground"
                    >
                      View on GitHub ↗
                    </a>
                  )}
                </div>
              </div>

              {/* In-progress */}
              {(job.status === "queued" || job.status === "running") && (
                <div
                  role="status"
                  className="flex items-center gap-3 rounded border border-[#222222] bg-surface px-4 py-3 text-xs text-muted"
                >
                  <span className="size-2 animate-pulse rounded-full bg-accent" />
                  Investigation {job.status}… this page updates automatically.
                </div>
              )}

              {/* Failed */}
              {job.status === "failed" && (
                <div role="alert" className="rounded border border-[#3c2222] bg-[#1f1616] p-5">
                  <p className="text-sm font-medium text-[#e06c75]">Investigation failed</p>
                  <p className="mt-1 text-xs text-muted">
                    {job.error ?? "No error detail was recorded."}
                  </p>
                  <Link
                    to={backTarget}
                    className="mt-4 inline-block rounded border border-[#222222] px-3 py-1.5 font-mono text-xs text-muted transition-colors hover:text-foreground"
                  >
                    Back to issues
                  </Link>
                </div>
              )}

              {/* Done but no/invalid result */}
              {job.status === "done" && !result && (
                <div role="alert" className="rounded border border-[#222222] bg-surface p-5">
                  <p className="text-sm font-medium text-foreground">
                    {job.result == null || job.result.trim() === ""
                      ? "No result was recorded for this run."
                      : "The stored result could not be understood."}
                  </p>
                  <p className="mt-1 font-mono text-[11px] text-muted">
                    {job.result == null || job.result.trim() === ""
                      ? "The job finished without producing output."
                      : `Reason: ${(parsed as { ok: false; reason: string }).reason}. Raw output is not shown.`}
                  </p>
                </div>
              )}

              {/* Done + parsed result */}
              {result && (
                <div className="flex flex-col gap-10">
                  <section className="space-y-3">
                    <SectionTitle>Summary</SectionTitle>
                    <p className="text-sm leading-relaxed text-foreground">{result.summary}</p>
                    <div className="flex flex-wrap gap-x-4 gap-y-1 font-mono text-[11px] text-muted">
                      <span>
                        Readiness:{" "}
                        <span
                          className={
                            result.readiness === "READY" ? "text-[#4edea3]" : "text-[#e0a84e]"
                          }
                        >
                          {result.readiness}
                        </span>
                      </span>
                      {result.missing_information.length > 0 && (
                        <span>{result.missing_information.length} required items</span>
                      )}
                    </div>
                  </section>

                  {result.missing_information.length > 0 && (
                    <section className="space-y-3">
                      <SectionTitle
                        aside={`${result.missing_information.length} required item${result.missing_information.length === 1 ? "" : "s"}`}
                      >
                        Missing information
                      </SectionTitle>
                      <ul className="divide-y divide-[#222222] rounded border border-[#222222] bg-surface">
                        {result.missing_information.map((item, i) => (
                          <li key={i} className="px-3.5 py-3 text-xs leading-relaxed text-foreground">
                            {item}
                          </li>
                        ))}
                      </ul>
                    </section>
                  )}

                  <section className="space-y-3">
                    <SectionTitle aside="Ranked potential causes">
                      Investigation hypotheses
                    </SectionTitle>
                    {result.hypotheses.length === 0 ? (
                      <p className="text-[12px] text-muted">
                        No hypotheses were recorded for this run.
                      </p>
                    ) : (
                      <div className="space-y-3">
                        {result.hypotheses.map((h, i) => (
                          <HypothesisCard
                            key={i}
                            hypothesis={h}
                            rank={i + 1}
                            onEvidence={(ref, rank) => setEvidenceOverride({ ref, rank })}
                          />
                        ))}
                      </div>
                    )}
                  </section>

                  {result.signals.length > 0 && (
                    <section className="border-t border-[#222222] pt-6">
                      <details>
                        <summary className="flex cursor-pointer list-none select-none items-center justify-between text-xs text-muted transition-colors hover:text-foreground">
                          <span>
                            <span className="font-medium text-foreground">Technical signals</span>{" "}
                            <span className="font-mono text-[11px]">
                              ({result.signals.length} detected)
                            </span>
                          </span>
                          <span className="font-mono text-[11px]">Toggle references</span>
                        </summary>
                        <ul className="mt-3 space-y-2 font-mono text-xs">
                          {result.signals.map((s, i) => (
                            <li
                              key={i}
                              className="rounded border border-[#222222] bg-surface p-2.5 text-foreground"
                            >
                              {s}
                            </li>
                          ))}
                        </ul>
                      </details>
                    </section>
                  )}

                  {(result.observed_behavior || result.expected_behavior || result.environment) && (
                    <section className="space-y-3 border-t border-[#222222] pt-6">
                      <SectionTitle>Observed context</SectionTitle>
                      <dl className="space-y-3 text-xs leading-relaxed">
                        {result.observed_behavior && (
                          <div>
                            <dt className="font-mono text-[11px] text-muted">Observed behavior</dt>
                            <dd className="mt-0.5 text-foreground">{result.observed_behavior}</dd>
                          </div>
                        )}
                        {result.expected_behavior && (
                          <div>
                            <dt className="font-mono text-[11px] text-muted">Expected behavior</dt>
                            <dd className="mt-0.5 text-foreground">{result.expected_behavior}</dd>
                          </div>
                        )}
                        {result.environment && (
                          <div>
                            <dt className="font-mono text-[11px] text-muted">Environment</dt>
                            <dd className="mt-0.5 font-mono text-[12px] text-foreground">
                              {result.environment}
                            </dd>
                          </div>
                        )}
                      </dl>
                    </section>
                  )}

                  {result.reproduction_steps.length > 0 && (
                    <section className="space-y-3 border-t border-[#222222] pt-6">
                      <SectionTitle>Reproduction steps</SectionTitle>
                      <ol className="list-decimal space-y-1.5 pl-5 text-xs leading-relaxed text-foreground">
                        {result.reproduction_steps.map((s, i) => (
                          <li key={i}>{s}</li>
                        ))}
                      </ol>
                    </section>
                  )}

                  {job.input_body && (
                    <section className="border-t border-[#222222] pt-6">
                      <details>
                        <summary className="cursor-pointer list-none select-none font-mono text-[11px] text-muted transition-colors hover:text-foreground">
                          Issue context captured for this run
                        </summary>
                        <p className="mt-1 font-mono text-[11px] text-muted">
                          Original issue title and body stored when this
                          investigation was queued — a snapshot, not the live
                          issue.
                        </p>
                        <pre className="mt-3 max-h-80 overflow-y-auto whitespace-pre-wrap rounded border border-[#1a1a1a] bg-[#0c0c0c] p-4 font-mono text-[12px] leading-relaxed text-muted">
                          {job.input_body}
                        </pre>
                      </details>
                    </section>
                  )}

                  {repoId && issueNumber > 0 && (
                    <HistorySection
                      repoId={repoId}
                      issueNumber={issueNumber}
                      currentId={job.id}
                    />
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </main>

      {/* Evidence drawer */}
      {asideEvidence && result && (
        <aside
          aria-label="Evidence details"
          className="sticky top-20 hidden h-[calc(100vh-6rem)] w-[360px] shrink-0 flex-col overflow-hidden border-l border-[#222222] bg-[#111111] xl:flex"
        >
          <EvidencePanel
            evidenceRef={asideEvidence.ref}
            rank={asideEvidence.rank}
            githubIssueUrl={githubIssueUrl}
            onClose={() => setEvidenceOverride("closed")}
          />
        </aside>
      )}
      {pickedEvidence && result && (
        <div className="fixed inset-y-0 right-0 z-40 flex w-[360px] max-w-[90vw] flex-col border-l border-[#222222] bg-[#111111] xl:hidden">
          <EvidencePanel
            evidenceRef={pickedEvidence.ref}
            rank={pickedEvidence.rank}
            githubIssueUrl={githubIssueUrl}
            onClose={() => setEvidenceOverride("closed")}
          />
        </div>
      )}

      {requestOpen && repoId && (
        <RequestInfoDialog
          repoId={repoId}
          issueNumber={issueNumber}
          investigationId={job?.id ?? ""}
          onClose={() => setRequestOpen(false)}
        />
      )}
      {reinvestigateOpen && repoId && (
        <ReinvestigateDialog
          repoId={repoId}
          issueNumber={issueNumber}
          investigationId={job?.id ?? ""}
          onClose={() => setReinvestigateOpen(false)}
        />
      )}
    </div>
  );
}

function EvidencePanel({
  evidenceRef,
  rank,
  githubIssueUrl,
  onClose,
}: {
  evidenceRef: string;
  rank: number;
  githubIssueUrl: string | null;
  onClose: () => void;
}) {
  const info = evidenceLabel(evidenceRef);
  return (
    <div className="flex h-full min-h-0 flex-col px-6 py-6">
      <div className="flex items-center justify-between border-b border-[#222222] pb-4">
        <h3 className="font-mono text-xs font-semibold uppercase tracking-wider text-foreground">
          Evidence · {info.kind === "unknown" ? "reference" : info.kind}
        </h3>
        <button
          type="button"
          onClick={onClose}
          aria-label="Close evidence panel"
          className="flex h-8 w-8 items-center justify-center rounded bg-foreground text-[14px] text-background transition-opacity hover:opacity-90 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-accent"
        >
          <CloseIcon />
        </button>
      </div>
      <div className="min-h-0 flex-1 space-y-6 overflow-y-auto py-6">
        <div className="space-y-1.5 font-mono text-xs text-muted">
          <div className="break-all text-foreground">{evidenceRef}</div>
          <div className="text-[11px]">
            {info.kind === "issue" && "The issue body supplied to this run."}
            {info.kind === "comment" && `Comment #${info.id} supplied to this run.`}
            {info.kind === "unknown" &&
              "Unrecognized reference format — shown verbatim, not resolved."}
          </div>
          {githubIssueUrl && info.kind === "issue" && (
            <div>
              <a
                href={githubIssueUrl}
                target="_blank"
                rel="noreferrer"
                className="text-[11px] text-accent hover:underline"
              >
                Open on GitHub ↗
              </a>
            </div>
          )}
        </div>
        <p className="font-mono text-[11px] leading-relaxed text-muted">
          Excerpt text is not stored with the run — only this reference. Linked
          to hypothesis #{rank}.
        </p>
      </div>
      <div className="border-t border-[#222222] pt-4 font-mono text-[11px] text-muted">
        Linked to Hypothesis #{rank}
      </div>
    </div>
  );
}
